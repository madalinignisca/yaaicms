-- Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
-- Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
-- All rights reserved. See LICENSE for details.

-- +goose Up
-- Custom domain mappings for tenants. Each domain goes through a verification
-- lifecycle before it can serve traffic: pending → verified → provisioning → active.

CREATE TABLE tenant_domains (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    domain      VARCHAR(255) NOT NULL,
    status      VARCHAR(30) NOT NULL DEFAULT 'pending_verification'
                CHECK (status IN ('pending_verification','verified','provisioning','active','failed')),
    verified_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_tenant_domains_domain ON tenant_domains(domain);
CREATE INDEX idx_tenant_domains_tenant_id ON tenant_domains(tenant_id);
CREATE INDEX idx_tenant_domains_status ON tenant_domains(status);

-- +goose Down
DROP TABLE IF EXISTS tenant_domains;
