-- Create onboarding_events table for funnel analytics
CREATE TABLE IF NOT EXISTS onboarding_events (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id       UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    step_id      VARCHAR(50) NOT NULL,
    event_type   VARCHAR(50) NOT NULL
                 CHECK (event_type IN ('step_started', 'step_completed', 'step_skipped', 'onboarding_completed', 'onboarding_abandoned')),
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_ms  BIGINT,
    metadata     JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_onboarding_events_org_time ON onboarding_events (org_id, occurred_at DESC);
CREATE INDEX idx_onboarding_events_org_step_type ON onboarding_events (org_id, step_id, event_type);
