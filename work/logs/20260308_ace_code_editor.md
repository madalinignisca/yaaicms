# Ace Code Editor for Template Editing

**Date:** 2026-03-08
**Branch:** feat/ace-code-editor
**Status:** Complete

## Summary

Replaced the plain `<textarea>` in the template form with Ace Editor — a full-featured code editor with syntax highlighting, autocompletion, bracket matching, search/replace, and theming.

## Changes

### `Dockerfile`
- Added Ace Editor vendor download stage: downloads `ace.js`, `ext-language_tools.js`, `ext-searchbox.js`, mode/worker files for HTML/CSS/JS, and Chrome + Monokai themes from jsDelivr CDN.
- Files placed in `web/static/js/ace/` for embedding into the Go binary.

### `internal/render/templates/admin/base.html`
- Added conditional Ace Editor script loading:
  - Dev mode: CDN (`cdn.jsdelivr.net/npm/ace-builds@1.36.5`)
  - Production: vendored local files (`/static/js/ace/`)
- Configured `ace.config.set('basePath', ...)` for worker file resolution.

### `internal/render/templates/admin/template_form.html`
- Replaced visible `<textarea>` with hidden textarea (for form submission) + Ace Editor div.
- Added `initAceEditor()` method: initializes Ace with HTML mode, Chrome theme, line wrap, autocompletion, soft tabs (2 spaces).
- Added `getEditorValue()` / `setEditorValue()` accessors — all read/write operations route through these (with textarea fallback if Ace fails to load).
- Added theme toggle (Chrome / Monokai) and wrap toggle buttons in the editor toolbar.
- Ace `change` event syncs content back to hidden textarea automatically.
- Updated `refreshFormPreview()` and `aiImprove()` to use the new accessors.

## Design Decisions

- **Ace over CodeMirror 6**: CodeMirror 6 requires a bundler (ES modules); Ace provides pre-built files that work with the project's no-bundler static asset pattern.
- **Ace over Monaco**: Monaco is 10MB+ and designed for VS Code — too heavy for an embedded CMS editor.
- **Hidden textarea pattern**: HTML forms only submit native form elements. Keeping a synced hidden textarea is the standard integration approach.
- **Graceful fallback**: If Ace fails to load, `getEditorValue()`/`setEditorValue()` fall back to the textarea directly.
