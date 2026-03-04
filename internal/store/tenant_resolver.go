// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package store

import "yaaicms/internal/models"

// TenantResolver wraps TenantStore and TenantDomainStore to provide both
// subdomain and custom domain resolution. It satisfies both the TenantFinder
// and DomainResolver interfaces used by the tenant middleware.
type TenantResolver struct {
	tenants *TenantStore
	domains *TenantDomainStore
}

// NewTenantResolver creates a resolver that can look up tenants by subdomain
// or by custom domain.
func NewTenantResolver(tenants *TenantStore, domains *TenantDomainStore) *TenantResolver {
	return &TenantResolver{tenants: tenants, domains: domains}
}

// FindBySubdomain delegates to TenantStore.FindBySubdomain.
func (r *TenantResolver) FindBySubdomain(subdomain string) (*models.Tenant, error) {
	return r.tenants.FindBySubdomain(subdomain)
}

// FindByCustomDomain looks up a tenant by a custom domain that has been
// verified and is actively serving. Returns nil if not found or not active.
func (r *TenantResolver) FindByCustomDomain(domain string) (*models.Tenant, error) {
	_, tenant, err := r.domains.FindByDomainWithTenant(domain)
	if err != nil {
		return nil, err
	}
	return tenant, nil
}
