// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package models

import "testing"

// TestContentIsPublished verifies that IsPublished returns true only for
// the "published" status.
func TestContentIsPublished(t *testing.T) {
	tests := []struct {
		name   string
		status ContentStatus
		want   bool
	}{
		{name: "published", status: ContentStatusPublished, want: true},
		{name: "draft", status: ContentStatusDraft, want: false},
		{name: "empty status", status: ContentStatus(""), want: false},
		{name: "unknown status", status: ContentStatus("archived"), want: false},
		{name: "uppercase PUBLISHED", status: ContentStatus("PUBLISHED"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Content{Status: tt.status}
			got := c.IsPublished()
			if got != tt.want {
				t.Errorf("Content{Status: %q}.IsPublished() = %v, want %v",
					tt.status, got, tt.want)
			}
		})
	}
}

// TestContentTypeConstants verifies that content type string constants have
// the expected values.
func TestContentTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		ct       ContentType
		expected string
	}{
		{name: "post type", ct: ContentTypePost, expected: "post"},
		{name: "page type", ct: ContentTypePage, expected: "page"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.ct) != tt.expected {
				t.Errorf("ContentType %s = %q, want %q", tt.name, string(tt.ct), tt.expected)
			}
		})
	}
}

// TestContentStatusConstants verifies that content status string constants
// have the expected values.
func TestContentStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		cs       ContentStatus
		expected string
	}{
		{name: "draft status", cs: ContentStatusDraft, expected: "draft"},
		{name: "published status", cs: ContentStatusPublished, expected: "published"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.cs) != tt.expected {
				t.Errorf("ContentStatus %s = %q, want %q", tt.name, string(tt.cs), tt.expected)
			}
		})
	}
}

// TestContentTypeDistinct ensures post and page types are different values.
func TestContentTypeDistinct(t *testing.T) {
	if ContentTypePost == ContentTypePage {
		t.Error("ContentTypePost and ContentTypePage must be distinct")
	}
}

// TestContentStatusDistinct ensures draft and published statuses are different.
func TestContentStatusDistinct(t *testing.T) {
	if ContentStatusDraft == ContentStatusPublished {
		t.Error("ContentStatusDraft and ContentStatusPublished must be distinct")
	}
}
