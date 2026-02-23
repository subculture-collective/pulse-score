-- Create invitations table for team member invitations
CREATE TABLE invitations (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id      UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    email       CITEXT NOT NULL,
    role        VARCHAR(20) NOT NULL DEFAULT 'member'
                CHECK (role IN ('owner', 'admin', 'member')),
    token       VARCHAR(255) UNIQUE NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'accepted', 'expired')),
    invited_by  UUID NOT NULL REFERENCES users (id),
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invitations_org_id ON invitations (org_id);
CREATE INDEX idx_invitations_email ON invitations (email);
CREATE INDEX idx_invitations_token ON invitations (token) WHERE status = 'pending';
