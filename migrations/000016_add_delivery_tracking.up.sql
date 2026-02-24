-- Add delivery tracking columns to alert_history
ALTER TABLE alert_history
    ADD COLUMN sendgrid_message_id VARCHAR(255),
    ADD COLUMN delivered_at        TIMESTAMPTZ,
    ADD COLUMN opened_at           TIMESTAMPTZ,
    ADD COLUMN clicked_at          TIMESTAMPTZ,
    ADD COLUMN bounced_at          TIMESTAMPTZ;

CREATE INDEX idx_alert_history_sendgrid_msg ON alert_history (sendgrid_message_id) WHERE sendgrid_message_id IS NOT NULL;
