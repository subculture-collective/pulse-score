-- Create onboarding_status table for per-organization wizard progress
CREATE TABLE IF NOT EXISTS onboarding_status (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    current_step    VARCHAR(50) NOT NULL DEFAULT 'welcome',
    completed_steps TEXT[] NOT NULL DEFAULT '{}',
    skipped_steps   TEXT[] NOT NULL DEFAULT '{}',
    step_payloads   JSONB NOT NULL DEFAULT '{}',
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (org_id)
);

CREATE INDEX idx_onboarding_status_org_id ON onboarding_status (org_id);
CREATE INDEX idx_onboarding_status_completed_at ON onboarding_status (completed_at);

CREATE TRIGGER set_onboarding_status_updated_at
    BEFORE UPDATE ON onboarding_status
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
