-- Create health_scores table (current score per customer)
CREATE TABLE health_scores (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id        UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    customer_id   UUID NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    overall_score INTEGER NOT NULL CHECK (overall_score >= 0 AND overall_score <= 100),
    risk_level    VARCHAR(20) NOT NULL CHECK (risk_level IN ('green', 'yellow', 'red')),
    factors       JSONB NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (customer_id)
);

CREATE INDEX idx_health_scores_org_risk ON health_scores (org_id, risk_level);

CREATE TRIGGER set_health_scores_updated_at
    BEFORE UPDATE ON health_scores
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- Create health_score_history table
CREATE TABLE health_score_history (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id        UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    customer_id   UUID NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    overall_score INTEGER NOT NULL CHECK (overall_score >= 0 AND overall_score <= 100),
    risk_level    VARCHAR(20) NOT NULL CHECK (risk_level IN ('green', 'yellow', 'red')),
    factors       JSONB NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_health_score_history_customer_time ON health_score_history (customer_id, calculated_at DESC);
CREATE INDEX idx_health_score_history_org_time ON health_score_history (org_id, calculated_at DESC);
