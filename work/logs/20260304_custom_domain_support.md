# Custom Domain Support Implementation

**Date:** 2026-03-04
**Branch:** feat/multi-tenancy

## What was done

### Database Layer
- Migration `00019_create_tenant_domains.sql` — `tenant_domains` table with status lifecycle
- Model `models.TenantDomain` — struct with status constants
- Store `store.TenantDomainStore` — full CRUD (FindByDomain, FindByDomainWithTenant, ListByTenant, ListByStatus, Create, UpdateStatus, SetVerified, Delete, FindByID)
- Adapter `store.TenantResolver` — wraps TenantStore + TenantDomainStore, satisfies both `TenantFinder` and `DomainResolver` interfaces

### Middleware
- Added `DomainResolver` interface to `middleware` package
- Updated `ResolveTenant` to accept `DomainResolver` for dual resolution (subdomain + custom domain)
- Resolution order: if Host doesn't end with baseDomain → custom domain lookup; otherwise → subdomain lookup
- Separate cache key prefixes (`tenant:` vs `domain:`) for the two resolution paths

### K8s Integration
- New `internal/k8s/` package
- `Manager` — creates/deletes cert-manager Certificate CRs and Traefik IngressRoute CRs via raw K8s REST API (no client-go dependency)
- `Verifier` — background goroutine (60s interval) that processes pending/provisioning domains: DNS verification → K8s resource creation → certificate readiness check

### Admin UI
- Template `tenant_domains.html` — domain list with status badges, add form, verify/delete actions
- Handlers: TenantDomains, TenantAddDomain, TenantDeleteDomain, TenantVerifyDomain
- Added "Domains" link to tenant list page

### Config & Wiring
- Added `K8sEnabled` and `K8sNamespace` config vars
- Updated `main.go` to initialize TenantDomainStore, TenantResolver, K8s Manager, and start Verifier goroutine

### K8s Manifests
- `cloudflare-issuer.yaml` — ClusterIssuer for DNS-01 wildcard certs
- `wildcard-cert.yaml` — Certificate for `*.madalin.me`
- `wildcard-ingressroute.yaml` — Traefik IngressRoute for wildcard subdomains
- `rbac.yaml` — ServiceAccount + Role + RoleBinding for custom domain management
- Updated deployment base with `serviceAccountName: yaaicms`
- Updated testing overlay with `BASE_DOMAIN`, `K8S_ENABLED`, `K8S_NAMESPACE` env vars

## Verification
- `go build ./...` — passes
- `go vet ./...` — passes
- `go test ./internal/k8s/` — 2 tests pass (sanitization + disabled noop)
- `go test ./internal/middleware/` — all 38 tests pass
