-- Create customer_events table
CREATE TABLE customer_events (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id            UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    customer_id       UUID NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    event_type        VARCHAR(100) NOT NULL,
    source            VARCHAR(50) NOT NULL,
    external_event_id VARCHAR(255),
    occurred_at       TIMESTAMPTZ NOT NULL,
    data              JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (org_id, source, external_event_id)
);

CREATE INDEX idx_customer_events_customer_time ON customer_events (customer_id, occurred_at DESC);
CREATE INDEX idx_customer_events_org_type ON customer_events (org_id, event_type);
CREATE INDEX idx_customer_events_org_time ON customer_events (org_id, occurred_at DESC);
