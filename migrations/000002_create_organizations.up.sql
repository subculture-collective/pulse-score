-- Create organizations table (tenant root)
CREATE TABLE organizations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    slug            CITEXT UNIQUE NOT NULL,
    plan            VARCHAR(50) NOT NULL DEFAULT 'free',
    stripe_customer_id VARCHAR(255),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_organizations_stripe_customer_id ON organizations (stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;

-- Auto-update updated_at trigger function (reusable)
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
