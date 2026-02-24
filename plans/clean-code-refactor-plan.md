# Clean Code Refactor Plan (PR-Sized)

Date: 2026-02-24
Source: `clean-code-review.json` (`CC-001` ... `CC-010`)

## Goal

Refactor high/medium/low clean-code findings **without changing runtime behavior**.

## Guardrails

- Preserve existing request/response behavior and background job behavior.
- Keep all public route paths, payloads, and status codes unchanged.
- Run validation after each PR:
  - `make lint`
  - `make test`
  - `make web-lint`
  - `make web-format-check`

## Execution Strategy

Work from low-risk, high-confidence extractions toward larger architectural splits.

---

## PR-01: Shared constants + webhook body helper

**Targets**: `CC-006`, `CC-010`
**Risk**: Low
**Estimated size**: ~150-250 LOC touched

### Scope

- Introduce shared constants for repeated literals:
  - webhook body max bytes (`65536`)
  - read chunk bytes (`1024`)
  - CORS max age (`300`)
  - connection monitor interval (`60`)
- Replace inline usage in:
  - `cmd/api/main.go`
  - `internal/handler/integration_stripe.go`
  - `internal/handler/integration_hubspot.go`
  - `internal/handler/integration_intercom.go`
  - `internal/handler/billing.go`
- Rename vague local names in the request-body reader (`buf/tmp` → `bodyBytes/chunkBuffer`).

### Acceptance Criteria

- No behavior change in webhook handling.
- No remaining inline `65536` / `1024` for webhook-body handling.
- All checks pass.

---

## PR-02: Centralized HTTP service error translation

**Targets**: `CC-007`
**Risk**: Low
**Estimated size**: ~80-140 LOC touched

### Scope

- Create a package-level handler utility (e.g., `internal/handler/errors.go`) for mapping service errors to HTTP responses.
- Remove indirect coupling from `internal/handler/integration_stripe.go:121-124` (`AuthHandler` proxy call).
- Switch affected handlers to the shared utility.

### Acceptance Criteria

- Existing error status codes/messages remain unchanged.
- No `AuthHandler` instantiation solely for error forwarding.
- All checks pass.

---

## PR-03: Deduplicate integration handler control flow

**Targets**: `CC-003`
**Risk**: Medium
**Estimated size**: ~250-450 LOC touched

### Scope

- Refactor repeated patterns across:
  - `internal/handler/integration_stripe.go`
  - `internal/handler/integration_hubspot.go`
  - `internal/handler/integration_intercom.go`
- Extract common patterns:
  - org resolution + unauthorized response
  - status/disconnect flow
  - trigger sync response flow
  - OAuth callback shared structure (provider-specific error text retained)

### Acceptance Criteria

- Endpoint behavior and response payloads are unchanged for all 3 providers.
- Duplicate handler logic reduced materially (target: eliminate copy-paste blocks for Connect/Status/Disconnect/TriggerSync).
- All checks pass.

---

## PR-04: Refactor `IntercomSyncService.SyncConversations`

**Targets**: `CC-002`, `CC-004` (partial)
**Risk**: Medium
**Estimated size**: ~220-380 LOC touched

### Scope

- Split `SyncConversations` into focused helpers:
  - customer resolution
  - conversation-to-repository mapping
  - optional event emission
- Keep outer method responsible only for pagination + orchestration.
- Align helper shapes so incremental method can reuse them.

### Acceptance Criteria

- `SyncConversations` reduced below 100 lines.
- No change in persistence fields written to `intercom_conversations` and `customer_events`.
- All checks pass.

---

## PR-05: Consolidate full/incremental sync pipelines (Stripe)

**Targets**: `CC-004` (Stripe portion)
**Risk**: Medium
**Estimated size**: ~220-360 LOC touched

### Scope

- In `internal/service/stripe_sync.go`, extract shared processors used by:
  - `SyncCustomers` + `SyncCustomersSince`
  - `SyncPayments` + `SyncPaymentsSince`
- Keep fetch-source differences isolated to iterator setup.

### Acceptance Criteria

- No change in event creation behavior for failed payments.
- Shared transformation/upsert logic has single implementation path per entity type.
- All checks pass.

---

## PR-06: Consolidate full/incremental sync pipelines (HubSpot + Intercom contacts)

**Targets**: `CC-004` (remaining scope)
**Risk**: Medium
**Estimated size**: ~250-420 LOC touched

### Scope

- Reuse shared processing functions for:
  - HubSpot `SyncContacts` + `SyncContactsSince`
  - HubSpot `SyncDeals` + `SyncDealsSince`
  - Intercom `SyncContacts` + `SyncContactsSince`
- Preserve existing logging semantics.

### Acceptance Criteria

- Data mapping/parsing behavior unchanged.
- Reduced duplication in listed pairs.
- All checks pass.

---

## PR-07: Constructor dependency object for alert scheduler

**Targets**: `CC-005`
**Risk**: Medium
**Estimated size**: ~100-180 LOC touched

### Scope

- Introduce `AlertSchedulerDeps` struct.
- Replace `NewAlertScheduler(...)` long parameter list with `NewAlertScheduler(deps AlertSchedulerDeps, intervalMinutes int, frontendURL string)` (or similar minimal delta approach).
- Update callsite(s) in `cmd/api/main.go`.

### Acceptance Criteria

- Constructor call is clearer and parameter order mistakes are less likely.
- Runtime wiring remains unchanged.
- All checks pass.

---

## PR-08: Split `cmd/api/main.go` bootstrap flow

**Targets**: `CC-001`, `CC-008`
**Risk**: High
**Estimated size**: ~400-700 LOC touched

### Scope

Refactor `main` into explicit phases with small functions:

- `initLogger()`
- `loadAndValidateConfig()`
- `openDatabase()`
- `buildDependencies()` (repos/services)
- `registerPublicRoutes()`
- `registerProtectedRoutes()`
- `startBackgroundWorkers()`
- `newHTTPServer()`
- `runServerWithGracefulShutdown()`

Keep behavior identical and avoid route/middleware drift.

### Acceptance Criteria

- `main` reduced well below 150 lines (target).
- Route registration still exposes identical endpoint map.
- Startup/shutdown behavior unchanged.
- All checks pass.

---

## PR-09: Repository error handling convention fix

**Targets**: `CC-009`
**Risk**: Low
**Estimated size**: ~20-60 LOC touched

### Scope

- Replace string comparison in `internal/repository/customer_event.go:47` with typed error handling (`errors.Is(..., pgx.ErrNoRows)` or equivalent).

### Acceptance Criteria

- Same logical behavior (`DO NOTHING` conflict returns nil path) is preserved.
- No string-comparison-based row-not-found checks remain at this location.
- All checks pass.

---

## Suggested Merge Order

1. PR-01
2. PR-02
3. PR-03
4. PR-04
5. PR-05
6. PR-06
7. PR-07
8. PR-08
9. PR-09

## Rollback Strategy

- Each PR must remain independently revertible.
- Avoid combining structural and behavioral changes in the same PR.
- If regressions appear in PR-08 (largest), revert only PR-08 while retaining earlier cleanup PRs.

## Definition of Done

- All `CC-001` to `CC-010` addressed.
- No API contract changes.
- CI checks (`make check`) pass.
- Refactor commits are small, reviewable, and scoped to one PR objective each.
