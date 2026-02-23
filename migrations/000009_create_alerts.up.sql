-- Create alert_rules table
CREATE TABLE alert_rules (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id       UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    trigger_type VARCHAR(50) NOT NULL,
    conditions   JSONB NOT NULL,
    channel      VARCHAR(50) NOT NULL DEFAULT 'email',
    recipients   JSONB,
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_by   UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_rules_org_active ON alert_rules (org_id, is_active);

CREATE TRIGGER set_alert_rules_updated_at
    BEFORE UPDATE ON alert_rules
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- Create alert_history table
CREATE TABLE alert_history (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id        UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    alert_rule_id UUID NOT NULL REFERENCES alert_rules (id) ON DELETE CASCADE,
    customer_id   UUID REFERENCES customers (id) ON DELETE SET NULL,
    trigger_data  JSONB,
    channel       VARCHAR(50) NOT NULL,
    status        VARCHAR(20) NOT NULL CHECK (status IN ('sent', 'failed', 'pending')),
    sent_at       TIMESTAMPTZ,
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_history_org_sent ON alert_history (org_id, sent_at DESC);
CREATE INDEX idx_alert_history_rule ON alert_history (alert_rule_id);
