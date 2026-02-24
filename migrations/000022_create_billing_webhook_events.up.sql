CREATE TABLE IF NOT EXISTS billing_webhook_events (
    event_id     VARCHAR(255) PRIMARY KEY,
    event_type   VARCHAR(120) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_billing_webhook_events_processed_at ON billing_webhook_events (processed_at DESC);
