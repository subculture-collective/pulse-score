-- PulseScore Development Seed Data
-- Idempotent: uses ON CONFLICT to allow re-running safely.

-- ============================================================
-- Organization
-- ============================================================
INSERT INTO organizations (id, name, slug, plan, stripe_customer_id)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Acme SaaS',
    'acme-saas',
    'pro',
    'cus_acme_demo_001'
) ON CONFLICT (slug) DO NOTHING;

-- ============================================================
-- Users  (passwords are bcrypt of "password123")
-- ============================================================
INSERT INTO users (id, email, password_hash, first_name, last_name, email_verified)
VALUES
    ('b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', 'owner@acme.com',
     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
     'Alice', 'Owner', true),
    ('b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a02', 'admin@acme.com',
     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
     'Bob', 'Admin', true),
    ('b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a03', 'member@acme.com',
     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
     'Carol', 'Member', true)
ON CONFLICT (email) DO NOTHING;

-- ============================================================
-- User ↔ Organization roles
-- ============================================================
INSERT INTO user_organizations (user_id, org_id, role)
VALUES
    ('b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'owner'),
    ('b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a02', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'admin'),
    ('b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a03', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'member')
ON CONFLICT (user_id, org_id) DO NOTHING;

-- ============================================================
-- Integration connection (Stripe active)
-- ============================================================
INSERT INTO integration_connections (id, org_id, provider, status, external_account_id, metadata)
VALUES (
    'c1eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'stripe',
    'active',
    'acct_demo_001',
    '{"connected_at": "2025-11-01T00:00:00Z"}'
) ON CONFLICT (org_id, provider) DO NOTHING;

-- ============================================================
-- 50 Customers  (deterministic UUIDs for referenceability)
-- ============================================================
DO $$
DECLARE
    org UUID := 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11';
    cust_id UUID;
    i INTEGER;
    risk TEXT;
    score INTEGER;
    mrr INTEGER;
    cust_name TEXT;
    company TEXT;
    factors JSONB;
    first_seen TIMESTAMPTZ;
    last_seen TIMESTAMPTZ;
BEGIN
    FOR i IN 1..50 LOOP
        cust_id := ('d0000000-0000-4000-a000-' || lpad(i::text, 12, '0'))::uuid;

        -- Distribute: 60% green, 25% yellow, 15% red
        IF i <= 30 THEN
            risk   := 'green';
            score  := 70 + (i % 31);       -- 70-100
            mrr    := 5000 + (i * 200);     -- $50-$160
        ELSIF i <= 42 THEN
            risk   := 'yellow';
            score  := 40 + (i % 30);        -- 40-69
            mrr    := 3000 + (i * 100);     -- $30-$72
        ELSE
            risk   := 'red';
            score  := 5 + (i % 35);         -- 5-39
            mrr    := 1000 + (i * 50);      -- $10-$35
        END IF;

        cust_name := 'Customer ' || i;
        company   := 'Company ' || chr(64 + (i % 26) + 1);
        first_seen := NOW() - INTERVAL '90 days' + (i || ' hours')::interval;
        last_seen  := NOW() - ((i % 7) || ' days')::interval;

        -- Upsert customer
        INSERT INTO customers (id, org_id, external_id, source, email, name, company_name,
                               mrr_cents, currency, first_seen_at, last_seen_at, metadata)
        VALUES (
            cust_id, org, 'cus_' || lpad(i::text, 4, '0'), 'stripe',
            'customer' || i || '@example.com', cust_name, company,
            mrr, 'USD', first_seen, last_seen, '{}'
        ) ON CONFLICT (org_id, source, external_id) DO NOTHING;

        -- Health score
        factors := jsonb_build_object(
            'payment_recency', GREATEST(0, score - 5 + (i % 10)),
            'mrr_trend',       GREATEST(0, score + 3 - (i % 8)),
            'support_tickets', GREATEST(0, score - (i % 15)),
            'usage_frequency', GREATEST(0, score + (i % 12) - 6)
        );

        INSERT INTO health_scores (id, org_id, customer_id, overall_score, risk_level, factors, calculated_at)
        VALUES (
            ('e0000000-0000-4000-a000-' || lpad(i::text, 12, '0'))::uuid,
            org, cust_id, score, risk, factors, NOW()
        ) ON CONFLICT (customer_id) DO NOTHING;

        -- Seed a history row (7 days ago)
        INSERT INTO health_score_history (org_id, customer_id, overall_score, risk_level, factors, calculated_at)
        VALUES (org, cust_id, GREATEST(0, LEAST(100, score + (i % 11) - 5)), risk, factors,
                NOW() - INTERVAL '7 days')
        ON CONFLICT DO NOTHING;

        -- Stripe subscription (one per customer)
        INSERT INTO stripe_subscriptions (org_id, customer_id, stripe_subscription_id, status,
                                          plan_name, amount_cents, currency, interval,
                                          current_period_start, current_period_end)
        VALUES (
            org, cust_id, 'sub_demo_' || lpad(i::text, 4, '0'),
            CASE WHEN risk = 'red' THEN 'canceled' ELSE 'active' END,
            CASE WHEN mrr > 8000 THEN 'Enterprise' WHEN mrr > 4000 THEN 'Pro' ELSE 'Starter' END,
            mrr, 'USD', 'month',
            date_trunc('month', NOW()), date_trunc('month', NOW()) + INTERVAL '1 month'
        ) ON CONFLICT (stripe_subscription_id) DO NOTHING;

    END LOOP;
END $$;

-- ============================================================
-- ~500 customer events spanning 90 days
-- ============================================================
DO $$
DECLARE
    org UUID := 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11';
    cust_id UUID;
    i INTEGER;
    j INTEGER;
    evt_type TEXT;
    evt_types TEXT[] := ARRAY[
        'payment.success', 'payment.failed',
        'subscription.created', 'subscription.renewed', 'subscription.cancelled',
        'ticket.opened', 'ticket.resolved',
        'login', 'feature.used'
    ];
    occurred TIMESTAMPTZ;
BEGIN
    FOR i IN 1..50 LOOP
        cust_id := ('d0000000-0000-4000-a000-' || lpad(i::text, 12, '0'))::uuid;
        FOR j IN 1..10 LOOP  -- 10 events per customer = 500 total
            evt_type := evt_types[1 + ((i + j) % array_length(evt_types, 1))];
            occurred := NOW() - ((j * 9 + i % 5) || ' days')::interval
                             - ((i * 37 + j * 13) % 1440 || ' minutes')::interval;

            INSERT INTO customer_events (org_id, customer_id, event_type, source,
                                         external_event_id, occurred_at, data)
            VALUES (
                org, cust_id, evt_type, 'stripe',
                'evt_' || i || '_' || j,
                occurred,
                jsonb_build_object('amount_cents', (i * 100 + j * 50), 'demo', true)
            ) ON CONFLICT (org_id, source, external_event_id) DO NOTHING;
        END LOOP;
    END LOOP;
END $$;

-- ============================================================
-- Sample Stripe payments (a few per customer)
-- ============================================================
DO $$
DECLARE
    org UUID := 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11';
    cust_id UUID;
    i INTEGER;
    j INTEGER;
BEGIN
    FOR i IN 1..50 LOOP
        cust_id := ('d0000000-0000-4000-a000-' || lpad(i::text, 12, '0'))::uuid;
        FOR j IN 1..3 LOOP
            INSERT INTO stripe_payments (org_id, customer_id, stripe_payment_id,
                                         amount_cents, currency, status, paid_at)
            VALUES (
                org, cust_id,
                'pi_demo_' || i || '_' || j,
                5000 + (i * 200),
                'USD',
                CASE WHEN i > 43 AND j = 1 THEN 'failed' ELSE 'succeeded' END,
                NOW() - ((j * 30) || ' days')::interval
            ) ON CONFLICT (stripe_payment_id) DO NOTHING;
        END LOOP;
    END LOOP;
END $$;

-- ============================================================
-- Alert rules
-- ============================================================
INSERT INTO alert_rules (id, org_id, name, description, trigger_type, conditions, channel, recipients, is_active, created_by)
VALUES
    ('f0000000-0000-4000-a000-000000000001',
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Score Drop Alert',
     'Alert when a customer health score drops below 50',
     'score_drop',
     '{"threshold": 50, "direction": "below"}',
     'email',
     '["owner@acme.com", "admin@acme.com"]',
     true,
     'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a01'),
    ('f0000000-0000-4000-a000-000000000002',
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'Payment Failed Alert',
     'Alert when a customer payment fails',
     'payment_failed',
     '{"consecutive_failures": 1}',
     'email',
     '["owner@acme.com"]',
     true,
     'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a01')
ON CONFLICT (id) DO NOTHING;
