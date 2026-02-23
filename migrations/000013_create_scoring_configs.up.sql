CREATE TABLE scoring_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    weights     JSONB NOT NULL DEFAULT '{"payment_recency": 0.3, "mrr_trend": 0.2, "failed_payments": 0.2, "support_tickets": 0.15, "engagement": 0.15}',
    thresholds  JSONB NOT NULL DEFAULT '{"green": 70, "yellow": 40}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT scoring_configs_org_unique UNIQUE (org_id)
);

CREATE INDEX idx_scoring_configs_org_id ON scoring_configs(org_id);
