-- Create customers table
CREATE TABLE customers (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id        UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    external_id   VARCHAR(255),
    source        VARCHAR(50) NOT NULL,
    email         CITEXT,
    name          VARCHAR(255),
    company_name  VARCHAR(255),
    mrr_cents     INTEGER NOT NULL DEFAULT 0,
    currency      VARCHAR(3) NOT NULL DEFAULT 'USD',
    first_seen_at TIMESTAMPTZ,
    last_seen_at  TIMESTAMPTZ,
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,

    UNIQUE (org_id, source, external_id)
);

CREATE INDEX idx_customers_org_id ON customers (org_id);
CREATE INDEX idx_customers_org_email ON customers (org_id, email);
CREATE INDEX idx_customers_org_source ON customers (org_id, source);
CREATE INDEX idx_customers_org_mrr ON customers (org_id, mrr_cents);

CREATE TRIGGER set_customers_updated_at
    BEFORE UPDATE ON customers
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
