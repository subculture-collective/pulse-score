-- Create stripe_subscriptions table
CREATE TABLE stripe_subscriptions (
    id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id                   UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    customer_id              UUID NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    stripe_subscription_id   VARCHAR(255) UNIQUE NOT NULL,
    status                   VARCHAR(50) NOT NULL,
    plan_name                VARCHAR(255),
    amount_cents             INTEGER NOT NULL DEFAULT 0,
    currency                 VARCHAR(3) NOT NULL DEFAULT 'USD',
    interval                 VARCHAR(20),
    current_period_start     TIMESTAMPTZ,
    current_period_end       TIMESTAMPTZ,
    canceled_at              TIMESTAMPTZ,
    metadata                 JSONB NOT NULL DEFAULT '{}',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stripe_subscriptions_customer_status ON stripe_subscriptions (customer_id, status);
CREATE INDEX idx_stripe_subscriptions_org_status ON stripe_subscriptions (org_id, status);

CREATE TRIGGER set_stripe_subscriptions_updated_at
    BEFORE UPDATE ON stripe_subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- Create stripe_payments table
CREATE TABLE stripe_payments (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id            UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    customer_id       UUID NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    stripe_payment_id VARCHAR(255) UNIQUE NOT NULL,
    amount_cents      INTEGER NOT NULL,
    currency          VARCHAR(3) NOT NULL DEFAULT 'USD',
    status            VARCHAR(50) NOT NULL,
    failure_code      VARCHAR(100),
    failure_message   TEXT,
    paid_at           TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stripe_payments_customer_status ON stripe_payments (customer_id, status);
CREATE INDEX idx_stripe_payments_org_status ON stripe_payments (org_id, status);
CREATE INDEX idx_stripe_payments_paid_at ON stripe_payments (paid_at DESC);
