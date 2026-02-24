# Clean Code Review

Scope: `cmd/` and `internal/` Go source files (focus on production code).

## High Severity

### Function Issues: `main` function does too many things

- **Principle**: Small Functions + SRP
- **Location**: `cmd/api/main.go:31-620`
- **Severity**: High
- **Issue**: `main` is 590 lines and mixes configuration, DB wiring, dependency construction, route registration, scheduler startup, and server lifecycle. This creates a high-change-risk hotspot.
- **Suggestion**: Extract cohesive units such as `initRuntime()`, `buildRepositories()`, `buildServices()`, `registerRoutes()`, `startBackgroundJobs()`, and `runHTTPServer()`.

### Function Issues: `SyncConversations` is oversized and multi-responsibility

- **Principle**: Small Functions + SRP
- **Location**: `internal/service/intercom_sync.go:127-247`
- **Severity**: High
- **Issue**: 121-line method combines API pagination, model mapping, customer lookup, persistence, and event emission.
- **Suggestion**: Split into helper methods (`resolveCustomerID`, `mapConversation`, `upsertConversation`, `emitConversationEvent`) and keep orchestration in the top-level method.

### Duplication Issues: Provider integration handlers repeat near-identical flows

- **Principle**: DRY
- **Location**: `internal/handler/integration_stripe.go:27-117`, `internal/handler/integration_hubspot.go:27-115`, `internal/handler/integration_intercom.go:27-115`
- **Severity**: High
- **Issue**: `Connect`, `Callback`, `Status`, `Disconnect`, and `TriggerSync` follow almost the same control flow across three files.
- **Suggestion**: Introduce shared helper(s) or a provider strategy abstraction for common auth/org lookup, error handling, and sync triggering.

## Medium Severity

### Duplication Issues: Full-sync and incremental-sync paths duplicate mapping/upsert logic

- **Principle**: DRY
- **Location**: `internal/service/stripe_sync.go:56-428`, `internal/service/intercom_sync.go:44-411`, `internal/service/hubspot_sync.go:48-507`
- **Severity**: Medium
- **Issue**: Full and incremental methods duplicate entity mapping and persistence logic, increasing drift risk and bug-fix overhead.
- **Suggestion**: Refactor to shared processing pipelines where only source iterators differ.

### Function Issues: Constructor has high parameter count

- **Principle**: Small Functions + SRP
- **Location**: `internal/service/alert_scheduler.go:29-38`
- **Severity**: Medium
- **Issue**: `NewAlertScheduler` requires 9 parameters, signaling high coupling and reducing readability/testability.
- **Suggestion**: Use a dependency struct (e.g., `AlertSchedulerDeps`) or grouped sub-dependencies.

### Magic Numbers: Repeated hardcoded limits and intervals

- **Principle**: Avoid Hardcoding
- **Location**: `cmd/api/main.go:85`, `cmd/api/main.go:383`, `internal/handler/integration_stripe.go:138`, `internal/handler/integration_stripe.go:165`, `internal/handler/integration_hubspot.go:130`, `internal/handler/integration_intercom.go:130`, `internal/handler/billing.go:143`
- **Severity**: Medium
- **Issue**: Values like `300`, `60`, `65536`, and `1024` appear inline and in multiple places.
- **Suggestion**: Replace with named constants (`corsMaxAgeSeconds`, `connectionMonitorIntervalSeconds`, `maxWebhookBodyBytes`, `readChunkBytes`) in shared config/constants.

### Over-Engineering: Error handling helper is indirectly wired through unrelated handler

- **Principle**: YAGNI
- **Location**: `internal/handler/integration_stripe.go:121-124`
- **Severity**: Medium
- **Issue**: `handleServiceError` creates an `AuthHandler` solely to call its method, introducing hidden coupling.
- **Suggestion**: Move service-error translation into a package-level utility used directly by all handlers.

### Structural Clarity: Deep nesting in route setup reduces scanability

- **Principle**: Readability First
- **Location**: `cmd/api/main.go:90-578`
- **Severity**: Medium
- **Issue**: Multiple nested `Route`/`Group` closures and conditional blocks make control flow difficult to follow.
- **Suggestion**: Split route registration by domain (`registerAuthRoutes`, `registerBillingRoutes`, `registerIntegrationRoutes`, etc.).

## Low Severity

### Project Conventions: String-based error matching is brittle

- **Principle**: Consistency
- **Location**: `internal/repository/customer_event.go:47`
- **Severity**: Low
- **Issue**: `err.Error() == "no rows in result set"` relies on message text instead of typed errors.
- **Suggestion**: Prefer typed checks (e.g., `errors.Is(err, pgx.ErrNoRows)`) to align with idiomatic Go error handling.

### Naming Issues: Temporary buffer names are vague

- **Principle**: Meaningful Names
- **Location**: `internal/handler/integration_stripe.go:163-172`
- **Severity**: Low
- **Issue**: `buf` and `tmp` are generic in request body parsing, reducing local readability.
- **Suggestion**: Rename to intent-revealing names like `bodyBytes` and `chunkBuffer`.

## Notes

- I did **not** flag import ordering style issues; files appear gofmt-consistent.
- Severity is maintainability-focused and does not imply current runtime bugs.
