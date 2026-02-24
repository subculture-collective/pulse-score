CREATE TABLE IF NOT EXISTS org_subscriptions (
    id                     UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id                 UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    stripe_subscription_id VARCHAR(255) UNIQUE,
    stripe_customer_id     VARCHAR(255),
    plan_tier              VARCHAR(50) NOT NULL DEFAULT 'free',
    billing_cycle          VARCHAR(20) NOT NULL DEFAULT 'monthly'
                           CHECK (billing_cycle IN ('monthly', 'annual')),
    status                 VARCHAR(50) NOT NULL DEFAULT 'inactive',
    current_period_start   TIMESTAMPTZ,
    current_period_end     TIMESTAMPTZ,
    cancel_at_period_end   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (org_id)
);

CREATE INDEX idx_org_subscriptions_org_status ON org_subscriptions (org_id, status);
CREATE INDEX idx_org_subscriptions_customer ON org_subscriptions (stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;

CREATE TRIGGER set_org_subscriptions_updated_at
    BEFORE UPDATE ON org_subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
