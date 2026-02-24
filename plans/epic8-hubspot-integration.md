# Epic 8: HubSpot Integration — Execution Plan

> **Epic Issue:** [#7 — HubSpot Integration](https://github.com/subculture-collective/pulse-score/issues/7)
> **Sub-Issues:** #123, #124, #125, #126, #127, #128, #129, #130
> **Phase:** MVP | **Priority:** High

---

## Overview

Implement HubSpot OAuth connection to enrich customer data with CRM information. Sync contacts, deals, and companies from HubSpot, match to existing Stripe-sourced customers by email, and process webhooks for real-time updates.

This plan follows the established Stripe integration patterns: handler → service → repository layering, AES-GCM token encryption, sync orchestrator sequencing, and the React connection card UI pattern.

---

## Dependency Graph

```
#123 HubSpot OAuth ──────────┬──→ #124 Contact & Deal Sync ──┬──→ #125 Company Enrichment
                             │                                │
                             │                                ├──→ #129 Data Mapping & Dedup
                             │                                │
                             │                                ├──→ #126 Webhook Handler
                             │                                │
                             │                                └──→ #130 Initial Sync Orchestrator
                             │                                           │
                             │         #129 ─────────────────────────────┘
                             │
                             └──→ #128 HubSpot Connection UI ──→ #127 Incremental Sync & UI
```

**Execution order (4 phases):**

1. **Phase A:** #123 (OAuth) — foundation, no other issue can start without it
2. **Phase B:** #124 (Contact & Deal Sync), #128 (Connection UI) — can run in parallel
3. **Phase C:** #125 (Company Enrichment), #129 (Data Mapping & Dedup), #126 (Webhook Handler) — depend on sync service
4. **Phase D:** #130 (Sync Orchestrator), #127 (Incremental Sync & UI) — depends on all sync + dedup

---

## Phase A: Foundation

### Issue #123 — Implement HubSpot OAuth Connect Flow

**Goal:** OAuth 2.0 connect/callback/disconnect flow with encrypted token storage and auto-refresh.

#### Step 1: Add HubSpot config

**File:** `internal/config/config.go`

- Add `HubSpotConfig` struct mirroring `StripeConfig`:
  ```go
  type HubSpotConfig struct {
      ClientID         string // HUBSPOT_CLIENT_ID
      ClientSecret     string // HUBSPOT_CLIENT_SECRET
      OAuthRedirectURL string // HUBSPOT_OAUTH_REDIRECT_URL
      EncryptionKey    string // HUBSPOT_ENCRYPTION_KEY (32-byte hex AES key)
      WebhookSecret    string // HUBSPOT_WEBHOOK_SECRET
      SyncIntervalMin  int    // HUBSPOT_SYNC_INTERVAL_MIN (default 15)
  }
  ```
- Add `HubSpot HubSpotConfig` field to the root `Config` struct
- Load from env vars in `Load()`, default `SyncIntervalMin` to 15

**File:** `.env.example`

- Add `HUBSPOT_CLIENT_ID`, `HUBSPOT_CLIENT_SECRET`, `HUBSPOT_OAUTH_REDIRECT_URL`, `HUBSPOT_ENCRYPTION_KEY`, `HUBSPOT_WEBHOOK_SECRET`, `HUBSPOT_SYNC_INTERVAL_MIN` entries

#### Step 2: Create HubSpot OAuth service

**File:** `internal/service/hubspot_oauth.go`

- `HubSpotOAuthService` struct with `cfg`, `connRepo`, `encryptionKey`
- `ConnectURL(orgID string) (string, error)` — Build HubSpot OAuth authorize URL:
  - URL: `https://app.hubspot.com/oauth/authorize`
  - Params: `client_id`, `redirect_uri`, `scope`, `state` (encode `orgID:timestamp`)
  - Scopes: `crm.objects.contacts.read crm.objects.deals.read crm.objects.companies.read`
- `ExchangeCode(ctx, orgID, code, state string) error` — POST to `https://api.hubapi.com/oauth/v1/token`:
  - Body: `grant_type=authorization_code`, `client_id`, `client_secret`, `redirect_uri`, `code`
  - Validate state contains correct orgID
  - Encrypt access_token and refresh_token with AES-GCM (reuse `encryptToken`/`decryptToken` from Stripe pattern)
  - Upsert `IntegrationConnection` with `provider="hubspot"`, `status="active"`, encrypted tokens, `token_expires_at` (30 min from now)
- `RefreshToken(ctx, orgID string) error` — POST to `https://api.hubapi.com/oauth/v1/token`:
  - Body: `grant_type=refresh_token`, `client_id`, `client_secret`, `refresh_token`
  - Update encrypted tokens + new expiry
- `GetAccessToken(ctx, orgID string) (string, error)` — Decrypt token, auto-refresh if expired
- `GetStatus(ctx, orgID string) (*HubSpotConnectionStatus, error)` — Return connection info
- `Disconnect(ctx, orgID string) error` — Delete via `connRepo.Delete(ctx, orgID, "hubspot")`

#### Step 3: Create HubSpot handler

**File:** `internal/handler/integration_hubspot.go`

- `IntegrationHubSpotHandler` struct with `oauthSvc`, `syncOrchestrator`
- `HandleConnect(w, r)` — GET, extract orgID from JWT, redirect to `oauthSvc.ConnectURL(orgID)`
- `HandleCallback(w, r)` — GET, extract code/state params, call `oauthSvc.ExchangeCode()`, trigger initial sync (background goroutine), redirect to frontend settings
- `HandleStatus(w, r)` — GET, return JSON connection status
- `HandleDisconnect(w, r)` — DELETE, call `oauthSvc.Disconnect()`
- `HandleSync(w, r)` — POST, trigger manual sync via orchestrator

#### Step 4: Register routes

**File:** `cmd/api/main.go`

- Instantiate `HubSpotOAuthService` and `IntegrationHubSpotHandler`
- Register routes under protected group (JWT + TenantIsolation + RequireRole("admin")):
  ```
  GET  /api/v1/integrations/hubspot/connect
  GET  /api/v1/integrations/hubspot/callback
  GET  /api/v1/integrations/hubspot/status
  DELETE /api/v1/integrations/hubspot
  POST /api/v1/integrations/hubspot/sync
  ```

#### Step 5: Add interface

**File:** `internal/handler/interfaces.go`

- Add `hubspotOAuthServicer` interface matching the handler's service dependency

#### Step 6: Tests

**File:** `internal/handler/integration_hubspot_test.go`

- Test OAuth redirect URL generation (correct scopes, state param)
- Test callback with valid/invalid code and state
- Test status endpoint
- Test disconnect endpoint
- Test error scenarios (denied access, invalid state)

**File:** `internal/service/hubspot_oauth_test.go`

- Test `ConnectURL` generates correct URL with scopes
- Test `ExchangeCode` with mocked HTTP (success/failure)
- Test `RefreshToken` flow
- Test `GetAccessToken` auto-refresh when expired
- Test state parameter CSRF validation

#### Acceptance Criteria
- [ ] OAuth flow redirects to HubSpot correctly
- [ ] Callback exchanges code for tokens
- [ ] Tokens stored encrypted in integration_connections
- [ ] Token refresh works when expired
- [ ] CSRF protection via state parameter
- [ ] Error handling for denied access
- [ ] Tests cover OAuth flow and token storage

---

## Phase B: Core Sync & Frontend (parallel tracks)

### Issue #124 — Implement HubSpot Contact and Deal Sync Service

**Goal:** Fetch contacts and deals from HubSpot API, map to local customer/event records with pagination and rate limiting.

#### Step 1: Create HubSpot-specific database tables

**File:** `migrations/000014_create_hubspot_data.up.sql`

```sql
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
```

**File:** `migrations/000014_create_hubspot_data.down.sql`

```sql
DROP TABLE IF EXISTS hubspot_companies;
DROP TABLE IF EXISTS hubspot_deals;
DROP TABLE IF EXISTS hubspot_contacts;
```

#### Step 2: Create HubSpot repositories

**File:** `internal/repository/hubspot_contact.go`

- `HubSpotContactRepository` with `*pgxpool.Pool`
- `Upsert(ctx, contact HubSpotContact) error` — ON CONFLICT (org_id, hubspot_contact_id) DO UPDATE
- `GetByOrgID(ctx, orgID string) ([]HubSpotContact, error)` — List all for org
- `GetByEmail(ctx, orgID, email string) (*HubSpotContact, error)`
- `CountByOrgID(ctx, orgID string) (int, error)`
- Define `HubSpotContact` model struct

**File:** `internal/repository/hubspot_deal.go`

- `HubSpotDealRepository` with `*pgxpool.Pool`
- `Upsert(ctx, deal HubSpotDeal) error`
- `GetByOrgID(ctx, orgID string) ([]HubSpotDeal, error)`
- `GetByContactID(ctx, orgID, hubspotContactID string) ([]HubSpotDeal, error)`
- `CountByOrgID(ctx, orgID string) (int, error)`

#### Step 3: Create HubSpot API client helper

**File:** `internal/service/hubspot_client.go`

- `HubSpotClient` struct wrapping `*http.Client` with rate-limiting:
  - Rate limiter: `time.Ticker` or `golang.org/x/time/rate` — 10 requests/second (HubSpot allows 100/10s, use 10/s for safety)
  - `Get(ctx, url, accessToken) (*http.Response, error)` — adds `Authorization: Bearer` header, respects rate limit
  - `ListContacts(ctx, accessToken, after string) (*ContactListResponse, error)` — GET `/crm/v3/objects/contacts?limit=100&after=...&properties=email,firstname,lastname,company,lifecyclestage,hs_lead_status,associatedcompanyid`
  - `ListDeals(ctx, accessToken, after string) (*DealListResponse, error)` — GET `/crm/v3/objects/deals?limit=100&after=...&properties=dealname,dealstage,amount,closedate,pipeline&associations=contacts`
  - `ListCompanies(ctx, accessToken, after string) (*CompanyListResponse, error)` — GET `/crm/v3/objects/companies?limit=100&after=...&properties=name,domain,industry,numberofemployees,annualrevenue`
  - `SearchContacts(ctx, accessToken, filterGroups, after) (*ContactListResponse, error)` — POST `/crm/v3/objects/contacts/search` (for incremental sync)
- Define HubSpot API response structs: `ContactListResponse`, `DealListResponse`, etc.

#### Step 4: Create HubSpot sync service

**File:** `internal/service/hubspot_sync.go`

- `HubSpotSyncService` struct with `oauthSvc`, `client`, `contactRepo`, `dealRepo`, `customerRepo`, `eventRepo`
- `SyncContacts(ctx, orgID string) (*SyncProgress, error)`:
  1. Get access token via `oauthSvc.GetAccessToken(ctx, orgID)`
  2. Paginate through all contacts using `client.ListContacts()`
  3. For each contact: upsert into `hubspot_contacts` table
  4. Also upsert into `customers` table with `source="hubspot"`, `external_id=hubspot_contact_id`
  5. Return progress stats
- `SyncDeals(ctx, orgID string) (*SyncProgress, error)`:
  1. Paginate through deals via `client.ListDeals()`
  2. Upsert into `hubspot_deals` table
  3. For each deal with associated contact: create customer_event with `event_type="deal_stage_change"`, `source="hubspot"`
- `SyncContactsSince(ctx, orgID string, since time.Time) (*SyncProgress, error)` — Incremental using HubSpot search API with `lastmodifieddate` filter
- `SyncDealsSince(ctx, orgID string, since time.Time) (*SyncProgress, error)` — Same pattern for deals

#### Step 5: Tests

**File:** `internal/service/hubspot_sync_test.go`

- Mock HubSpot API responses (contacts, deals with pagination)
- Test contacts sync with pagination (multiple pages)
- Test deals sync with contact associations
- Test customer upsert (no duplicates)
- Test customer_event creation for deals
- Test rate limiting behavior
- Test token auto-refresh on 401

**File:** `internal/repository/hubspot_contact_test.go` / `hubspot_deal_test.go`

- Test upsert behavior (insert + update)
- Test query methods

#### Acceptance Criteria
- [ ] Contacts fetched with pagination
- [ ] Deals fetched and linked to contacts
- [ ] Customers upserted without duplicates
- [ ] HubSpot rate limits respected
- [ ] Token auto-refresh works
- [ ] Customer events created for deal changes
- [ ] Tests with mocked HubSpot API

---

### Issue #128 — Create HubSpot Connection UI Component (parallel with #124)

**Goal:** React connection card for HubSpot in integrations settings, matching Stripe card pattern.

#### Step 1: Create HubSpot API client

**File:** `web/src/lib/hubspot.ts`

- Export `hubspotApi` object mirroring `stripeApi`:
  ```ts
  export const hubspotApi = {
    getConnectUrl: () => api.get<{ url: string }>('/integrations/hubspot/connect'),
    callback: (code: string, state: string) => api.post('/integrations/hubspot/callback', { code, state }),
    getStatus: () => api.get<HubSpotConnectionStatus>('/integrations/hubspot/status'),
    disconnect: () => api.delete('/integrations/hubspot'),
    triggerSync: () => api.post('/integrations/hubspot/sync'),
  }
  ```
- Define `HubSpotConnectionStatus` type: `status`, `externalAccountId`, `lastSyncAt`, `lastSyncError`, `contactCount`, `dealCount`, `connectedAt`

#### Step 2: Create HubSpot connection card

**File:** `web/src/components/integrations/HubSpotConnectionCard.tsx`

- Mirror `StripeConnectionCard` structure
- States: disconnected, connecting, connected, error
- Connected view: show portal info, contact count, deal count, last sync time
- Connect button → `hubspotApi.getConnectUrl()` → `window.location.href = url`
- Disconnect → confirm dialog → `hubspotApi.disconnect()`
- Manual sync → `hubspotApi.triggerSync()`
- Status badge (active/error/syncing)

#### Step 3: Create HubSpot OAuth callback page

**File:** `web/src/pages/settings/HubSpotCallbackPage.tsx`

- Mirror `StripeCallbackPage`
- Read `code` and `state` from URL params (HubSpot sends `code` on success)
- Call `hubspotApi.callback(code, state)` → navigate to `/settings` on success
- Show loading spinner during exchange
- Show error state on failure

#### Step 4: Integrate into settings page

**File:** `web/src/pages/settings/IntegrationsTab.tsx`

- Import and render `HubSpotConnectionCard` alongside `StripeConnectionCard`

**File:** Router config (add route for `/settings/hubspot/callback` → `HubSpotCallbackPage`)

#### Step 5: Tests

- Test all card states render correctly (disconnected, connecting, connected, error)
- Test connect initiates OAuth redirect
- Test disconnect with confirmation
- Test manual sync trigger
- Test callback page handles success/error

#### Acceptance Criteria
- [ ] Card matches Stripe card pattern
- [ ] All states render correctly
- [ ] Connect initiates OAuth flow
- [ ] Disconnect with confirmation
- [ ] Manual sync triggers refresh
- [ ] Tests cover all states

---

## Phase C: Enrichment, Dedup & Webhooks

### Issue #125 — Implement HubSpot Company Data Enrichment

**Goal:** Fetch company data from HubSpot to enrich customer profiles with company size, industry, and revenue.

**Depends on:** #124

#### Step 1: Create HubSpot company repository

**File:** `internal/repository/hubspot_company.go`

- `HubSpotCompanyRepository` with `*pgxpool.Pool`
- `Upsert(ctx, company HubSpotCompany) error`
- `GetByOrgID(ctx, orgID string) ([]HubSpotCompany, error)`
- `GetByHubSpotID(ctx, orgID, hubspotCompanyID string) (*HubSpotCompany, error)`

#### Step 2: Add company sync to HubSpot sync service

**File:** `internal/service/hubspot_sync.go` (extend)

- `SyncCompanies(ctx, orgID string) (*SyncProgress, error)`:
  1. Get access token
  2. Paginate through companies via `client.ListCompanies()`
  3. Upsert into `hubspot_companies` table
- `EnrichCustomersWithCompanyData(ctx, orgID string) error`:
  1. Query all `hubspot_contacts` for the org that have a `hubspot_company_id`
  2. For each, look up the company in `hubspot_companies`
  3. Update the linked `customers` record: set `company_name` from HubSpot company, add `metadata.hubspot_company` with industry, employee count, revenue
  4. Handle missing company associations gracefully (skip, log)

#### Step 3: Tests

**File:** `internal/service/hubspot_sync_test.go` (extend)

- Test company sync with pagination
- Test customer enrichment updates company_name and metadata
- Test missing company association handled gracefully
- Test enrichment runs after contact sync

#### Acceptance Criteria
- [ ] Companies fetched for associated contacts
- [ ] Customer company_name updated from HubSpot
- [ ] Industry and revenue stored in metadata
- [ ] Missing company associations handled gracefully
- [ ] Tests cover enrichment logic

---

### Issue #129 — Implement HubSpot Data Mapping and Deduplication

**Goal:** Intelligent matching between HubSpot records and existing customers (from Stripe or other sources), with merge logic.

**Depends on:** #124

#### Step 1: Create customer merge service

**File:** `internal/service/customer_merge.go`

- `CustomerMergeService` struct with `customerRepo`, `hubspotContactRepo`
- `MergeOrCreateFromHubSpot(ctx, orgID string, contact HubSpotContact) (*Customer, error)`:
  1. Look up existing customer by email: `customerRepo.GetByEmail(ctx, orgID, contact.Email)`
  2. **Match found:** Merge data — add HubSpot source info without overwriting existing Stripe fields
     - Shared fields: use most recently updated source (compare `updated_at`)
     - Add `hubspot` to `metadata.sources` array
     - Update `metadata.hubspot` with HubSpot-specific data
     - Do NOT overwrite MRR (Stripe is source of truth for billing)
  3. **No match:** Create new customer record with `source="hubspot"`
  4. Link `hubspot_contacts.customer_id` to the resolved customer
- `DeduplicateCustomers(ctx, orgID string) (*DeduplicationResult, error)`:
  1. Find customers with matching emails across different sources
  2. For each duplicate group: pick primary (oldest `first_seen_at`), merge metadata from others
  3. Return stats: `merged`, `skipped`, `errors`
- Field priority rules:
  - `email`: use from whichever source has it (both should match)
  - `name`: most recently updated
  - `company_name`: prefer HubSpot (CRM is authoritative for company data)
  - `mrr_cents`: always Stripe (billing source of truth)
  - `metadata`: deep merge, per-source namespacing (`metadata.stripe`, `metadata.hubspot`)

#### Step 2: Integrate into sync flow

**File:** `internal/service/hubspot_sync.go` (modify `SyncContacts`)

- After upserting each HubSpot contact, call `mergeSvc.MergeOrCreateFromHubSpot()` instead of direct customer upsert
- This replaces the naive upsert from #124 with intelligent dedup

#### Step 3: Tests

**File:** `internal/service/customer_merge_test.go`

- Test: HubSpot contact matches existing Stripe customer by email → merge
- Test: HubSpot contact has no match → create new
- Test: Conflict resolution follows priority rules (MRR stays from Stripe)
- Test: Multiple sources tracked in metadata
- Test: Deduplication across org
- Test: No data loss during merge

#### Acceptance Criteria
- [ ] Duplicate customers detected by email
- [ ] Data merged from multiple sources
- [ ] Conflict resolution follows priority rules
- [ ] Customer tracks all sources
- [ ] No data loss during merge
- [ ] Tests cover merge, conflict, no-match scenarios

---

### Issue #126 — Implement HubSpot Webhook Handler

**Goal:** Handle HubSpot webhook events for real-time updates with signature verification.

**Depends on:** #124

#### Step 1: Create HubSpot webhook service

**File:** `internal/service/hubspot_webhook.go`

- `HubSpotWebhookService` struct with `cfg`, `syncSvc`, `mergeSvc`, `connRepo`
- `VerifySignature(requestBody []byte, signatureHeader, timestamp string) error`:
  - HubSpot v3 signature: `HMAC-SHA256(clientSecret, httpMethod + requestURI + requestBody + timestamp)`
  - Compare computed hash with `X-HubSpot-Signature-v3` header
  - Validate timestamp is within 5 minutes (replay protection)
- `ProcessEvents(ctx, events []HubSpotWebhookEvent) error`:
  - For each event, route by `subscriptionType`:
    - `contact.propertyChange` → update local customer data via merge service
    - `deal.creation` → create customer event + upsert deal
    - `deal.propertyChange` → update deal + create event if stage change
    - `company.propertyChange` → update company data + re-enrich linked customers
  - Find org by looking up `integration_connections` with `provider="hubspot"` and matching `portal_id` from event
- Idempotency: in-memory map with `eventId` → `time.Time`, 24h TTL cleanup (same as Stripe pattern)

#### Step 2: Create webhook handler

**File:** `internal/handler/webhook_hubspot.go`

- `WebhookHubSpotHandler` struct with `webhookSvc`
- `HandleWebhook(w, r)`:
  - Read body (max 64KB)
  - Extract `X-HubSpot-Signature-v3` and `X-HubSpot-Request-Timestamp` headers
  - Call `webhookSvc.VerifySignature()`
  - Parse body as `[]HubSpotWebhookEvent` (HubSpot sends arrays)
  - Call `webhookSvc.ProcessEvents()`
  - Return 200 (even on processing errors, to prevent HubSpot retries)

#### Step 3: Register webhook route

**File:** `cmd/api/main.go`

- Register public route (no JWT auth):
  ```
  POST /api/v1/webhooks/hubspot
  ```

#### Step 4: Define event types

**File:** `internal/service/hubspot_webhook.go` (within)

```go
type HubSpotWebhookEvent struct {
    EventId          int64  `json:"eventId"`
    SubscriptionId   int64  `json:"subscriptionId"`
    PortalId         int64  `json:"portalId"`
    AppId            int64  `json:"appId"`
    OccurredAt       int64  `json:"occurredAt"`
    SubscriptionType string `json:"subscriptionType"`
    AttemptNumber    int    `json:"attemptNumber"`
    ObjectId         int64  `json:"objectId"`
    PropertyName     string `json:"propertyName"`
    PropertyValue    string `json:"propertyValue"`
}
```

#### Step 5: Tests

**File:** `internal/service/hubspot_webhook_test.go`

- Test signature verification (valid/invalid/expired timestamp)
- Test contact property change updates customer
- Test deal creation creates customer event
- Test batch event processing
- Test idempotency (duplicate events skipped)
- Test unknown event types ignored gracefully

**File:** `internal/handler/webhook_hubspot_test.go`

- Test webhook endpoint returns 200 on valid payload
- Test rejects invalid signature
- Test rejects oversized body

#### Acceptance Criteria
- [ ] Webhook endpoint accepts HubSpot events
- [ ] Signature verification prevents spoofing
- [ ] Contact changes update local customer data
- [ ] Deal changes create customer events
- [ ] Batch events processed correctly
- [ ] Tests cover signature verification and event handling

---

## Phase D: Orchestration & Incremental Sync

### Issue #130 — Create HubSpot Initial Sync Orchestrator

**Goal:** Full initial sync pipeline that runs when HubSpot is first connected.

**Depends on:** #124, #125, #129

#### Step 1: Create HubSpot sync orchestrator

**File:** `internal/service/hubspot_orchestrator.go`

- `HubSpotSyncOrchestratorService` struct with `connRepo`, `syncSvc`, `mergeSvc`, `enrichSvc`
- `RunFullSync(ctx, orgID string) (*SyncResult, error)`:
  1. `connRepo.UpdateSyncStatus(ctx, orgID, "hubspot", "syncing", nil)`
  2. **Step 1 — Contacts:** `syncSvc.SyncContacts(ctx, orgID)` → on failure, `markSyncError()`, continue (partial success)
  3. **Step 2 — Deals:** `syncSvc.SyncDeals(ctx, orgID)` → on failure, continue
  4. **Step 3 — Companies:** `syncSvc.SyncCompanies(ctx, orgID)` → on failure, continue
  5. **Step 4 — Enrichment:** `syncSvc.EnrichCustomersWithCompanyData(ctx, orgID)` → on failure, continue
  6. **Step 5 — Deduplication:** `mergeSvc.DeduplicateCustomers(ctx, orgID)` → on failure, continue
  7. `connRepo.UpdateSyncStatus(ctx, orgID, "hubspot", "active", &now)`
  8. Return `SyncResult{Contacts, Deals, Companies, Enriched, Deduplicated, Duration, Errors[]}`
- `RunIncrementalSync(ctx, orgID string, since time.Time) (*SyncResult, error)`:
  - Same pipeline but uses `SyncContactsSince` / `SyncDealsSince`
  - Skip full company sync (only refresh companies for changed contacts)
  - Re-run enrichment and dedup for changed records only
- Progress tracking: update `integration_connections.metadata` with step progress (JSON: `{"sync_step": "contacts", "contacts_synced": 150, ...}`)

#### Step 2: Wire into OAuth callback

**File:** `internal/handler/integration_hubspot.go` (modify `HandleCallback`)

- After `ExchangeCode()`, trigger `go orchestrator.RunFullSync(ctx, orgID)` in background goroutine

#### Step 3: Wire into manual sync

**File:** `internal/handler/integration_hubspot.go` (modify `HandleSync`)

- Call `orchestrator.RunFullSync(ctx, orgID)` synchronously (or background with status polling)

#### Step 4: Tests

**File:** `internal/service/hubspot_orchestrator_test.go`

- Test full sync executes steps in correct order
- Test partial failure: contacts succeed, deals fail → contacts still saved
- Test connection status updated throughout
- Test incremental sync uses `since` parameter
- Test progress tracking in metadata

#### Acceptance Criteria
- [ ] Full sync runs on HubSpot connection
- [ ] Steps execute in correct order
- [ ] Progress tracked and queryable
- [ ] Partial failures don't block other steps
- [ ] Connection status updated
- [ ] Tests verify orchestration

---

### Issue #127 — Implement HubSpot Incremental Sync and Connection UI

**Goal:** Incremental sync scheduler + enhanced connection card with sync stats.

**Depends on:** #124, #130

#### Step 1: Extend sync scheduler for HubSpot

**File:** `internal/service/sync_scheduler.go` (modify existing)

- In the scheduler loop, after processing Stripe connections, also query:
  ```go
  hubspotConns, _ := connRepo.ListActiveByProvider(ctx, "hubspot")
  ```
- For each active HubSpot connection:
  - Acquire per-org lock (same `TryLock()` pattern)
  - Call `hubspotOrchestrator.RunIncrementalSync(ctx, orgID, conn.LastSyncAt)`

#### Step 2: Add HubSpot connection monitoring

**File:** `internal/service/connection_monitor.go` (modify existing, or create `hubspot_connection_monitor.go`)

- Extend or add HubSpot-specific health check: make a lightweight HubSpot API call (e.g., `GET /crm/v3/objects/contacts?limit=1`) to verify token is still valid
- On failure: mark connection as error, log

#### Step 3: Enhance connection card with sync stats

**File:** `web/src/components/integrations/HubSpotConnectionCard.tsx` (extend from #128)

- Display from status API: contact count, deal count, company count, last sync time, next sync time
- Show sync step progress during active sync (poll status endpoint)
- Error state: show last sync error with retry button

#### Step 4: Tests

**File:** `internal/service/sync_scheduler_test.go` (extend)

- Test scheduler picks up HubSpot connections
- Test incremental sync uses correct `since` timestamp
- Test concurrent sync prevention (TryLock)

#### Acceptance Criteria
- [ ] Incremental sync fetches only changed data
- [ ] Connection card shows status and stats
- [ ] Connect/disconnect flow works
- [ ] Last sync time displayed
- [ ] Error states handled
- [ ] Tests cover incremental logic and UI

---

## Files Created / Modified Summary

### New Files

| File | Issue | Description |
|------|-------|-------------|
| `migrations/000014_create_hubspot_data.up.sql` | #124 | HubSpot contacts, deals, companies tables |
| `migrations/000014_create_hubspot_data.down.sql` | #124 | Drop HubSpot tables |
| `internal/service/hubspot_oauth.go` | #123 | OAuth connect, callback, token refresh |
| `internal/service/hubspot_client.go` | #124 | HubSpot API client with rate limiting |
| `internal/service/hubspot_sync.go` | #124, #125 | Contact, deal, company sync + enrichment |
| `internal/service/hubspot_webhook.go` | #126 | Webhook signature verification + event processing |
| `internal/service/hubspot_orchestrator.go` | #130 | Full/incremental sync orchestration |
| `internal/service/customer_merge.go` | #129 | Cross-source customer matching + dedup |
| `internal/handler/integration_hubspot.go` | #123 | OAuth + management HTTP handlers |
| `internal/handler/webhook_hubspot.go` | #126 | Webhook HTTP handler |
| `internal/repository/hubspot_contact.go` | #124 | HubSpot contact CRUD |
| `internal/repository/hubspot_deal.go` | #124 | HubSpot deal CRUD |
| `internal/repository/hubspot_company.go` | #125 | HubSpot company CRUD |
| `web/src/lib/hubspot.ts` | #128 | Frontend API client |
| `web/src/components/integrations/HubSpotConnectionCard.tsx` | #128 | Connection card component |
| `web/src/pages/settings/HubSpotCallbackPage.tsx` | #128 | OAuth callback page |

### Modified Files

| File | Issue | Description |
|------|-------|-------------|
| `internal/config/config.go` | #123 | Add `HubSpotConfig` struct |
| `.env.example` | #123 | Add HubSpot env vars |
| `cmd/api/main.go` | #123, #126 | Wire services, register routes |
| `internal/handler/interfaces.go` | #123 | Add HubSpot service interfaces |
| `internal/service/sync_scheduler.go` | #127 | Add HubSpot to sync loop |
| `internal/service/connection_monitor.go` | #127 | Add HubSpot health check |
| `web/src/pages/settings/IntegrationsTab.tsx` | #128 | Add HubSpot card |
| Router config | #128 | Add callback route |

### Test Files

| File | Issues |
|------|--------|
| `internal/service/hubspot_oauth_test.go` | #123 |
| `internal/service/hubspot_sync_test.go` | #124, #125 |
| `internal/service/hubspot_webhook_test.go` | #126 |
| `internal/service/hubspot_orchestrator_test.go` | #130 |
| `internal/service/customer_merge_test.go` | #129 |
| `internal/handler/integration_hubspot_test.go` | #123 |
| `internal/handler/webhook_hubspot_test.go` | #126 |
| `internal/repository/hubspot_contact_test.go` | #124 |
| `internal/repository/hubspot_deal_test.go` | #124 |

---

## Implementation Notes

### HubSpot API Specifics
- **Base URL:** `https://api.hubapi.com`
- **Auth:** Bearer token in `Authorization` header
- **Rate limits:** 100 requests per 10 seconds (use 10/s with `x/time/rate`)
- **Token expiry:** Access tokens expire in 30 minutes; always check + auto-refresh
- **Pagination:** Cursor-based with `after` parameter, `100` items per page
- **Search API:** POST `/crm/v3/objects/{type}/search` with `filterGroups` for incremental sync
- **Associations:** Deals → contacts resolved via `associations` query param or separate association API

### Key Differences from Stripe Pattern
1. **Token refresh:** HubSpot tokens expire in 30 min (vs Stripe's long-lived tokens) — must auto-refresh before every API call
2. **Rate limiting:** Must be explicit (Stripe SDK handles it internally)
3. **Webhook signature:** HMAC-SHA256 with client secret (vs Stripe's webhook secret + SDK verification)
4. **Data model:** Contacts/Deals/Companies (vs Customers/Subscriptions/Payments)
5. **Deduplication:** HubSpot records may overlap with Stripe customers by email — must merge, not duplicate
6. **Incremental sync:** Use HubSpot search API with `lastmodifieddate` filter (vs Stripe's `created` parameter)

### Security Considerations
- All tokens encrypted at rest with AES-GCM (same key management as Stripe)
- Webhook signature verification prevents spoofing
- State parameter in OAuth prevents CSRF
- Webhook endpoint is public but signature-protected (max 64KB body)
- No HubSpot credentials logged or exposed in error messages

---

## OpenAPI Updates

**File:** `docs/openapi.yaml`

Add the following endpoints to the spec:
- `GET /api/v1/integrations/hubspot/connect` — Initiate HubSpot OAuth
- `GET /api/v1/integrations/hubspot/callback` — OAuth callback
- `GET /api/v1/integrations/hubspot/status` — Connection status
- `DELETE /api/v1/integrations/hubspot` — Disconnect
- `POST /api/v1/integrations/hubspot/sync` — Trigger manual sync
- `POST /api/v1/webhooks/hubspot` — Webhook receiver

Add response schemas: `HubSpotConnectionStatus`, `HubSpotSyncResult`
