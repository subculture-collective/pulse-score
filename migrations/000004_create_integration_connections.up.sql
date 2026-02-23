-- Create integration_connections table
CREATE TABLE integration_connections (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id                  UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    provider                VARCHAR(50) NOT NULL,
    status                  VARCHAR(20) NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'active', 'error', 'disconnected')),
    access_token_encrypted  BYTEA,
    refresh_token_encrypted BYTEA,
    token_expires_at        TIMESTAMPTZ,
    external_account_id     VARCHAR(255),
    scopes                  TEXT[],
    metadata                JSONB NOT NULL DEFAULT '{}',
    last_sync_at            TIMESTAMPTZ,
    last_sync_error         TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (org_id, provider)
);

CREATE INDEX idx_integration_connections_provider_status ON integration_connections (provider, status);

CREATE TRIGGER set_integration_connections_updated_at
    BEFORE UPDATE ON integration_connections
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
