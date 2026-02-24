CREATE TABLE IF NOT EXISTS intercom_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    intercom_contact_id VARCHAR(255) NOT NULL,
    email CITEXT,
    name VARCHAR(500),
    role VARCHAR(100),
    intercom_company_id VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, intercom_contact_id)
);

CREATE TABLE IF NOT EXISTS intercom_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    intercom_conversation_id VARCHAR(255) NOT NULL,
    intercom_contact_id VARCHAR(255),
    state VARCHAR(100),
    rating INTEGER,
    rating_remark TEXT,
    open BOOLEAN DEFAULT true,
    read BOOLEAN DEFAULT false,
    priority VARCHAR(50),
    subject VARCHAR(1000),
    created_at_remote TIMESTAMPTZ,
    updated_at_remote TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    first_response_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, intercom_conversation_id)
);

CREATE INDEX idx_intercom_contacts_org_id ON intercom_contacts(org_id);
CREATE INDEX idx_intercom_contacts_email ON intercom_contacts(email);
CREATE INDEX idx_intercom_contacts_customer_id ON intercom_contacts(customer_id);
CREATE INDEX idx_intercom_conversations_org_id ON intercom_conversations(org_id);
CREATE INDEX idx_intercom_conversations_customer_id ON intercom_conversations(customer_id);
CREATE INDEX idx_intercom_conversations_state ON intercom_conversations(org_id, state);
CREATE INDEX idx_intercom_conversations_created_remote ON intercom_conversations(org_id, created_at_remote);
