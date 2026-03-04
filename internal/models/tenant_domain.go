// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package models

import (
	"time"

	"github.com/google/uuid"
)

// Domain status constants track the verification lifecycle.
const (
	DomainStatusPending      = "pending_verification"
	DomainStatusVerified     = "verified"
	DomainStatusProvisioning = "provisioning"
	DomainStatusActive       = "active"
	DomainStatusFailed       = "failed"
)

// TenantDomain maps a custom external domain (e.g., "someblog.com") to a tenant.
// Domains go through a verification lifecycle before they can serve traffic.
type TenantDomain struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	Domain     string     `json:"domain"`
	Status     string     `json:"status"`
	IsPrimary  bool       `json:"is_primary"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
