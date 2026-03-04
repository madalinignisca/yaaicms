// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package k8s

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"yaaicms/internal/models"
	"yaaicms/internal/store"
)

const (
	// verifyInterval is how often the verifier checks domain status.
	verifyInterval = 60 * time.Second

	// pendingTimeout is how long a domain can stay pending before being marked failed.
	pendingTimeout = 24 * time.Hour
)

// Verifier periodically checks pending and provisioning domains, advancing
// them through the verification lifecycle.
type Verifier struct {
	domainStore *store.TenantDomainStore
	manager     *Manager
	cache       *redis.Client
	baseDomain  string
}

// NewVerifier creates a new domain verifier.
func NewVerifier(domainStore *store.TenantDomainStore, manager *Manager, cache *redis.Client, baseDomain string) *Verifier {
	return &Verifier{
		domainStore: domainStore,
		manager:     manager,
		cache:       cache,
		baseDomain:  baseDomain,
	}
}

// Run starts the verification loop. Blocks until ctx is cancelled.
func (v *Verifier) Run(ctx context.Context) {
	slog.Info("domain verifier started", "interval", verifyInterval)

	ticker := time.NewTicker(verifyInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("domain verifier stopped")
			return
		case <-ticker.C:
			v.processPending()
			v.processProvisioning()
		}
	}
}

// processPending checks domains waiting for DNS verification.
// If the domain's CNAME or A record resolves to the base domain, it is marked
// verified and K8s resources are created.
func (v *Verifier) processPending() {
	domains, err := v.domainStore.ListByStatus(models.DomainStatusPending)
	if err != nil {
		slog.Error("verifier: failed to list pending domains", "error", err)
		return
	}

	for _, d := range domains {
		// Check if domain has been pending too long.
		if time.Since(d.CreatedAt) > pendingTimeout {
			slog.Warn("domain pending too long, marking failed", "domain", d.Domain)
			_ = v.domainStore.UpdateStatus(d.ID, models.DomainStatusFailed)
			continue
		}

		if v.verifyDNS(d.Domain) {
			slog.Info("domain DNS verified", "domain", d.Domain)

			if err := v.domainStore.SetVerified(d.ID); err != nil {
				slog.Error("verifier: failed to set verified", "domain", d.Domain, "error", err)
				continue
			}

			// Create K8s resources (Certificate + IngressRoute).
			if err := v.manager.CreateDomainResources(d.Domain); err != nil {
				slog.Error("verifier: failed to create k8s resources", "domain", d.Domain, "error", err)
				_ = v.domainStore.UpdateStatus(d.ID, models.DomainStatusFailed)
				continue
			}

			_ = v.domainStore.UpdateStatus(d.ID, models.DomainStatusProvisioning)
		}
	}
}

// processProvisioning checks domains waiting for TLS certificate issuance.
func (v *Verifier) processProvisioning() {
	domains, err := v.domainStore.ListByStatus(models.DomainStatusProvisioning)
	if err != nil {
		slog.Error("verifier: failed to list provisioning domains", "error", err)
		return
	}

	for _, d := range domains {
		if v.manager.CertificateReady(d.Domain) {
			slog.Info("domain certificate ready, marking active", "domain", d.Domain)
			_ = v.domainStore.UpdateStatus(d.ID, models.DomainStatusActive)

			// Invalidate any negative cache entry for this domain.
			if v.cache != nil {
				v.cache.Del(context.Background(), "domain:"+d.Domain)
			}
		}
	}
}

// verifyDNS checks if a domain's DNS records resolve to the base domain.
// Accepts CNAME pointing to baseDomain or A/AAAA records matching the base domain's IPs.
func (v *Verifier) verifyDNS(domain string) bool {
	// Check CNAME first.
	cname, err := net.LookupCNAME(domain)
	if err == nil {
		cname = strings.TrimSuffix(cname, ".")
		if strings.HasSuffix(cname, v.baseDomain) || cname == v.baseDomain {
			return true
		}
	}

	// Fall back to A/AAAA record comparison.
	domainIPs, err := net.LookupHost(domain)
	if err != nil {
		return false
	}

	baseIPs, err := net.LookupHost(v.baseDomain)
	if err != nil {
		return false
	}

	baseIPSet := make(map[string]bool, len(baseIPs))
	for _, ip := range baseIPs {
		baseIPSet[ip] = true
	}

	for _, ip := range domainIPs {
		if baseIPSet[ip] {
			return true
		}
	}

	return false
}
