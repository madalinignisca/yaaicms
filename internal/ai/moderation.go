// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// ModerationResult contains the outcome of a prompt safety check.
type ModerationResult struct {
	Safe       bool     // true if the prompt passes moderation
	Categories []string // list of flagged category names (empty when safe)
}

// Moderator checks user prompts for policy violations before sending
// them to AI generation endpoints.
type Moderator interface {
	// CheckSafety evaluates a text prompt and returns whether it is safe
	// to send to an AI provider. If not safe, Categories lists the reasons.
	CheckSafety(ctx context.Context, text string) (*ModerationResult, error)
}

// --- OpenAI Moderation (free endpoint) ---

// openAIModerator uses the OpenAI Moderation API (POST /v1/moderations)
// which is free for all OpenAI API key holders.
type openAIModerator struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// newOpenAIModerator creates a moderator that uses OpenAI's free moderation API.
func newOpenAIModerator(apiKey, baseURL string) *openAIModerator {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &openAIModerator{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *openAIModerator) CheckSafety(ctx context.Context, text string) (*ModerationResult, error) {
	body := openAIModRequest{
		Model: "omni-moderation-latest",
		Input: text,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("moderation marshal: %w", err)
	}

	url := m.baseURL + "/moderations"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("moderation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("moderation http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("moderation read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("moderation API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result openAIModResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("moderation unmarshal: %w", err)
	}

	if len(result.Results) == 0 {
		return &ModerationResult{Safe: true}, nil
	}

	r := result.Results[0]
	if !r.Flagged {
		return &ModerationResult{Safe: true}, nil
	}

	// Collect flagged category names in human-readable form.
	var flagged []string
	for cat, isFlagged := range r.Categories {
		if isFlagged {
			// Convert "hate/threatening" → "hate (threatening)" for readability.
			display := strings.ReplaceAll(cat, "/", " (")
			if strings.Contains(cat, "/") {
				display += ")"
			}
			display = strings.ReplaceAll(display, "_", " ")
			flagged = append(flagged, display)
		}
	}

	return &ModerationResult{
		Safe:       false,
		Categories: flagged,
	}, nil
}

// --- Mistral Moderation (paid, fallback) ---

// mistralModerator uses the Mistral Moderation API (POST /v1/moderations).
type mistralModerator struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// newMistralModerator creates a moderator using Mistral's classification endpoint.
func newMistralModerator(apiKey, baseURL string) *mistralModerator {
	if baseURL == "" {
		baseURL = "https://api.mistral.ai"
	}
	return &mistralModerator{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *mistralModerator) CheckSafety(ctx context.Context, text string) (*ModerationResult, error) {
	body := mistralModRequest{
		Model: "mistral-moderation-latest",
		Input: text,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("mistral moderation marshal: %w", err)
	}

	url := m.baseURL + "/v1/moderations"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("mistral moderation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mistral moderation http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mistral moderation read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mistral moderation API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result mistralModResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("mistral moderation unmarshal: %w", err)
	}

	if len(result.Results) == 0 {
		return &ModerationResult{Safe: true}, nil
	}

	// Mistral doesn't have a top-level "flagged" — check each category.
	var flagged []string
	for cat, isFlagged := range result.Results[0].Categories {
		if isFlagged {
			display := strings.ReplaceAll(cat, "_", " ")
			flagged = append(flagged, display)
		}
	}

	return &ModerationResult{
		Safe:       len(flagged) == 0,
		Categories: flagged,
	}, nil
}

// --- Fallback Moderator ---

// fallbackModerator tries the primary moderator first; if it returns a
// persistent error (e.g. 403 from a project-scoped API key), it automatically
// switches to the secondary moderator for all future calls.
type fallbackModerator struct {
	primary   Moderator
	secondary Moderator
	usePrimary atomic.Bool
}

// newFallbackModerator creates a moderator that tries primary first and
// falls back to secondary on persistent errors.
func newFallbackModerator(primary, secondary Moderator) *fallbackModerator {
	m := &fallbackModerator{
		primary:   primary,
		secondary: secondary,
	}
	m.usePrimary.Store(true)
	return m
}

func (m *fallbackModerator) CheckSafety(ctx context.Context, text string) (*ModerationResult, error) {
	if m.usePrimary.Load() {
		result, err := m.primary.CheckSafety(ctx, text)
		if err == nil {
			return result, nil
		}
		// On auth/permission errors, permanently switch to secondary.
		if strings.Contains(err.Error(), "status 403") || strings.Contains(err.Error(), "status 401") {
			slog.Warn("primary moderator returned auth error, switching to fallback",
				"error", err)
			m.usePrimary.Store(false)
			return m.secondary.CheckSafety(ctx, text)
		}
		// For transient errors, try the secondary once but don't switch permanently.
		slog.Warn("primary moderator failed, trying fallback", "error", err)
		return m.secondary.CheckSafety(ctx, text)
	}
	return m.secondary.CheckSafety(ctx, text)
}

// --- Request/Response types ---

type openAIModRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openAIModResponse struct {
	Results []openAIModResult `json:"results"`
}

type openAIModResult struct {
	Flagged    bool            `json:"flagged"`
	Categories map[string]bool `json:"categories"`
}

type mistralModRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type mistralModResponse struct {
	Results []mistralModResult `json:"results"`
}

type mistralModResult struct {
	Categories map[string]bool `json:"categories"`
}
