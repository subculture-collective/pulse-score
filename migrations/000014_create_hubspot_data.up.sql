CREATE TABLE IF NOT EXISTS hubspot_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    hubspot_contact_id VARCHAR(255) NOT NULL,
    email CITEXT,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    hubspot_company_id VARCHAR(255),
    lifecycle_stage VARCHAR(100),
    lead_status VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, hubspot_contact_id)
);

CREATE TABLE IF NOT EXISTS hubspot_deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    hubspot_deal_id VARCHAR(255) NOT NULL,
    hubspot_contact_id VARCHAR(255),
    deal_name VARCHAR(500),
    stage VARCHAR(255),
    amount_cents BIGINT DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    close_date TIMESTAMPTZ,
    pipeline VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, hubspot_deal_id)
);

CREATE TABLE IF NOT EXISTS hubspot_companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    hubspot_company_id VARCHAR(255) NOT NULL,
    name VARCHAR(500),
    domain VARCHAR(500),
    industry VARCHAR(255),
    number_of_employees INTEGER,
    annual_revenue_cents BIGINT DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, hubspot_company_id)
);

CREATE INDEX idx_hubspot_contacts_org_id ON hubspot_contacts(org_id);
CREATE INDEX idx_hubspot_contacts_email ON hubspot_contacts(email);
CREATE INDEX idx_hubspot_contacts_customer_id ON hubspot_contacts(customer_id);
CREATE INDEX idx_hubspot_deals_org_id ON hubspot_deals(org_id);
CREATE INDEX idx_hubspot_deals_customer_id ON hubspot_deals(customer_id);
CREATE INDEX idx_hubspot_companies_org_id ON hubspot_companies(org_id);
