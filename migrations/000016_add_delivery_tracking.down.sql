ALTER TABLE alert_history
    DROP COLUMN IF EXISTS sendgrid_message_id,
    DROP COLUMN IF EXISTS delivered_at,
    DROP COLUMN IF EXISTS opened_at,
    DROP COLUMN IF EXISTS clicked_at,
    DROP COLUMN IF EXISTS bounced_at;
