# Execution Plan: Epic 6 — REST API Layer (#21)

## Overview

**Epic:** [#21 — REST API Layer](https://github.com/subculture-collective/pulse-score/issues/21)
**Sub-issues:** #94–#103 (10 issues)
**Scope:** Build the complete REST API serving the React dashboard. Includes customer list/detail/timeline endpoints, dashboard analytics, integration management, organization settings, user management, alert rules CRUD, and OpenAPI documentation.

## Current State

The following foundations are already in place:

- **Database migrations** for all required tables: `customers` (000005), `customer_events` (000006), `health_scores` + `health_score_history` (000007), `stripe_data` (000008), `alerts` (000009), `scoring_configs` (000013)
- **Existing handlers:** `auth.go`, `health.go`, `integration_stripe.go`, `invitation.go`, `organization.go`, `scoring.go`, `user.go`
- **Existing repositories:** `CustomerRepository` (UpsertByExternal, GetByID, ListByOrg, UpdateMRR), `CustomerEventRepository` (Upsert, ListByCustomer, ListByCustomerAndType, CountEventsByTypeForOrg), `HealthScoreRepository` (UpsertCurrent, GetByCustomerID, ListByOrg, InsertHistory), `StripeSubscriptionRepository` (Upsert, ListActiveByCustomer, ListByOrg), `StripePaymentRepository` (Upsert, ListByCustomerAndStatus, ListFailedAfter), `IntegrationConnectionRepository` (Upsert, GetByOrgAndProvider, UpdateStatus, ListActiveByProvider, Delete), `OrganizationRepository` (Create, GetByID, AddMember, IsMember), `UserRepository` (Create, GetByEmail, GetByID, GetUserOrgs, UpdateProfile)
- **Existing services:** `AuthService`, `UserService`, `OrganizationService`, `InvitationService`, `StripeOAuthService`, `StripeSyncService`, `MRRService`, `PaymentHealthService`, `SyncOrchestratorService`, scoring services (`ScoreAggregator`, `RiskCategorizer`, `ConfigService`)
- **Middleware:** JWT auth, tenant isolation, `RequireRole("admin")`/`RequireRole("owner")`, security headers
- **Error types:** `ValidationError` (422), `ConflictError` (409), `AuthError` (401), `NotFoundError` (404), `ForbiddenError` (403)
- **Handler patterns:** constructor injection, `writeJSON`/`errorResponse`/`handleServiceError` helpers, Chi URL params, JSON decode

### Existing Repository Methods Available for Reuse

| Repository | Methods | Notes |
|------------|---------|-------|
| `CustomerRepository` | `ListByOrg`, `GetByID`, `UpdateMRR` | Need new methods for paginated/filtered list |
| `CustomerEventRepository` | `ListByCustomer`, `ListByCustomerAndType`, `CountEventsByTypeForOrg` | Need new paginated method with date range |
| `HealthScoreRepository` | `GetByCustomerID`, `ListByOrg`, `InsertHistory` | Existing ListByOrg useful for dashboard |
| `StripeSubscriptionRepository` | `ListActiveByCustomer`, `ListByOrg` | Used for customer detail |
| `IntegrationConnectionRepository` | `GetByOrgAndProvider`, `UpdateStatus`, `ListActiveByProvider`, `Delete` | Mostly reusable as-is |
| `OrganizationRepository` | `GetByID`, `IsMember` | Need new methods for settings update |
| `UserRepository` | `GetByID`, `GetUserOrgs`, `UpdateProfile` | Need methods for member listing/management |

### Existing Routes That Overlap

Some routes already exist that overlap with Epic 6 scope:
- `GET /api/v1/scoring/risk-distribution` — exists in `scoring.go`
- `GET /api/v1/scoring/histogram` — exists in `scoring.go`
- `GET /api/v1/integrations/stripe/*` — exists in `integration_stripe.go`

These will be referenced/reused rather than duplicated.

## Dependency Graph

```
#94 Customer List ──► #95 Customer Detail ──► #96 Customer Timeline
                                                                    
#97 Dashboard Summary ─────────┐                                    
#98 Score Distribution ────────┤ (independent of each other)       
#99 Integration Management ────┤                                    
#100 Organization Settings ────┤                                    
#101 User Management ──────────┤                                    
#102 Alert Rules CRUD ─────────┘                                    
                               │                                    
                               └──► #103 OpenAPI Documentation (after all endpoints)
```

## Execution Phases

Issues are grouped into phases based on dependency chains. Issues within the same phase can be worked on in parallel.

---

### Phase 1 — Customer List Endpoint (foundation for detail + timeline)

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#94](https://github.com/subculture-collective/pulse-score/issues/94) | Implement customer list endpoint | critical | `internal/handler/customer.go`, `internal/service/customer.go`, `internal/repository/customer.go` (modify) |

**Details:**

1. Add new repository methods to `internal/repository/customer.go`:

    ```go
    // Pagination/filter params
    type CustomerListParams struct {
        OrgID    uuid.UUID
        Page     int
        PerPage  int
        Sort     string // "name", "mrr", "score", "last_seen"
        Order    string // "asc", "desc"
        Risk     string // "green", "yellow", "red" (filter)
        Search   string // ILIKE on name/email
        Source   string // filter by source
    }

    type CustomerListResult struct {
        Customers  []CustomerWithScore
        Total      int
        Page       int
        PerPage    int
        TotalPages int
    }

    type CustomerWithScore struct {
        Customer
        OverallScore *int    // from health_scores join
        RiskLevel    *string // from health_scores join
    }

    // ListWithScores — paginated customer list with health score join
    func (r *CustomerRepository) ListWithScores(ctx context.Context, params CustomerListParams) (*CustomerListResult, error)
    ```

    - SQL: `SELECT c.*, hs.overall_score, hs.risk_level FROM customers c LEFT JOIN health_scores hs ON c.id = hs.customer_id WHERE c.org_id = $1 AND c.deleted_at IS NULL`
    - Add dynamic WHERE clauses: `risk_level = $N` (if risk set), `(c.name ILIKE $N OR c.email ILIKE $N)` (if search set), `c.source = $N` (if source set)
    - Add ORDER BY: validate sort field against allowlist (`name`, `mrr_cents`, `overall_score`, `last_seen_at`), default `name ASC`
    - Add `LIMIT $N OFFSET $N` for pagination
    - Run `SELECT COUNT(*)` with same WHERE (without LIMIT/OFFSET) for total count
    - Use parameterized queries — never interpolate user input into SQL

2. Create `internal/service/customer.go` — `CustomerService`:

    ```go
    type CustomerService struct {
        customerRepo *repository.CustomerRepository
        healthRepo   *repository.HealthScoreRepository
    }

    func NewCustomerService(cr *repository.CustomerRepository, hr *repository.HealthScoreRepository) *CustomerService

    func (s *CustomerService) List(ctx context.Context, params CustomerListParams) (*CustomerListResult, error)
    ```

    - Validate params: page >= 1 (default 1), per_page 1-100 (default 25), sort in allowlist, order in {"asc","desc"}
    - Return `ValidationError` for invalid params

3. Create `internal/handler/customer.go` — `CustomerHandler`:

    ```go
    type CustomerHandler struct {
        customerService *service.CustomerService
    }

    func NewCustomerHandler(cs *service.CustomerService) *CustomerHandler
    func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request)
    ```

    - Parse query params: `r.URL.Query().Get("page")`, etc.
    - Convert page/per_page to int with defaults
    - Extract `auth.GetOrgID(r.Context())` for tenant scoping
    - Call service, return JSON: `{customers: [...], pagination: {page, per_page, total, total_pages}}`

4. Register route in `cmd/api/main.go`:

    ```go
    // Inside protected routes group
    r.Get("/api/v1/customers", customerHandler.List)
    ```

5. Wire dependencies in `main.go`: create `CustomerService`, `CustomerHandler`, inject repos

**Tests:** `internal/handler/customer_test.go` — pagination (page 1 vs page 2, per_page limits), sorting (by name asc/desc, by mrr, by score, by last_seen), filtering (by risk level, by source), search (name match, email match, case-insensitive), empty results, default params, max per_page capped at 100

**Acceptance criteria:**

- [ ] Pagination works correctly with total count
- [ ] Sorting by all supported fields
- [ ] Filter by risk level returns correct results
- [ ] Search matches name or email (case-insensitive)
- [ ] Health score included in each customer record
- [ ] Response matches documented schema
- [ ] Performance: <200ms for 500 customers
- [ ] Tests cover pagination, sorting, filtering, search

---

### Phase 2 — Customer Detail + Timeline (sequential dependency on Phase 1)

#### Phase 2a — Customer Detail

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#95](https://github.com/subculture-collective/pulse-score/issues/95) | Implement customer detail endpoint | critical | `internal/handler/customer.go` (modify), `internal/service/customer.go` (modify), `internal/repository/customer.go` (modify) |

**Depends on:** Phase 1 (#94)

**Details:**

1. Add repository methods:
    - `CustomerRepository.GetByIDAndOrg(ctx, id, orgID uuid.UUID) (*Customer, error)` — returns nil for not found, enforces tenant isolation at DB level

2. Add service method to `CustomerService`:

    ```go
    type CustomerDetail struct {
        Customer      Customer
        HealthScore   *HealthScoreDetail    // overall + factors breakdown
        Subscriptions []StripeSubscription   // active subscriptions
        RecentEvents  []CustomerEvent        // last 10 events
        Sources       []string               // integration sources
    }

    type HealthScoreDetail struct {
        OverallScore int
        RiskLevel    string
        Factors      map[string]float64  // factor name → score
        CalculatedAt time.Time
    }

    func (s *CustomerService) GetDetail(ctx context.Context, customerID, orgID uuid.UUID) (*CustomerDetail, error)
    ```

    - Load customer by ID + org_id (404 if not found — `NotFoundError`)
    - Load health score via `HealthScoreRepository.GetByCustomerID`
    - Load active subscriptions via `StripeSubscriptionRepository.ListActiveByCustomer`
    - Load recent events via `CustomerEventRepository.ListByCustomer` (limit 10)
    - Aggregate sources from customer record + events
    - Can fire parallel queries using goroutines + `errgroup`

3. Add handler method:

    ```go
    func (h *CustomerHandler) GetDetail(w http.ResponseWriter, r *http.Request)
    ```

    - Parse `chi.URLParam(r, "id")`, validate as UUID
    - Extract org_id from context
    - Call service, return full detail JSON
    - 404 if not found or not in org

4. Register route:
    ```go
    r.Get("/api/v1/customers/{id}", customerHandler.GetDetail)
    ```

**Tests:** `internal/handler/customer_test.go` (add tests) — customer found with full detail, customer not found (404), cross-tenant isolation (customer in other org returns 404), invalid UUID format (400), score factors included in response, subscriptions listed

**Acceptance criteria:**

- [ ] Returns full customer profile
- [ ] Score includes factor-level breakdown
- [ ] Subscriptions listed with status and amounts
- [ ] Recent events included
- [ ] 404 for non-existent or other-org customer
- [ ] Tests cover found, not found, cross-tenant isolation

#### Phase 2b — Customer Timeline/Events

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#96](https://github.com/subculture-collective/pulse-score/issues/96) | Implement customer timeline/events endpoint | high | `internal/handler/customer.go` (modify), `internal/service/customer.go` (modify), `internal/repository/customer_event.go` (modify) |

**Depends on:** Phase 1 (#94 — uses same handler/service files)

**Details:**

1. Add repository method to `internal/repository/customer_event.go`:

    ```go
    type EventListParams struct {
        CustomerID uuid.UUID
        OrgID      uuid.UUID
        Page       int
        PerPage    int
        EventType  string    // filter by event_type
        From       time.Time // filter occurred_at >= from
        To         time.Time // filter occurred_at <= to
    }

    type EventListResult struct {
        Events     []CustomerEvent
        Total      int
        Page       int
        PerPage    int
        TotalPages int
    }

    func (r *CustomerEventRepository) ListPaginated(ctx context.Context, params EventListParams) (*EventListResult, error)
    ```

    - SQL: `SELECT * FROM customer_events WHERE customer_id = $1 AND org_id = $2`
    - Add optional WHERE: `event_type = $N`, `occurred_at >= $N`, `occurred_at <= $N`
    - ORDER BY `occurred_at DESC` (most recent first)
    - LIMIT + OFFSET for pagination
    - COUNT query for total

2. Add service method to `CustomerService`:

    ```go
    func (s *CustomerService) ListEvents(ctx context.Context, params EventListParams) (*EventListResult, error)
    ```

    - Validate params: page >= 1, per_page 1-100
    - Validate customer exists and belongs to org (404 if not)
    - Validate date range: `from` before `to` if both set
    - Parse event type against known types (or allow any)

3. Add handler method:

    ```go
    func (h *CustomerHandler) ListEvents(w http.ResponseWriter, r *http.Request)
    ```

    - Parse `chi.URLParam(r, "id")` for customer ID
    - Parse query params: `page`, `per_page`, `type`, `from` (RFC3339), `to` (RFC3339)
    - Return: `{events: [...], pagination: {...}}`

4. Register route:
    ```go
    r.Get("/api/v1/customers/{id}/events", customerHandler.ListEvents)
    ```

**Tests:** `internal/handler/customer_test.go` (add tests) — paginated events, filter by event type, date range filter, empty results, invalid customer ID, events ordered by most recent first

**Acceptance criteria:**

- [ ] Paginated event list returned
- [ ] Filter by event type works
- [ ] Date range filtering works
- [ ] Events ordered by most recent first
- [ ] Efficient query with proper index usage
- [ ] Tests cover pagination, filters, empty results

---

### Phase 3 — Dashboard & Analytics Endpoints (parallel)

These two endpoints are independent and can be developed in parallel.

#### Phase 3a — Dashboard Summary Stats

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#97](https://github.com/subculture-collective/pulse-score/issues/97) | Implement dashboard summary stats endpoint | high | `internal/handler/dashboard.go`, `internal/service/dashboard.go`, `internal/repository/customer.go` (modify), `internal/repository/health_score.go` (modify) |

**Details:**

1. Add repository helper methods:

    **In `internal/repository/customer.go`:**
    ```go
    func (r *CustomerRepository) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
    func (r *CustomerRepository) TotalMRRByOrg(ctx context.Context, orgID uuid.UUID) (int64, error) // sum of mrr_cents
    ```

    **In `internal/repository/health_score.go`:**
    ```go
    type RiskDistribution struct {
        Green  int
        Yellow int
        Red    int
    }

    func (r *HealthScoreRepository) GetRiskDistribution(ctx context.Context, orgID uuid.UUID) (*RiskDistribution, error)
    func (r *HealthScoreRepository) GetAverageScore(ctx context.Context, orgID uuid.UUID) (float64, error)
    ```

    **In `internal/repository/health_score.go` (history queries):**
    ```go
    func (r *HealthScoreRepository) GetAverageScoreAt(ctx context.Context, orgID uuid.UUID, at time.Time) (float64, error)
    func (r *HealthScoreRepository) CountAtRiskAt(ctx context.Context, orgID uuid.UUID, at time.Time) (int, error)
    ```

    Note: Some of these methods may already exist in the scoring services (`RiskCategorizer`). Check and reuse rather than duplicate.

2. Create `internal/service/dashboard.go` — `DashboardService`:

    ```go
    type DashboardSummary struct {
        TotalCustomers    int     `json:"total_customers"`
        RiskDistribution  RiskDist `json:"risk_distribution"`
        TotalMRRCents     int64   `json:"total_mrr_cents"`
        MRRChange30DCents int64   `json:"mrr_change_30d_cents"`
        AtRiskCount       int     `json:"at_risk_count"`
        AtRiskChange7D    int     `json:"at_risk_change_7d"`
        AvgHealthScore    float64 `json:"avg_health_score"`
        ScoreChange7D     float64 `json:"score_change_7d"`
    }

    type RiskDist struct {
        Green  int `json:"green"`
        Yellow int `json:"yellow"`
        Red    int `json:"red"`
    }

    func (s *DashboardService) GetSummary(ctx context.Context, orgID uuid.UUID) (*DashboardSummary, error)
    ```

    - Fire multiple queries in parallel with `errgroup`:
      - Customer count
      - Risk distribution
      - Total MRR (current) + MRR from 30d ago (for change)
      - At-risk count now vs 7d ago
      - Average score now vs 7d ago
    - Compute deltas: `mrr_change = current - 30d_ago`, `at_risk_change = current - 7d_ago`, `score_change = current - 7d_ago`
    - For historical comparisons, query `health_score_history` for snapshots at prior dates

3. Create `internal/handler/dashboard.go` — `DashboardHandler`:

    ```go
    type DashboardHandler struct {
        dashboardService *service.DashboardService
    }

    func NewDashboardHandler(ds *service.DashboardService) *DashboardHandler
    func (h *DashboardHandler) GetSummary(w http.ResponseWriter, r *http.Request)
    ```

    - Extract org_id from context
    - Call service, return JSON

4. Register route:
    ```go
    r.Get("/api/v1/dashboard/summary", dashboardHandler.GetSummary)
    ```

5. Wire dependencies in `main.go`

**Tests:** `internal/handler/dashboard_test.go` — correct customer counts, MRR totals accurate, risk distribution matches, trend deltas computed correctly, empty org returns zeros, response time acceptable

**Acceptance criteria:**

- [ ] All summary stats returned correctly
- [ ] Risk distribution matches actual customer counts
- [ ] MRR totals calculated accurately
- [ ] Trend calculations compare to historical values
- [ ] Response time <500ms for 500 customers
- [ ] Tests cover stats accuracy

#### Phase 3b — Health Score Distribution

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#98](https://github.com/subculture-collective/pulse-score/issues/98) | Implement health score distribution endpoint | medium | `internal/handler/dashboard.go` (modify), `internal/service/dashboard.go` (modify), `internal/repository/health_score.go` (modify) |

**Details:**

1. Add repository method to `internal/repository/health_score.go`:

    ```go
    type ScoreBucket struct {
        Range string `json:"range"` // e.g., "0-10", "11-20"
        Count int    `json:"count"`
    }

    type ScoreDistribution struct {
        Buckets       []ScoreBucket `json:"buckets"`
        RiskBreakdown RiskBreakdown `json:"risk_breakdown"`
        AverageScore  float64       `json:"average_score"`
        MedianScore   float64       `json:"median_score"`
    }

    type RiskBreakdownEntry struct {
        Count   int     `json:"count"`
        Percent float64 `json:"pct"`
    }

    type RiskBreakdown struct {
        Green  RiskBreakdownEntry `json:"green"`
        Yellow RiskBreakdownEntry `json:"yellow"`
        Red    RiskBreakdownEntry `json:"red"`
    }

    func (r *HealthScoreRepository) GetScoreDistribution(ctx context.Context, orgID uuid.UUID) (*ScoreDistribution, error)
    ```

    - SQL for buckets:
      ```sql
      SELECT
        CASE
          WHEN overall_score BETWEEN 0 AND 10 THEN '0-10'
          WHEN overall_score BETWEEN 11 AND 20 THEN '11-20'
          -- ... up to 91-100
        END AS range,
        COUNT(*) as count
      FROM health_scores
      WHERE org_id = $1
      GROUP BY range
      ORDER BY MIN(overall_score)
      ```
    - SQL for risk breakdown: `SELECT risk_level, COUNT(*) FROM health_scores WHERE org_id = $1 GROUP BY risk_level`
    - SQL for average: `SELECT AVG(overall_score) FROM health_scores WHERE org_id = $1`
    - SQL for median: `SELECT PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY overall_score) FROM health_scores WHERE org_id = $1`
    - Can combine into 1-2 queries for efficiency

    Note: Check if `scoring.RiskCategorizer` already provides `GetDistribution` / `GetHistogram` — if so, reuse the existing service instead of adding new repo methods. The existing `GET /api/v1/scoring/risk-distribution` and `GET /api/v1/scoring/histogram` endpoints may already cover this. If so, this issue becomes a thin wrapper or route alias under `/api/v1/dashboard/`.

2. Add service method to `DashboardService`:

    ```go
    func (s *DashboardService) GetScoreDistribution(ctx context.Context, orgID uuid.UUID) (*ScoreDistribution, error)
    ```

3. Add handler method to `DashboardHandler`:

    ```go
    func (h *DashboardHandler) GetScoreDistribution(w http.ResponseWriter, r *http.Request)
    ```

4. Register route:
    ```go
    r.Get("/api/v1/dashboard/score-distribution", dashboardHandler.GetScoreDistribution)
    ```

**Tests:** `internal/handler/dashboard_test.go` (add tests) — buckets cover 0-100 range, counts match actual scores, risk breakdown with percentages, average and median calculated, empty org returns zeros

**Acceptance criteria:**

- [ ] Histogram buckets cover full 0-100 range
- [ ] Counts match actual customer scores
- [ ] Risk breakdown with counts and percentages
- [ ] Average and median calculated
- [ ] Empty org returns zeros (not errors)
- [ ] Tests verify distribution accuracy

---

### Phase 4 — Management Endpoints (parallel)

All four of these can be developed in parallel as they are independent.

#### Phase 4a — Integration Management

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#99](https://github.com/subculture-collective/pulse-score/issues/99) | Implement integration management endpoints | high | `internal/handler/integration.go`, `internal/service/integration.go`, `internal/repository/integration_connection.go` (modify) |

**Details:**

1. Add repository methods to `internal/repository/integration_connection.go`:

    ```go
    func (r *IntegrationConnectionRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]IntegrationConnection, error)
    func (r *IntegrationConnectionRepository) GetCustomerCountBySource(ctx context.Context, orgID uuid.UUID, source string) (int, error)
    ```

2. Create `internal/service/integration.go` — `IntegrationService`:

    ```go
    type IntegrationSummary struct {
        Provider     string     `json:"provider"`
        Status       string     `json:"status"`
        LastSyncAt   *time.Time `json:"last_sync_at"`
        LastSyncError *string   `json:"last_sync_error"`
        CustomerCount int       `json:"customer_count"`
        ConnectedAt  time.Time  `json:"connected_at"`
    }

    type IntegrationStatus struct {
        IntegrationSummary
        ExternalAccountID string  `json:"external_account_id"`
        Scopes            []string `json:"scopes"`
    }

    func (s *IntegrationService) List(ctx context.Context, orgID uuid.UUID) ([]IntegrationSummary, error)
    func (s *IntegrationService) GetStatus(ctx context.Context, orgID uuid.UUID, provider string) (*IntegrationStatus, error)
    func (s *IntegrationService) TriggerSync(ctx context.Context, orgID uuid.UUID, provider string) error
    func (s *IntegrationService) Disconnect(ctx context.Context, orgID uuid.UUID, provider string) error
    ```

    - `List`: query all connections for org, enrich with customer count per source
    - `GetStatus`: get connection details, return 404 if not found
    - `TriggerSync`: validate connection exists and is active, dispatch async sync via `SyncOrchestratorService`
    - `Disconnect`: soft-delete connection, return 404 if not found

    Note: Some integration logic already exists in `StripeOAuthService` (`GetStatus`, `Disconnect`). The new `IntegrationService` should be a higher-level abstraction that delegates to provider-specific services (e.g., `StripeOAuthService`) for actual operations. This keeps the door open for future providers (HubSpot, Intercom).

3. Create `internal/handler/integration.go` — `IntegrationHandler`:

    ```go
    type IntegrationHandler struct {
        integrationService *service.IntegrationService
    }

    func NewIntegrationHandler(is *service.IntegrationService) *IntegrationHandler
    func (h *IntegrationHandler) List(w http.ResponseWriter, r *http.Request)
    func (h *IntegrationHandler) GetStatus(w http.ResponseWriter, r *http.Request)
    func (h *IntegrationHandler) TriggerSync(w http.ResponseWriter, r *http.Request)
    func (h *IntegrationHandler) Disconnect(w http.ResponseWriter, r *http.Request)
    ```

    - `List`: GET, all members can access
    - `GetStatus`: GET with `chi.URLParam(r, "provider")`
    - `TriggerSync`: POST, admin+ only (wrap with `RequireRole("admin")`)
    - `Disconnect`: DELETE, admin+ only

4. Register routes:
    ```go
    r.Get("/api/v1/integrations", integrationHandler.List)
    r.Route("/api/v1/integrations/{provider}", func(r chi.Router) {
        r.Get("/status", integrationHandler.GetStatus)
        r.With(middleware.RequireRole("admin")).Post("/sync", integrationHandler.TriggerSync)
        r.With(middleware.RequireRole("admin")).Delete("/", integrationHandler.Disconnect)
    })
    ```

    Note: Ensure these routes don't conflict with existing `/api/v1/integrations/stripe/*` routes. Consider nesting under the same route group or refactoring the existing Stripe routes to use the generic `{provider}` pattern.

5. Wire dependencies in `main.go`

**Tests:** `internal/handler/integration_test.go` — list returns connected integrations, status returns detail for valid provider, 404 for disconnected provider, sync triggers for active connection, disconnect removes connection, RBAC enforcement (member cannot sync/disconnect)

**Acceptance criteria:**

- [ ] List returns all connections with summarized status
- [ ] Status endpoint returns detailed sync information
- [ ] Manual sync triggers background sync job
- [ ] Disconnect removes connection (with confirmation)
- [ ] RBAC enforced for write operations
- [ ] Tests cover all CRUD operations and RBAC

#### Phase 4b — Organization Settings

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#100](https://github.com/subculture-collective/pulse-score/issues/100) | Implement organization settings endpoints | medium | `internal/handler/organization.go` (modify), `internal/service/organization.go` (modify), `internal/repository/organization.go` (modify) |

**Depends on:** Existing `OrganizationService`, `OrganizationRepository`

**Details:**

1. Add repository methods to `internal/repository/organization.go`:

    ```go
    func (r *OrganizationRepository) GetWithStats(ctx context.Context, orgID uuid.UUID) (*OrganizationWithStats, error)
    func (r *OrganizationRepository) Update(ctx context.Context, orgID uuid.UUID, name string) error
    func (r *OrganizationRepository) CountMembers(ctx context.Context, orgID uuid.UUID) (int, error)

    type OrganizationWithStats struct {
        Organization
        MemberCount   int `json:"member_count"`
        CustomerCount int `json:"customer_count"`
    }
    ```

    - `GetWithStats`: JOIN with COUNT of users and customers for org
    - `Update`: UPDATE name + auto-generate slug from new name + updated_at

2. Add service methods to `OrganizationService`:

    ```go
    func (s *OrganizationService) GetCurrent(ctx context.Context, orgID uuid.UUID) (*OrganizationWithStats, error)
    func (s *OrganizationService) UpdateSettings(ctx context.Context, orgID uuid.UUID, name string) error
    ```

    - `GetCurrent`: retrieve org with member/customer counts
    - `UpdateSettings`: validate name (length 2-100, no special chars), generate slug, check slug uniqueness, update

3. Add handler methods to existing `OrganizationHandler` in `internal/handler/organization.go`:

    ```go
    func (h *OrganizationHandler) GetCurrent(w http.ResponseWriter, r *http.Request)
    func (h *OrganizationHandler) UpdateCurrent(w http.ResponseWriter, r *http.Request)
    ```

    - `GetCurrent`: extract org_id from context, return org with stats
    - `UpdateCurrent`: parse `{name: "New Name"}`, validate, update

4. Register routes:
    ```go
    r.Get("/api/v1/organizations/current", orgHandler.GetCurrent)
    r.With(middleware.RequireRole("admin")).Patch("/api/v1/organizations/current", orgHandler.UpdateCurrent)
    ```

**Tests:** `internal/handler/organization_test.go` — get returns org with stats, update changes name and slug, RBAC (member can read but not update), validation errors for bad names, slug uniqueness enforcement

**Acceptance criteria:**

- [ ] GET returns current org with summary stats
- [ ] PATCH updates allowed fields
- [ ] Slug updates when name changes
- [ ] RBAC enforced (members read, admin/owner write)
- [ ] Tests cover get, update, RBAC

#### Phase 4c — User Management

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#101](https://github.com/subculture-collective/pulse-score/issues/101) | Implement user management endpoints | medium | `internal/handler/member.go`, `internal/service/member.go`, `internal/repository/user.go` (modify), `internal/repository/organization.go` (modify) |

**Details:**

1. Add repository methods:

    **In `internal/repository/organization.go`:**
    ```go
    type OrgMember struct {
        UserID    uuid.UUID `json:"user_id"`
        Email     string    `json:"email"`
        FirstName string    `json:"first_name"`
        LastName  string    `json:"last_name"`
        AvatarURL *string   `json:"avatar_url"`
        Role      string    `json:"role"`
        JoinedAt  time.Time `json:"joined_at"`
    }

    func (r *OrganizationRepository) ListMembers(ctx context.Context, orgID uuid.UUID) ([]OrgMember, error)
    func (r *OrganizationRepository) GetMemberRole(ctx context.Context, orgID, userID uuid.UUID) (string, error)
    func (r *OrganizationRepository) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error
    func (r *OrganizationRepository) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error
    func (r *OrganizationRepository) CountOwners(ctx context.Context, orgID uuid.UUID) (int, error)
    ```

    - `ListMembers`: JOIN users + user_organizations for org
    - `UpdateMemberRole`: UPDATE user_organizations SET role = $1 WHERE org_id = $2 AND user_id = $3
    - `RemoveMember`: DELETE FROM user_organizations WHERE org_id = $1 AND user_id = $2
    - `CountOwners`: SELECT COUNT(*) WHERE role = 'owner' AND org_id = $1

2. Create `internal/service/member.go` — `MemberService`:

    ```go
    func (s *MemberService) ListMembers(ctx context.Context, orgID uuid.UUID) ([]OrgMember, error)
    func (s *MemberService) UpdateRole(ctx context.Context, orgID, userID, actingUserID uuid.UUID, newRole string) error
    func (s *MemberService) RemoveMember(ctx context.Context, orgID, userID, actingUserID uuid.UUID) error
    ```

    - `UpdateRole` validations:
      - Cannot demote yourself → `ForbiddenError`
      - Role must be in {"member", "admin"} (owner cannot be assigned via API)
      - Only owner can promote to admin
      - Validate user is member of org → `NotFoundError`
    - `RemoveMember` validations:
      - Cannot remove yourself → `ForbiddenError`
      - Cannot remove last owner → `ValidationError`
      - Validate user is member of org → `NotFoundError`

3. Create `internal/handler/member.go` — `MemberHandler`:

    ```go
    type MemberHandler struct {
        memberService *service.MemberService
    }

    func NewMemberHandler(ms *service.MemberService) *MemberHandler
    func (h *MemberHandler) List(w http.ResponseWriter, r *http.Request)
    func (h *MemberHandler) UpdateRole(w http.ResponseWriter, r *http.Request)
    func (h *MemberHandler) Remove(w http.ResponseWriter, r *http.Request)
    ```

    - `List`: return members array
    - `UpdateRole`: parse `{role: "admin"}` from body, parse `user_id` from URL
    - `Remove`: parse `user_id` from URL, return 204 on success

4. Register routes:
    ```go
    r.Route("/api/v1/organizations/current/members", func(r chi.Router) {
        r.Use(middleware.RequireRole("admin"))
        r.Get("/", memberHandler.List)
        r.Patch("/{user_id}", memberHandler.UpdateRole)
        r.Delete("/{user_id}", memberHandler.Remove)
    })
    ```

5. Wire dependencies in `main.go`

**Tests:** `internal/handler/member_test.go` — list returns all members, update role works, cannot demote self, cannot remove self, cannot remove last owner, only owner can promote to admin, RBAC (member cannot access), remove unlinks user

**Acceptance criteria:**

- [ ] List returns all members with roles
- [ ] Role update works with proper validation
- [ ] Cannot demote yourself or remove yourself
- [ ] Cannot remove last owner
- [ ] Only owner can promote to admin
- [ ] Remove unlinks user from org
- [ ] Tests cover all operations and edge cases

#### Phase 4d — Alert Rules CRUD

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#102](https://github.com/subculture-collective/pulse-score/issues/102) | Implement alert rules CRUD endpoints | high | `internal/handler/alert_rule.go`, `internal/service/alert_rule.go`, `internal/repository/alert_rule.go` |

**Details:**

1. Create `internal/repository/alert_rule.go` — `AlertRuleRepository`:

    ```go
    type AlertRule struct {
        ID          uuid.UUID      `json:"id"`
        OrgID       uuid.UUID      `json:"org_id"`
        Name        string         `json:"name"`
        Description *string        `json:"description"`
        TriggerType string         `json:"trigger_type"`  // "score_drop", "risk_change", "payment_failed"
        Conditions  map[string]any `json:"conditions"`    // JSONB
        Channel     string         `json:"channel"`       // "email"
        Recipients  []string       `json:"recipients"`    // JSONB array of emails
        IsActive    bool           `json:"is_active"`
        CreatedBy   uuid.UUID      `json:"created_by"`
        CreatedAt   time.Time      `json:"created_at"`
        UpdatedAt   time.Time      `json:"updated_at"`
    }

    func NewAlertRuleRepository(pool *pgxpool.Pool) *AlertRuleRepository
    func (r *AlertRuleRepository) List(ctx context.Context, orgID uuid.UUID) ([]AlertRule, error)
    func (r *AlertRuleRepository) GetByID(ctx context.Context, id, orgID uuid.UUID) (*AlertRule, error)
    func (r *AlertRuleRepository) Create(ctx context.Context, rule *AlertRule) error
    func (r *AlertRuleRepository) Update(ctx context.Context, rule *AlertRule) error
    func (r *AlertRuleRepository) Delete(ctx context.Context, id, orgID uuid.UUID) error
    ```

    - `List`: SELECT * WHERE org_id = $1 ORDER BY created_at DESC
    - `GetByID`: SELECT * WHERE id = $1 AND org_id = $2 (tenant isolation)
    - `Create`: INSERT with all fields
    - `Update`: UPDATE SET name, description, trigger_type, conditions, channel, recipients, is_active, updated_at WHERE id = $1 AND org_id = $2
    - `Delete`: DELETE WHERE id = $1 AND org_id = $2

2. Create `internal/service/alert_rule.go` — `AlertRuleService`:

    ```go
    type CreateAlertRuleRequest struct {
        Name        string         `json:"name"`
        Description *string        `json:"description"`
        TriggerType string         `json:"trigger_type"`
        Conditions  map[string]any `json:"conditions"`
        Channel     string         `json:"channel"`
        Recipients  []string       `json:"recipients"`
    }

    type UpdateAlertRuleRequest struct {
        Name        *string        `json:"name"`
        Description *string        `json:"description"`
        TriggerType *string        `json:"trigger_type"`
        Conditions  map[string]any `json:"conditions"`
        Channel     *string        `json:"channel"`
        Recipients  []string       `json:"recipients"`
        IsActive    *bool          `json:"is_active"`
    }

    func (s *AlertRuleService) List(ctx context.Context, orgID uuid.UUID) ([]AlertRule, error)
    func (s *AlertRuleService) GetByID(ctx context.Context, id, orgID uuid.UUID) (*AlertRule, error)
    func (s *AlertRuleService) Create(ctx context.Context, orgID, userID uuid.UUID, req CreateAlertRuleRequest) (*AlertRule, error)
    func (s *AlertRuleService) Update(ctx context.Context, id, orgID uuid.UUID, req UpdateAlertRuleRequest) (*AlertRule, error)
    func (s *AlertRuleService) Delete(ctx context.Context, id, orgID uuid.UUID) error
    ```

    - **Validation rules:**
      - `name`: required, 1-200 chars
      - `trigger_type`: must be one of `score_drop`, `risk_change`, `payment_failed`
      - `conditions`: validated per trigger type:
        - `score_drop`: requires `threshold` (int 0-100), `direction` ("below")
        - `risk_change`: requires `from` and/or `to` (valid risk levels)
        - `payment_failed`: optional `min_amount_cents` (positive int)
      - `channel`: must be `email` (for MVP)
      - `recipients`: non-empty, each must be valid email format (use `net/mail.ParseAddress`)
    - Return `ValidationError` for any validation failure

3. Create `internal/handler/alert_rule.go` — `AlertRuleHandler`:

    ```go
    type AlertRuleHandler struct {
        alertRuleService *service.AlertRuleService
    }

    func NewAlertRuleHandler(as *service.AlertRuleService) *AlertRuleHandler
    func (h *AlertRuleHandler) List(w http.ResponseWriter, r *http.Request)
    func (h *AlertRuleHandler) Get(w http.ResponseWriter, r *http.Request)
    func (h *AlertRuleHandler) Create(w http.ResponseWriter, r *http.Request)
    func (h *AlertRuleHandler) Update(w http.ResponseWriter, r *http.Request)
    func (h *AlertRuleHandler) Delete(w http.ResponseWriter, r *http.Request)
    ```

    - `List`: GET, all members can read
    - `Get`: GET with `{id}` param, all members can read
    - `Create`: POST, admin+ only, return 201 + created rule
    - `Update`: PATCH with `{id}` param, admin+ only, return updated rule
    - `Delete`: DELETE with `{id}` param, admin+ only, return 204

4. Register routes:
    ```go
    r.Route("/api/v1/alert-rules", func(r chi.Router) {
        r.Get("/", alertRuleHandler.List)
        r.Get("/{id}", alertRuleHandler.Get)
        r.With(middleware.RequireRole("admin")).Post("/", alertRuleHandler.Create)
        r.With(middleware.RequireRole("admin")).Patch("/{id}", alertRuleHandler.Update)
        r.With(middleware.RequireRole("admin")).Delete("/{id}", alertRuleHandler.Delete)
    })
    ```

5. Wire dependencies in `main.go`

**Tests:** `internal/handler/alert_rule_test.go` — full CRUD cycle, trigger type validation, conditions schema validation per type, recipient email validation, toggle active/inactive, RBAC (member can read but not create/update/delete), 404 for non-existent rule, cross-tenant isolation

**Acceptance criteria:**

- [ ] Full CRUD operations work
- [ ] Trigger type and conditions validated
- [ ] Recipients validated (valid emails)
- [ ] Toggle active/inactive
- [ ] RBAC enforced
- [ ] Tests cover CRUD, validation, RBAC

---

### Phase 5 — OpenAPI Documentation (after all endpoints)

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#103](https://github.com/subculture-collective/pulse-score/issues/103) | Generate OpenAPI/Swagger documentation | medium | `docs/openapi.yaml`, `internal/handler/docs.go`, `cmd/api/main.go` (modify) |

**Depends on:** All other issues (#94–#102)

**Details:**

1. **Choose approach:** Hand-crafted `openapi.yaml` (recommended for quality control) vs `swaggo/swag` annotations.

    Recommended: hand-crafted `docs/openapi.yaml` because:
    - Full control over descriptions and examples
    - No annotation clutter in handler code
    - Easier to review in PRs
    - CI validates spec matches handlers

2. Create `docs/openapi.yaml` — OpenAPI 3.0 specification:

    ```yaml
    openapi: "3.0.3"
    info:
      title: PulseScore API
      version: "1.0.0"
      description: Customer health scoring platform API
    servers:
      - url: /api/v1
    security:
      - bearerAuth: []
    ```

    - Document all endpoints from Epic 6:
      - `/customers` — GET (list with pagination/sort/filter)
      - `/customers/{id}` — GET (detail with score breakdown)
      - `/customers/{id}/events` — GET (timeline with pagination)
      - `/dashboard/summary` — GET (aggregate stats)
      - `/dashboard/score-distribution` — GET (histogram data)
      - `/integrations` — GET (list connections)
      - `/integrations/{provider}/status` — GET (detailed status)
      - `/integrations/{provider}/sync` — POST (trigger sync)
      - `/integrations/{provider}` — DELETE (disconnect)
      - `/organizations/current` — GET, PATCH
      - `/organizations/current/members` — GET
      - `/organizations/current/members/{user_id}` — PATCH, DELETE
      - `/alert-rules` — GET, POST
      - `/alert-rules/{id}` — GET, PATCH, DELETE
    - Also document existing auth endpoints
    - Define schemas for all request/response objects
    - Include authentication documentation (Bearer JWT)
    - Document error response schema: `{error: string}`
    - Add examples for key endpoints

3. Serve Swagger UI:

    **Option A — `swaggo/http-swagger`:**
    ```go
    import httpSwagger "github.com/swaggo/http-swagger/v2"

    r.Get("/api/docs/*", httpSwagger.Handler(
        httpSwagger.URL("/api/docs/openapi.json"),
    ))
    ```

    **Option B — Embed static swagger-ui files:**
    - Download swagger-ui dist, embed with `go:embed`
    - Serve at `/api/docs/`

    Recommended: Option A for simplicity.

4. Serve the spec:
    ```go
    r.Get("/api/docs/openapi.json", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "docs/openapi.yaml")
    })
    ```

5. Add CI validation: `openapi-generator validate -i docs/openapi.yaml` or `spectral lint docs/openapi.yaml`

6. Add to `go.mod`: `github.com/swaggo/http-swagger/v2` (if using Option A)

**Tests:** Validate OpenAPI spec is valid 3.0, all defined paths have matching handlers, spec serves at `/api/docs`

**Acceptance criteria:**

- [ ] OpenAPI 3.0 spec generated/maintained
- [ ] All endpoints documented with request/response schemas
- [ ] Swagger UI accessible at /api/docs
- [ ] Authentication documented (Bearer token)
- [ ] Error response schemas documented
- [ ] Examples included for key endpoints
- [ ] Tests validate spec is valid OpenAPI 3.0

---

## Implementation Checklist

### Files to Create

| File | Phase | Issue |
|------|-------|-------|
| `internal/handler/customer.go` | 1 | #94, #95, #96 |
| `internal/service/customer.go` | 1 | #94, #95, #96 |
| `internal/handler/dashboard.go` | 3 | #97, #98 |
| `internal/service/dashboard.go` | 3 | #97, #98 |
| `internal/handler/integration.go` | 4 | #99 |
| `internal/service/integration.go` | 4 | #99 |
| `internal/handler/member.go` | 4 | #101 |
| `internal/service/member.go` | 4 | #101 |
| `internal/handler/alert_rule.go` | 4 | #102 |
| `internal/service/alert_rule.go` | 4 | #102 |
| `internal/repository/alert_rule.go` | 4 | #102 |
| `docs/openapi.yaml` | 5 | #103 |
| `internal/handler/docs.go` | 5 | #103 |

### Files to Modify

| File | Phase | Issue | Changes |
|------|-------|-------|---------|
| `internal/repository/customer.go` | 1, 2, 3 | #94, #95, #97 | `ListWithScores`, `GetByIDAndOrg`, `CountByOrg`, `TotalMRRByOrg` |
| `internal/repository/customer_event.go` | 2 | #96 | `ListPaginated` |
| `internal/repository/health_score.go` | 3 | #97, #98 | `GetRiskDistribution`, `GetAverageScore`, `GetAverageScoreAt`, `CountAtRiskAt`, `GetScoreDistribution` |
| `internal/repository/integration_connection.go` | 4 | #99 | `ListByOrg`, `GetCustomerCountBySource` |
| `internal/repository/organization.go` | 4 | #100, #101 | `GetWithStats`, `Update`, `ListMembers`, `GetMemberRole`, `UpdateMemberRole`, `RemoveMember`, `CountOwners`, `CountMembers` |
| `internal/handler/organization.go` | 4 | #100 | `GetCurrent`, `UpdateCurrent` |
| `internal/service/organization.go` | 4 | #100 | `GetCurrent`, `UpdateSettings` |
| `cmd/api/main.go` | All | All | Route registration + dependency wiring |

### Test Files to Create

| File | Phase |
|------|-------|
| `internal/handler/customer_test.go` | 1, 2 |
| `internal/handler/dashboard_test.go` | 3 |
| `internal/handler/integration_test.go` | 4 |
| `internal/handler/organization_test.go` | 4 |
| `internal/handler/member_test.go` | 4 |
| `internal/handler/alert_rule_test.go` | 4 |

## Cross-Cutting Concerns

### Pagination Convention

Establish a shared pagination pattern used across all list endpoints:

```go
// Could be placed in internal/handler/pagination.go or a shared helper

type PaginationParams struct {
    Page    int
    PerPage int
}

type PaginationMeta struct {
    Page       int `json:"page"`
    PerPage    int `json:"per_page"`
    Total      int `json:"total"`
    TotalPages int `json:"total_pages"`
}

func ParsePagination(r *http.Request) PaginationParams {
    // Parse page (default 1), per_page (default 25, max 100)
}
```

Used by: #94 (customers), #96 (events), #102 (alert rules if list grows large)

### Route Registration Order

When modifying `cmd/api/main.go`, register new routes inside the existing protected routes group:

```go
r.Group(func(r chi.Router) {
    r.Use(middleware.JWTAuth(jwtService))
    r.Use(middleware.TenantIsolation(orgRepo))

    // Existing routes...

    // Epic 6 — Customer endpoints
    r.Get("/api/v1/customers", customerHandler.List)
    r.Get("/api/v1/customers/{id}", customerHandler.GetDetail)
    r.Get("/api/v1/customers/{id}/events", customerHandler.ListEvents)

    // Epic 6 — Dashboard endpoints
    r.Get("/api/v1/dashboard/summary", dashboardHandler.GetSummary)
    r.Get("/api/v1/dashboard/score-distribution", dashboardHandler.GetScoreDistribution)

    // Epic 6 — Integration management
    r.Get("/api/v1/integrations", integrationHandler.List)
    r.Route("/api/v1/integrations/{provider}", func(r chi.Router) {
        r.Get("/status", integrationHandler.GetStatus)
        r.With(middleware.RequireRole("admin")).Post("/sync", integrationHandler.TriggerSync)
        r.With(middleware.RequireRole("admin")).Delete("/", integrationHandler.Disconnect)
    })

    // Epic 6 — Organization settings
    r.Get("/api/v1/organizations/current", orgHandler.GetCurrent)
    r.With(middleware.RequireRole("admin")).Patch("/api/v1/organizations/current", orgHandler.UpdateCurrent)

    // Epic 6 — User/Member management
    r.Route("/api/v1/organizations/current/members", func(r chi.Router) {
        r.Use(middleware.RequireRole("admin"))
        r.Get("/", memberHandler.List)
        r.Patch("/{user_id}", memberHandler.UpdateRole)
        r.Delete("/{user_id}", memberHandler.Remove)
    })

    // Epic 6 — Alert rules
    r.Route("/api/v1/alert-rules", func(r chi.Router) {
        r.Get("/", alertRuleHandler.List)
        r.Get("/{id}", alertRuleHandler.Get)
        r.With(middleware.RequireRole("admin")).Post("/", alertRuleHandler.Create)
        r.With(middleware.RequireRole("admin")).Patch("/{id}", alertRuleHandler.Update)
        r.With(middleware.RequireRole("admin")).Delete("/{id}", alertRuleHandler.Delete)
    })
})
```

### SQL Injection Prevention

All queries MUST use parameterized queries (`$1`, `$2`, etc.). Dynamic WHERE clauses and ORDER BY must be built from allowlists — never interpolate user-provided sort/filter values into SQL strings. Example:

```go
var sortColumns = map[string]string{
    "name":      "c.name",
    "mrr":       "c.mrr_cents",
    "score":     "hs.overall_score",
    "last_seen": "c.last_seen_at",
}

col, ok := sortColumns[params.Sort]
if !ok {
    col = "c.name" // default
}
query += " ORDER BY " + col // safe — from allowlist
```

### Tenant Isolation

All queries MUST include `org_id = $N` in the WHERE clause. This is the primary multi-tenancy enforcement at the data layer, complementing the middleware-level check. Never trust the absence of middleware — always filter by org_id in SQL.
