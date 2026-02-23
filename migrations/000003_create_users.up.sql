-- Create users table
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email           CITEXT UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    first_name      VARCHAR(100),
    last_name       VARCHAR(100),
    avatar_url      TEXT,
    email_verified  BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE TRIGGER set_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- Create user_organizations join table
CREATE TABLE user_organizations (
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    org_id     UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    role       VARCHAR(20) NOT NULL DEFAULT 'member'
               CHECK (role IN ('owner', 'admin', 'member')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, org_id)
);

CREATE INDEX idx_user_organizations_org_id ON user_organizations (org_id);
