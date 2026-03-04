// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"yaaicms/internal/models"
)

const (
	// TenantKey is the context key for the resolved tenant.
	TenantKey contextKey = "tenant"

	// tenantCachePrefix is the Valkey key prefix for cached tenant lookups.
	tenantCachePrefix = "tenant:"

	// domainCachePrefix is the Valkey key prefix for custom domain lookups.
	domainCachePrefix = "domain:"

	// canonicalCachePrefix is the Valkey key prefix for canonical host lookups.
	canonicalCachePrefix = "canonical:"

	// tenantCacheTTL is how long tenant lookups are cached in Valkey.
	tenantCacheTTL = 5 * time.Minute
)

// TenantFinder is the interface needed by the tenant middleware to look up
// tenants by subdomain. Satisfied by store.TenantStore.
type TenantFinder interface {
	FindBySubdomain(subdomain string) (*models.Tenant, error)
}

// DomainResolver is the interface for looking up tenants by custom domain
// and resolving canonical (primary) domains. Satisfied by store.TenantResolver.
type DomainResolver interface {
	FindByCustomDomain(domain string) (*models.Tenant, error)
	FindPrimaryDomain(tenantID uuid.UUID) (string, error)
}

// ResolveTenant extracts the tenant from the Host header using dual resolution:
//  1. If the host does NOT end with baseDomain, try custom domain lookup.
//  2. If the host ends with baseDomain, extract the subdomain (existing logic).
//
// Both paths use Valkey caching with separate key prefixes.
// The domains parameter may be nil to disable custom domain resolution.
func ResolveTenant(finder TenantFinder, domains DomainResolver, cache *redis.Client, baseDomain string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Strip port from Host header.
			host := r.Host
			if idx := strings.LastIndex(host, ":"); idx != -1 {
				host = host[:idx]
			}

			var tenant *models.Tenant

			if !strings.HasSuffix(host, baseDomain) && domains != nil {
				// Host doesn't match base domain — try custom domain resolution.
				tenant = resolveByCustomDomain(r.Context(), domains, cache, host)
			} else {
				// Host matches base domain — use subdomain resolution.
				subdomain := extractSubdomain(r.Host, baseDomain)
				if subdomain == "" {
					subdomain = "default"
				}
				tenant = resolveBySubdomain(r.Context(), finder, cache, subdomain)

				// Fallback: if no tenant found by subdomain, try custom domain.
				// This handles cases like www.basedomain mapped as a custom domain.
				if tenant == nil && domains != nil {
					tenant = resolveByCustomDomain(r.Context(), domains, cache, host)
				}
			}

			if tenant == nil {
				http.NotFound(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), TenantKey, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CanonicalRedirect ensures each tenant is served from a single canonical host
// for SEO. If a primary domain is set, all other hosts (including the subdomain)
// redirect to it. If no primary is set, custom domain requests redirect to the
// tenant's subdomain. Must run after ResolveTenant.
func CanonicalRedirect(domains DomainResolver, cache *redis.Client, baseDomain string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenant := TenantFromCtx(r.Context())
			if tenant == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Strip port from Host header.
			host := r.Host
			if idx := strings.LastIndex(host, ":"); idx != -1 {
				host = host[:idx]
			}

			canonical := resolveCanonicalHost(r.Context(), domains, cache, tenant, baseDomain)

			if host != canonical {
				target := "https://" + canonical + r.URL.RequestURI()
				http.Redirect(w, r, target, http.StatusMovedPermanently)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// resolveCanonicalHost returns the canonical host for a tenant, using Valkey cache.
// Priority: primary domain > subdomain.baseDomain fallback.
func resolveCanonicalHost(ctx context.Context, domains DomainResolver, cache *redis.Client, tenant *models.Tenant, baseDomain string) string {
	cacheKey := canonicalCachePrefix + tenant.ID.String()

	// Try cache first.
	if cache != nil {
		if val, err := cache.Get(ctx, cacheKey).Result(); err == nil && val != "" {
			return val
		}
	}

	// Cache miss — look up primary domain.
	canonical := tenant.Subdomain + "." + baseDomain
	if domains != nil {
		if primary, err := domains.FindPrimaryDomain(tenant.ID); err == nil && primary != "" {
			canonical = primary
		}
	}

	// Cache the result.
	if cache != nil {
		cache.Set(ctx, cacheKey, canonical, tenantCacheTTL)
	}

	return canonical
}

// resolveBySubdomain looks up a tenant by subdomain with caching.
func resolveBySubdomain(ctx context.Context, finder TenantFinder, cache *redis.Client, subdomain string) *models.Tenant {
	// Try Valkey cache first.
	tenant := getCachedTenant(ctx, cache, tenantCachePrefix+subdomain)
	if tenant != nil {
		return tenant
	}

	// Cache miss — hit the database.
	tenant, err := finder.FindBySubdomain(subdomain)
	if err != nil {
		slog.Error("tenant lookup failed", "subdomain", subdomain, "error", err)
		return nil
	}
	if tenant == nil || !tenant.IsActive {
		return nil
	}

	// Cache the result.
	cacheTenant(ctx, cache, tenantCachePrefix+subdomain, tenant)
	return tenant
}

// resolveByCustomDomain looks up a tenant by custom domain with caching.
func resolveByCustomDomain(ctx context.Context, domains DomainResolver, cache *redis.Client, host string) *models.Tenant {
	// Try Valkey cache first.
	tenant := getCachedTenant(ctx, cache, domainCachePrefix+host)
	if tenant != nil {
		return tenant
	}

	// Cache miss — hit the database.
	tenant, err := domains.FindByCustomDomain(host)
	if err != nil {
		slog.Error("custom domain lookup failed", "domain", host, "error", err)
		return nil
	}
	if tenant == nil {
		return nil
	}

	// Cache the result.
	cacheTenant(ctx, cache, domainCachePrefix+host, tenant)
	return tenant
}

// TenantFromCtx extracts the tenant from the request context.
// Returns nil if no tenant was resolved (e.g., health check routes).
func TenantFromCtx(ctx context.Context) *models.Tenant {
	t, _ := ctx.Value(TenantKey).(*models.Tenant)
	return t
}

// extractSubdomain extracts the subdomain from a host string given a base domain.
// For "blog1.smartpress.io" with baseDomain "smartpress.io", returns "blog1".
// For "localhost:8080" with baseDomain "localhost", returns "".
func extractSubdomain(host, baseDomain string) string {
	// Strip port if present.
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Strip the base domain suffix.
	if !strings.HasSuffix(host, baseDomain) {
		return ""
	}

	prefix := strings.TrimSuffix(host, baseDomain)
	prefix = strings.TrimSuffix(prefix, ".")

	// No subdomain — bare domain.
	if prefix == "" {
		return ""
	}

	// Return the last component (handles nested subdomains).
	parts := strings.Split(prefix, ".")
	return parts[len(parts)-1]
}

// getCachedTenant retrieves a tenant from Valkey cache by its full cache key.
// Returns nil on cache miss or error.
func getCachedTenant(ctx context.Context, cache *redis.Client, cacheKey string) *models.Tenant {
	if cache == nil {
		return nil
	}

	// We store a simple serialized format: "id|name|subdomain|active"
	val, err := cache.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil
	}

	parts := strings.SplitN(val, "|", 4)
	if len(parts) != 4 {
		return nil
	}

	id, err := uuid.Parse(parts[0])
	if err != nil {
		return nil
	}

	isActive := parts[3] == "1"
	if !isActive {
		return nil
	}

	return &models.Tenant{
		ID:        id,
		Name:      parts[1],
		Subdomain: parts[2],
		IsActive:  isActive,
	}
}

// cacheTenant stores a tenant in the Valkey cache under the given key.
func cacheTenant(ctx context.Context, cache *redis.Client, cacheKey string, t *models.Tenant) {
	if cache == nil {
		return
	}

	active := "0"
	if t.IsActive {
		active = "1"
	}

	val := fmt.Sprintf("%s|%s|%s|%s", t.ID.String(), t.Name, t.Subdomain, active)
	cache.Set(ctx, cacheKey, val, tenantCacheTTL)
}
