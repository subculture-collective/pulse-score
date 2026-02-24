# Execution Plan: Epic 5 — Health Scoring Engine (#4)

## Overview

**Epic:** [#4 — Health Scoring Engine](https://github.com/subculture-collective/pulse-score/issues/4)
**Sub-issues:** #83–#93 (11 issues)
**Scope:** Build the core health scoring engine that calculates a 0-100 health score per customer using weighted factors (payment recency, MRR trend, failed payments, support tickets, engagement). Scores are categorized into risk levels (green/yellow/red) with configurable weights/thresholds per org.

## Current State

The following foundations are already in place:

- **Database migrations** for `health_scores` + `health_score_history` (migration 000007), `customers` (000005), `customer_events` (000006)
- **Existing services:** `PaymentRecencyService` (produces 0-100 score), `PaymentHealthService` (produces 0-100 score with failure rates), `MRRService` (calculates MRR with event tracking)
- **Repository layer:** `CustomerRepository`, `CustomerEventRepository`, `StripePaymentRepository`, `StripeSubscriptionRepository`
- **Auth/RBAC middleware:** JWT auth, tenant isolation, `RequireRole("admin")` middleware
- **Handler patterns:** thin wrappers calling services, JSON in/out, `service.ValidationError` handling

### Existing Scoring-Adjacent Services (reuse these)

| Service | File | Output | Reuse As |
|---------|------|--------|----------|
| `PaymentRecencyService` | `internal/service/payment_recency.go` | `PaymentRecencyResult{Score: 0-100}` | Payment recency scoring factor input |
| `PaymentHealthService` | `internal/service/payment_health.go` | `PaymentHealthResult{Score: 0-100}` | Failed payment scoring factor input |
| `MRRService` | `internal/service/mrr.go` | MRR calculation + events | MRR trend factor input |

### Existing Tables (already migrated)

| Table | Migration | Key Columns |
|-------|-----------|-------------|
| `health_scores` | 000007 | org_id, customer_id, overall_score (0-100), risk_level (green/yellow/red), factors (JSONB), calculated_at |
| `health_score_history` | 000007 | Same as health_scores (append-only history) |
| `customers` | 000005 | org_id, external_id, source, mrr_cents, metadata (JSONB) |
| `customer_events` | 000006 | org_id, customer_id, event_type, occurred_at, data (JSONB) |

## Dependency Graph

```
#83 Scoring Config Model + Migration
 ├──► #84 Payment Recency Factor ─────┐
 ├──► #85 MRR Trend Factor ───────────┤
 ├──► #86 Failed Payment Factor ──────┼──► #89 Weighted Aggregation ──► #90 Recalculation Scheduler ──► #91 Change Detection + History
 ├──► #87 Support Ticket Factor ──────┤                              │
 └──► #88 Engagement Factor ──────────┘                              ├──► #92 Risk Categorization
                                                                     └──► #93 Score Config Admin API
```

## Execution Phases

Issues are grouped into phases based on dependency chains. Issues within the same phase can be worked on in parallel.

---

### Phase 1 — Scoring Configuration Model & Migration

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#83](https://github.com/subculture-collective/pulse-score/issues/83) | Create scoring configuration model and migration | critical | new migration, `internal/repository/scoring_config.go` |

**Details:**

1. Create migration `000013_create_scoring_configs.up.sql`:
    ```sql
    CREATE TABLE scoring_configs (
        id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
        weights     JSONB NOT NULL DEFAULT '{"payment_recency": 0.3, "mrr_trend": 0.2, "failed_payments": 0.2, "support_tickets": 0.15, "engagement": 0.15}',
        thresholds  JSONB NOT NULL DEFAULT '{"green": 70, "yellow": 40}',
        created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        CONSTRAINT scoring_configs_org_unique UNIQUE (org_id)
    );
    CREATE INDEX idx_scoring_configs_org_id ON scoring_configs(org_id);
    ```
2. Create migration `000013_create_scoring_configs.down.sql`:
    ```sql
    DROP TABLE IF EXISTS scoring_configs;
    ```
3. Create `internal/repository/scoring_config.go` — `ScoringConfigRepository`:
    - Struct: `ScoringConfig{ID, OrgID, Weights map[string]float64, Thresholds map[string]int, CreatedAt, UpdatedAt}`
    - `GetByOrgID(ctx, orgID) (*ScoringConfig, error)` — returns config or nil
    - `Upsert(ctx, config *ScoringConfig) error` — INSERT ON CONFLICT UPDATE
    - `CreateDefault(ctx, orgID) (*ScoringConfig, error)` — create with default weights/thresholds
4. Add validation functions:
    - `ValidateWeights(weights map[string]float64) error` — must sum to 1.0, each 0.0-1.0
    - `ValidateThresholds(thresholds map[string]int) error` — green > yellow > 0
5. Hook into organization creation flow: when a new org is created, insert default scoring config (add to `OrganizationService.Create` or seed on first access)

**Tests:** `internal/repository/scoring_config_test.go` — weight validation (sums to 1.0, invalid sums, out-of-range values), threshold validation (green > yellow, boundary cases), upsert idempotency, default creation

**Acceptance criteria:**

- [ ] Migration creates scoring_configs table
- [ ] One config per org (unique constraint)
- [ ] Default config created with org
- [ ] Weights validated to sum to 1.0
- [ ] Thresholds validated (green > yellow)
- [ ] Down migration drops table cleanly
- [ ] Tests verify validation logic

---

### Phase 2 — Individual Scoring Factors (parallel)

All five scoring factors can be implemented in parallel. They share a common interface and produce normalized 0.0-1.0 scores.

**Common Interface** — Create `internal/service/scoring/factor.go`:

```go
package scoring

type FactorResult struct {
    Name  string
    Score *float64 // nil = factor unavailable (skip in aggregation)
}

type ScoreFactor interface {
    Name() string
    Calculate(ctx context.Context, customerID, orgID uuid.UUID) (*FactorResult, error)
}
```

#### Phase 2a — Payment Recency Factor

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#84](https://github.com/subculture-collective/pulse-score/issues/84) | Implement payment recency scoring factor | critical | `internal/service/scoring/factor.go`, `internal/service/scoring/payment_recency.go` |

**Depends on:** Phase 1 (#83), existing `PaymentRecencyService`

**Details:**

1. Create `internal/service/scoring/payment_recency.go` — `PaymentRecencyFactor`:
    - Wraps existing `PaymentRecencyService` which already produces a 0-100 score
    - Normalize: divide service score by 100 → 0.0-1.0
    - Configurable decay curve support:
      - Monthly: 100 at day 0, decays to ~50 at day 45, ~0 at day 90
      - Annual: 100 at day 0, decays more slowly (full until month 11)
    - No payment history: return 0.5 (neutral)
    - Implements `ScoreFactor` interface

**Tests:** `internal/service/scoring/payment_recency_test.go` — monthly subscription scores, annual subscription scores, no history returns 0.5, decay curve correctness, edge cases (payment on exact boundary)

**Acceptance criteria:**

- [ ] Score calculated correctly for monthly subscriptions
- [ ] Score calculated correctly for annual subscriptions
- [ ] Decay curve produces intuitive results
- [ ] No payment history returns neutral score (0.5)
- [ ] Score normalized to 0.0-1.0 range
- [ ] Tests cover monthly, annual, no history, edge cases

#### Phase 2b — MRR Trend Factor

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#85](https://github.com/subculture-collective/pulse-score/issues/85) | Implement MRR trend scoring factor | critical | `internal/service/scoring/mrr_trend.go` |

**Depends on:** Phase 1 (#83), existing `MRRService`

**Details:**

1. Create `internal/service/scoring/mrr_trend.go` — `MRRTrendFactor`:
    - Query `customer_events` for MRR change events over 30d, 60d, 90d windows
    - Compare current `mrr_cents` to historical values at T-30d, T-60d, T-90d
    - Scoring bands:
      - Growing (>5% increase): 0.8-1.0
      - Stable (-5% to +5%): 0.5-0.7
      - Declining (>5% decrease): 0.0-0.4
    - Time weighting: 30d trend gets 50% weight, 60d trend 30%, 90d trend 20%
    - No historical data: return 0.5 (neutral)
    - Implements `ScoreFactor` interface
2. May need helper query in `CustomerEventRepository`: `GetMRREventsInWindow(ctx, customerID, orgID, since time.Time) ([]CustomerEvent, error)`

**Tests:** `internal/service/scoring/mrr_trend_test.go` — growing MRR (high score), declining MRR (low score), stable MRR (mid score), recent changes weighted more heavily, no history returns 0.5

**Acceptance criteria:**

- [ ] Growing MRR produces high score
- [ ] Declining MRR produces low score
- [ ] Stable MRR produces mid-range score
- [ ] Recent changes weighted more than older changes
- [ ] No historical data returns neutral score
- [ ] Tests cover growth, decline, stable, no history

#### Phase 2c — Failed Payment History Factor

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#86](https://github.com/subculture-collective/pulse-score/issues/86) | Implement failed payment history scoring factor | critical | `internal/service/scoring/failed_payments.go` |

**Depends on:** Phase 1 (#83), existing `PaymentHealthService`

**Details:**

1. Create `internal/service/scoring/failed_payments.go` — `FailedPaymentsFactor`:
    - Wraps existing `PaymentHealthService` which produces a 0-100 score
    - Additional refinements on top of base score:
      - No failures in 90d: 1.0
      - 1 failure, resolved: 0.7-0.8
      - Multiple failures: scale down proportionally
      - Unresolved failure: significant penalty (0.0-0.3)
      - Recent failures weighted more than older ones
      - Consecutive failures: extra penalty
    - Implements `ScoreFactor` interface
2. May need helper query: `GetPaymentFailureEvents(ctx, customerID, orgID, since time.Time) ([]CustomerEvent, error)`

**Tests:** `internal/service/scoring/failed_payments_test.go` — no failures (1.0), single resolved (0.7-0.8), multiple failures (proportional), unresolved (heavy penalty), consecutive failures (extra penalty), recency weighting

**Acceptance criteria:**

- [ ] No failures = perfect score
- [ ] Single resolved failure = minor penalty
- [ ] Multiple failures = proportional penalty
- [ ] Unresolved failure = significant penalty
- [ ] Recent failures weighted more heavily
- [ ] Tests cover all failure patterns

#### Phase 2d — Support Ticket Volume Factor

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#87](https://github.com/subculture-collective/pulse-score/issues/87) | Implement support ticket volume scoring factor | high | `internal/service/scoring/support_tickets.go` |

**Depends on:** Phase 1 (#83), `CustomerEventRepository`

**Details:**

1. Create `internal/service/scoring/support_tickets.go` — `SupportTicketsFactor`:
    - Query `customer_events` for ticket events (type: `ticket.opened`, `ticket.resolved`) in 90d window
    - Normalize customer ticket count against org median:
      - Below org average: 0.7-1.0
      - Average: 0.4-0.6
      - Above average: 0.0-0.3
    - Open/unresolved tickets penalized more than resolved
    - Increasing ticket trend lowers score further
    - **Graceful degradation:** if no ticket data exists (integration not connected), return `nil` (factor skipped in aggregation)
    - Implements `ScoreFactor` interface
2. Need helper query: `CountEventsByTypeForOrg(ctx, orgID, eventType string, since time.Time) (map[uuid.UUID]int, error)` — returns ticket counts per customer for normalization

**Tests:** `internal/service/scoring/support_tickets_test.go` — below average tickets (high score), above average (low score), open vs resolved weighting, no data returns nil, edge case with few customers in org

**Acceptance criteria:**

- [ ] Score normalized against org-wide ticket volume
- [ ] Open tickets penalized more than resolved
- [ ] Increasing trend produces lower score
- [ ] No ticket data returns nil (factor skipped in aggregation)
- [ ] Normalization handles orgs with few customers
- [ ] Tests cover various ticket patterns and missing data

#### Phase 2e — Engagement/Activity Factor

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#88](https://github.com/subculture-collective/pulse-score/issues/88) | Implement engagement/activity scoring factor | high | `internal/service/scoring/engagement.go` |

**Depends on:** Phase 1 (#83), `CustomerEventRepository`

**Details:**

1. Create `internal/service/scoring/engagement.go` — `EngagementFactor`:
    - Query `customer_events` for activity events (types: `login`, `feature_use`, `api_call`, etc.) in 30d rolling window
    - Compare customer activity count to org median:
      - Above median: high score (0.7-1.0)
      - At median: mid score (0.4-0.6)
      - Below median: low score (0.0-0.3)
    - Recency bonus: recent activity in last 7d weighted higher
    - Trend: increasing activity is positive signal
    - **Graceful degradation:** no activity data → return `nil` (factor skipped)
    - Implements `ScoreFactor` interface
2. Reuse `CountEventsByTypeForOrg` query pattern from support tickets factor

**Tests:** `internal/service/scoring/engagement_test.go` — active customer (high score), inactive (low score), relative to org baseline, no data returns nil, recency bonus working

**Acceptance criteria:**

- [ ] Active customers score high
- [ ] Inactive customers score low
- [ ] Score relative to org baseline
- [ ] Missing data returns nil (factor skipped)
- [ ] Recent activity weighted more
- [ ] Tests cover active, inactive, no data cases

---

### Phase 3 — Weighted Score Aggregation

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#89](https://github.com/subculture-collective/pulse-score/issues/89) | Implement weighted score aggregation service | critical | `internal/service/scoring/aggregator.go`, `internal/repository/health_score.go` |

**Depends on:** Phase 1 (#83), Phase 2 (all five factors)

**Details:**

1. Create `internal/service/scoring/aggregator.go` — `ScoreAggregator`:
    - Inject all five `ScoreFactor` implementations + `ScoringConfigRepository`
    - `Calculate(ctx, customerID, orgID uuid.UUID) (*HealthScoreResult, error)`:
      1. Load scoring config for org (weights + thresholds)
      2. Call each factor's `Calculate()`, collect results
      3. Separate present factors (non-nil) from absent factors (nil)
      4. Redistribute absent factor weights proportionally to present factors
      5. Compute weighted sum: `overall = sum(factor_score * adjusted_weight) * 100`
      6. Round to integer 0-100
      7. Assign risk level based on thresholds: `>= green → "green"`, `>= yellow → "yellow"`, else `"red"`
      8. Return `HealthScoreResult{OverallScore, RiskLevel, Factors, CalculatedAt}`
    - Edge case: all factors nil → return error (cannot score)
    - Result struct:
      ```go
      type HealthScoreResult struct {
          CustomerID   uuid.UUID
          OrgID        uuid.UUID
          OverallScore int                // 0-100
          RiskLevel    string             // "green", "yellow", "red"
          Factors      map[string]float64 // factor name → normalized score
          CalculatedAt time.Time
      }
      ```

2. Create `internal/repository/health_score.go` — `HealthScoreRepository`:
    - Struct: `HealthScore{ID, OrgID, CustomerID, OverallScore, RiskLevel, Factors, CalculatedAt}`
    - `UpsertCurrent(ctx, score *HealthScore) error` — INSERT ON CONFLICT(customer_id) UPDATE
    - `GetByCustomerID(ctx, customerID, orgID uuid.UUID) (*HealthScore, error)`
    - `ListByOrg(ctx, orgID uuid.UUID, filters HealthScoreFilters) ([]HealthScore, error)` — paginated, filterable by risk_level
    - `InsertHistory(ctx, score *HealthScore) error` — append to health_score_history
    - `GetHistory(ctx, customerID uuid.UUID, limit int) ([]HealthScore, error)` — ordered by calculated_at DESC

**Tests:** `internal/service/scoring/aggregator_test.go`:
- All 5 factors present → correct weighted sum
- 2 factors missing → weights redistributed, correct sum
- Only 1 factor present → gets full weight
- All factors nil → returns error
- Threshold boundaries: score exactly at green threshold → green, one below → yellow
- Score always in 0-100 range (no overflow/underflow)

**Acceptance criteria:**

- [ ] Weighted aggregation produces correct overall score
- [ ] Missing factors handled by weight redistribution
- [ ] Score always in 0-100 range
- [ ] Risk level assigned by thresholds
- [ ] All present factors included in result
- [ ] Edge case: all factors missing → return nil/error
- [ ] Tests cover full factors, partial factors, threshold boundaries

---

### Phase 4 — Recalculation Scheduler

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#90](https://github.com/subculture-collective/pulse-score/issues/90) | Implement score recalculation scheduler | critical | `internal/service/scoring/scheduler.go` |

**Depends on:** Phase 3 (#89 aggregation service)

**Details:**

1. Create `internal/service/scoring/scheduler.go` — `ScoreScheduler`:
    - Inject `ScoreAggregator`, `HealthScoreRepository`, `CustomerRepository`, `CustomerEventRepository`
    - Follow existing `SyncSchedulerService` pattern from `internal/service/stripe_sync.go`
    - **Batch mode** — `RunBatch(ctx context.Context) error`:
      1. List all orgs with active integration connections
      2. For each org: list all customers
      3. Optimization: track `last_event_at` per customer, skip if no new events since last score calculation
      4. Calculate score for each eligible customer via `ScoreAggregator.Calculate()`
      5. Upsert into `health_scores`, insert into `health_score_history`
      6. Continue on per-customer errors (log and count failures like `MRRService`)
    - **Event-triggered mode** — `RecalculateCustomer(ctx, customerID, orgID uuid.UUID) error`:
      1. Called on payment.failed, subscription.cancelled, mrr.changed events
      2. Recalculate single customer immediately
      3. Upsert + history insert
    - **Scheduling** — `Start(ctx context.Context)`:
      1. Ticker-based loop (configurable interval, default 1 hour)
      2. `go scheduler.Start(bgCtx)` from `main.go` like existing background services
    - **Concurrency**: worker pool with configurable parallelism (default 5 goroutines)
    - **Config**: `SCORE_RECALC_INTERVAL` (default "1h"), `SCORE_RECALC_WORKERS` (default 5)

2. Wire event-triggered recalculation into Stripe webhook handler: when payment or subscription events arrive, call `scheduler.RecalculateCustomer()`

3. Wire batch scheduler startup in `cmd/api/main.go`:
    ```go
    scoreScheduler := scoring.NewScoreScheduler(aggregator, healthScoreRepo, customerRepo, eventRepo)
    go scoreScheduler.Start(bgCtx)
    ```

**Tests:** `internal/service/scoring/scheduler_test.go`:
- Batch mode processes all eligible customers
- Customers with no new events are skipped
- Event-triggered recalculation works within single call
- Errors on individual customers don't stop batch
- Worker pool processes customers concurrently

**Acceptance criteria:**

- [ ] Batch recalculation runs on configured schedule
- [ ] Event-triggered recalculation works within seconds
- [ ] Customers with no new events skipped efficiently
- [ ] Concurrent processing with configurable parallelism
- [ ] health_scores and health_score_history updated
- [ ] Significant changes create customer_events
- [ ] Tests cover batch, event-triggered, optimization logic

---

### Phase 5 — Change Detection, Risk Categorization, Admin API (parallel)

All three issues in this phase depend on the scheduler (Phase 4) or aggregation (Phase 3) and can be worked on in parallel.

#### Phase 5a — Score Change Detection & History Tracking

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#91](https://github.com/subculture-collective/pulse-score/issues/91) | Implement score change detection and history tracking | high | `internal/service/scoring/change_detector.go` |

**Depends on:** Phase 4 (#90 scheduler)

**Details:**

1. Create `internal/service/scoring/change_detector.go` — `ChangeDetector`:
    - Called by `ScoreScheduler` after each score calculation
    - `Detect(ctx, previous *HealthScore, current *HealthScoreResult) (*ScoreChange, error)`:
      1. Compare previous score to current score
      2. Significant change detection:
         - Absolute change > 10 points
         - Risk level transition (green→yellow, yellow→red, etc.)
      3. If significant: create `customer_event` with type `score.changed` or `risk_level.changed`
      4. Event data JSONB:
         ```json
         {
           "previous_score": 72,
           "new_score": 58,
           "change_amount": -14,
           "previous_risk": "green",
           "new_risk": "yellow"
         }
         ```
    - All scores (significant or not) are recorded in `health_score_history`
    - Risk level transition events feed into the alert system (Epic 6)
    - Struct:
      ```go
      type ScoreChange struct {
          CustomerID    uuid.UUID
          PreviousScore int
          NewScore      int
          ChangeAmount  int
          PreviousRisk  string
          NewRisk       string
          IsSignificant bool
      }
      ```

2. Integrate into `ScoreScheduler`: after each calculation, fetch previous score, run change detection, create events if significant

**Tests:** `internal/service/scoring/change_detector_test.go`:
- No previous score (first calculation) → not significant
- 5-point change → not significant
- 15-point change → significant
- Risk level change green→yellow → significant
- Risk level change yellow→red → significant
- Same score → not significant
- Boundary: exactly 10-point change → not significant (> 10 required)
- Events created correctly with proper data

**Acceptance criteria:**

- [ ] All score changes recorded in history
- [ ] Significant changes detected (>10 points or risk level change)
- [ ] Customer events created for significant changes
- [ ] Risk level transitions tracked (green→yellow, yellow→red, etc.)
- [ ] Events include both old and new values
- [ ] History supports time-series queries
- [ ] Tests cover change detection thresholds and risk transitions

#### Phase 5b — Customer Risk Categorization

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#92](https://github.com/subculture-collective/pulse-score/issues/92) | Implement customer risk categorization | high | `internal/service/scoring/categorizer.go` |

**Depends on:** Phase 3 (#89 aggregation), Phase 1 (#83 scoring config)

**Details:**

1. Create `internal/service/scoring/categorizer.go` — `RiskCategorizer`:
    - Standalone query service for risk distribution/stats (dashboard use)
    - Inject `HealthScoreRepository`, `ScoringConfigRepository`
    - `GetRiskDistribution(ctx, orgID uuid.UUID) (*RiskDistribution, error)`:
      1. Load thresholds from scoring config
      2. Query health_scores grouped by risk_level for org
      3. Return counts and percentages per level
    - `GetScoreHistogram(ctx, orgID uuid.UUID) (*ScoreHistogram, error)`:
      1. Query health_scores for org
      2. Bucket into 10-point ranges (0-10, 11-20, ..., 91-100)
      3. Return bucket counts
    - Structs:
      ```go
      type RiskDistribution struct {
          Green  RiskBucket
          Yellow RiskBucket
          Red    RiskBucket
          Total  int
      }
      type RiskBucket struct {
          Count      int
          Percentage float64
      }
      type ScoreHistogram struct {
          Buckets []HistogramBucket // 10 buckets
      }
      type HistogramBucket struct {
          Min   int
          Max   int
          Count int
      }
      ```

2. Add repository queries to `HealthScoreRepository`:
    - `CountByRiskLevel(ctx, orgID uuid.UUID) (map[string]int, error)` — GROUP BY risk_level
    - `ScoreDistribution(ctx, orgID uuid.UUID) ([]int, error)` — all scores for histogram

**Tests:** `internal/service/scoring/categorizer_test.go`:
- Correct categorization at threshold boundaries
- Percentages sum to 100%
- Histogram buckets cover full 0-100 range
- Empty org (no scores) → all zeros
- Custom thresholds applied correctly

**Acceptance criteria:**

- [ ] Customers categorized correctly by thresholds
- [ ] Risk distribution stats calculated accurately
- [ ] Histogram data generated for score distribution
- [ ] Custom thresholds applied per org
- [ ] Tests verify categorization at threshold boundaries

#### Phase 5c — Score Configuration Admin API

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#93](https://github.com/subculture-collective/pulse-score/issues/93) | Implement score configuration admin API | high | `internal/handler/scoring_config.go`, `internal/service/scoring/config_service.go` |

**Depends on:** Phase 1 (#83 scoring config model), Phase 4 (#90 scheduler for recalc trigger)

**Details:**

1. Create `internal/service/scoring/config_service.go` — `ConfigService`:
    - `GetConfig(ctx, orgID uuid.UUID) (*ScoringConfig, error)` — return current config, create default if none exists
    - `UpdateConfig(ctx, orgID uuid.UUID, req UpdateConfigRequest) (*ScoringConfig, error)`:
      1. Validate weights sum to 1.0, each 0.0-1.0
      2. Validate thresholds: green > yellow > 0
      3. Upsert config
      4. Trigger async full score recalculation for all org customers
      5. Return updated config

2. Create `internal/handler/scoring_config.go` — `ScoringConfigHandler`:
    - `GetConfig(w, r)` — `GET /api/v1/scoring/config`
      - Extract orgID from context (tenant middleware)
      - Call service.GetConfig()
      - Return JSON response with config + default annotations
    - `UpdateConfig(w, r)` — `PUT /api/v1/scoring/config`
      - Extract orgID from context
      - Parse and validate request body
      - Call service.UpdateConfig()
      - Return JSON response with updated config + `recalculation_status: "triggered"`
    - Request body:
      ```json
      {
        "weights": {
          "payment_recency": 0.3,
          "mrr_trend": 0.2,
          "failed_payments": 0.2,
          "support_tickets": 0.15,
          "engagement": 0.15
        },
        "thresholds": {
          "green": 70,
          "yellow": 40
        }
      }
      ```
    - Validation errors return 422

3. Wire routes in `cmd/api/main.go`:
    ```go
    r.Route("/api/v1/scoring", func(r chi.Router) {
        r.Use(middleware.JWTAuth(jwtMgr))
        r.Use(middleware.TenantIsolation(orgRepo))
        r.Use(middleware.RequireRole("admin"))
        r.Get("/config", scoringConfigHandler.GetConfig)
        r.Put("/config", scoringConfigHandler.UpdateConfig)
    })
    ```

**Tests:** `internal/handler/scoring_config_test.go`:
- GET returns current config
- PUT updates config successfully
- PUT with weights not summing to 1.0 → 422
- PUT with invalid thresholds (green < yellow) → 422
- Member role → 403 Forbidden
- Admin role → 200 Success
- Config change triggers recalculation

**Acceptance criteria:**

- [ ] GET returns current config
- [ ] PUT updates config with validation
- [ ] Invalid weights (don't sum to 1.0) return 422
- [ ] Invalid thresholds return 422
- [ ] Config change triggers score recalculation
- [ ] Only admin/owner can modify (403 for members)
- [ ] Tests cover validation, update, RBAC

---

## File Summary

### New Files to Create

| File | Phase | Issue |
|------|-------|-------|
| `migrations/000013_create_scoring_configs.up.sql` | 1 | #83 |
| `migrations/000013_create_scoring_configs.down.sql` | 1 | #83 |
| `internal/repository/scoring_config.go` | 1 | #83 |
| `internal/service/scoring/factor.go` | 2 | #84-#88 |
| `internal/service/scoring/payment_recency.go` | 2a | #84 |
| `internal/service/scoring/mrr_trend.go` | 2b | #85 |
| `internal/service/scoring/failed_payments.go` | 2c | #86 |
| `internal/service/scoring/support_tickets.go` | 2d | #87 |
| `internal/service/scoring/engagement.go` | 2e | #88 |
| `internal/service/scoring/aggregator.go` | 3 | #89 |
| `internal/repository/health_score.go` | 3 | #89 |
| `internal/service/scoring/scheduler.go` | 4 | #90 |
| `internal/service/scoring/change_detector.go` | 5a | #91 |
| `internal/service/scoring/categorizer.go` | 5b | #92 |
| `internal/service/scoring/config_service.go` | 5c | #93 |
| `internal/handler/scoring_config.go` | 5c | #93 |

### Existing Files to Modify

| File | Phase | Changes |
|------|-------|---------|
| `cmd/api/main.go` | 1, 4, 5c | Wire repos, services, scheduler, routes |
| `internal/service/organization.go` | 1 | Create default scoring config on org creation |
| `internal/repository/customer_event.go` | 2b, 2d, 2e | Add helper queries for event-based factors |
| `internal/handler/integration_stripe.go` | 4 | Trigger event-based recalculation on webhook events |
| `internal/config/config.go` | 4 | Add `SCORE_RECALC_INTERVAL`, `SCORE_RECALC_WORKERS` |

### Test Files to Create

| File | Phase | Coverage |
|------|-------|----------|
| `internal/repository/scoring_config_test.go` | 1 | Validation, CRUD |
| `internal/service/scoring/payment_recency_test.go` | 2a | Monthly, annual, no history, decay curves |
| `internal/service/scoring/mrr_trend_test.go` | 2b | Growth, decline, stable, time weighting |
| `internal/service/scoring/failed_payments_test.go` | 2c | All failure patterns |
| `internal/service/scoring/support_tickets_test.go` | 2d | Normalization, missing data |
| `internal/service/scoring/engagement_test.go` | 2e | Active, inactive, no data |
| `internal/service/scoring/aggregator_test.go` | 3 | Weight redistribution, thresholds |
| `internal/service/scoring/scheduler_test.go` | 4 | Batch, event-triggered, optimization |
| `internal/service/scoring/change_detector_test.go` | 5a | Thresholds, risk transitions |
| `internal/service/scoring/categorizer_test.go` | 5b | Distribution, histogram |
| `internal/handler/scoring_config_test.go` | 5c | API, validation, RBAC |

---

## Implementation Order (recommended)

```
Week 1:  Phase 1 (#83) → Phase 2a (#84) + Phase 2b (#85) + Phase 2c (#86) in parallel
Week 2:  Phase 2d (#87) + Phase 2e (#88) in parallel → Phase 3 (#89)
Week 3:  Phase 4 (#90) → Phase 5a (#91) + Phase 5b (#92) + Phase 5c (#93) in parallel
```

## Notes

- **Existing `PaymentRecencyService` and `PaymentHealthService`** already compute 0-100 scores — the scoring factors should wrap/delegate to these rather than duplicating logic.
- **The `health_scores` and `health_score_history` tables already exist** (migration 000007) — only the `scoring_configs` table needs a new migration.
- **The `ScoreFactor` interface** is the key abstraction: all factors produce a `*float64` (nil = unavailable), enabling clean weight redistribution in the aggregator.
- **Graceful degradation** is critical: support tickets and engagement factors will often be nil in early-stage orgs. The aggregator must handle this without penalty.
- **Event-triggered recalculation** should be debounced or rate-limited to avoid excessive recalcs during bulk webhook processing (e.g., initial Stripe sync).
