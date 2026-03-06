// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

// Package markdown converts Markdown source text into HTML using goldmark.
// It enables unsafe HTML pass-through so that existing raw-HTML content
// (created before the Markdown editor was introduced) renders correctly.
package markdown

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

// md is the configured goldmark instance, reused across calls.
var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,            // GitHub-Flavored Markdown: tables, strikethrough, autolinks, task lists
		extension.Typographer,    // Smart quotes and dashes
		highlighting.NewHighlighting( // Syntax highlighting for fenced code blocks
			highlighting.WithStyle("monokai"),
			highlighting.WithFormatOptions(),
		),
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(), // Auto-generate heading IDs for anchors
	),
	goldmark.WithRendererOptions(
		html.WithUnsafe(), // Allow raw HTML blocks — needed for backward compat with existing HTML content
	),
)

// ToHTML converts Markdown source into HTML. Raw HTML embedded in the
// Markdown is passed through unchanged (WithUnsafe), which ensures
// backward compatibility with content created before the Markdown editor.
func ToHTML(source string) (string, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
