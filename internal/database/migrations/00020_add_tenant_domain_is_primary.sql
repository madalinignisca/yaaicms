-- +goose Up
ALTER TABLE tenant_domains ADD COLUMN is_primary BOOLEAN NOT NULL DEFAULT false;

-- Only one primary domain per tenant (partial unique index).
CREATE UNIQUE INDEX idx_tenant_domains_primary
    ON tenant_domains (tenant_id) WHERE is_primary = true;

-- +goose Down
DROP INDEX IF EXISTS idx_tenant_domains_primary;
ALTER TABLE tenant_domains DROP COLUMN IF EXISTS is_primary;
