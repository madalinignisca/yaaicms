// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

// Package engine provides an in-memory cache for compiled Go templates.
// This is the L1 cache — it avoids re-parsing template strings on every
// request. Templates are keyed by their database ID and version, so an
// update or activation automatically produces a cache miss.
package engine

import (
	"html/template"
	"log/slog"
	"sync"
)

// cacheKey uniquely identifies a compiled template version.
// Using ID+Version means any update (which bumps version) auto-invalidates.
type cacheKey struct {
	id      string // UUID as string
	version int
}

// templateCache is a concurrency-safe in-memory cache of compiled templates.
type templateCache struct {
	mu      sync.RWMutex
	entries map[cacheKey]*template.Template
}

// newTemplateCache creates an empty template cache.
func newTemplateCache() *templateCache {
	return &templateCache{
		entries: make(map[cacheKey]*template.Template),
	}
}

// get retrieves a compiled template from cache. Returns nil on miss.
func (c *templateCache) get(id string, version int) *template.Template {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.entries[cacheKey{id: id, version: version}]
}

// put stores a compiled template in the cache.
func (c *templateCache) put(id string, version int, tmpl *template.Template) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[cacheKey{id: id, version: version}] = tmpl
	slog.Debug("template cached", "id", id, "version", version, "size", len(c.entries))
}

// invalidate removes all cached versions for a given template ID.
// Called when a template is updated, activated, or deleted.
func (c *templateCache) invalidate(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.entries {
		if k.id == id {
			delete(c.entries, k)
		}
	}
	slog.Debug("template cache invalidated", "id", id)
}

// invalidateAll clears the entire cache. Used when templates are activated
// (since activation changes which template is "active" for a type, all
// fragments like header/footer may need recompilation).
func (c *templateCache) invalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[cacheKey]*template.Template)
	slog.Debug("template cache fully cleared")
}
