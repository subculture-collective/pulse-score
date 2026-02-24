-- Create notification_preferences table
CREATE TABLE notification_preferences (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id           UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    org_id            UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    email_enabled     BOOLEAN NOT NULL DEFAULT true,
    in_app_enabled    BOOLEAN NOT NULL DEFAULT true,
    digest_enabled    BOOLEAN NOT NULL DEFAULT false,
    digest_frequency  VARCHAR(20) NOT NULL DEFAULT 'weekly' CHECK (digest_frequency IN ('daily', 'weekly')),
    muted_rule_ids    UUID[] DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, org_id)
);

CREATE INDEX idx_notification_prefs_user_org ON notification_preferences (user_id, org_id);

CREATE TRIGGER set_notification_preferences_updated_at
    BEFORE UPDATE ON notification_preferences
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
