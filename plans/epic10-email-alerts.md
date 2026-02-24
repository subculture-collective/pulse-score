# Execution Plan: Epic 10 — Email Alerts (#11)

## Overview

**Epic:** [#11 — Email Alerts](https://github.com/subculture-collective/pulse-score/issues/11)
**Sub-issues:** #138–#145 (8 issues)
**Scope:** Build an email alert system using SendGrid for transactional emails. Implement alert rule evaluation engine, health score drop detection, email templates, delivery tracking, notification preferences, alert configuration UI, and in-app notification system.

## Current State

The following foundations are already in place:

- **Alert Rules CRUD API** — Handler, service, repository, and migration fully implemented (`alert_rules` + `alert_history` tables via migration 000009)
- **SendGrid integration** — `SendGridEmailService` with `SendInvitation` and `SendPasswordReset` methods, dev-mode logging, raw HTTP client
- **Health Scoring Engine** — `ScoreScheduler` runs periodic batch recalculation with `ChangeDetector` that records `score.changed` and `risk_level.changed` events to `customer_events`
- **Auth/RBAC** — JWT auth, tenant isolation, `RequireRole("admin")` middleware
- **Frontend** — Settings page with tab navigation (Organization, Profile, Integrations, Scoring, Billing, Team), axios API client

### Existing Infrastructure (reuse these)

| Component | File | Output | Reuse As |
|-----------|------|--------|----------|
| `AlertRuleRepository` | `internal/repository/alert_rule.go` | CRUD for `alert_rules` | Rule configuration storage |
| `AlertRuleService` | `internal/service/alert_rule.go` | Validation + CRUD | Rule management logic |
| `AlertRuleHandler` | `internal/handler/alert_rule.go` | REST endpoints | Alert rules API |
| `SendGridEmailService` | `internal/service/sendgrid.go` | Email delivery via SendGrid | Extend for alert emails |
| `ChangeDetector` | `internal/service/scoring/change_detector.go` | Records `score.changed`, `risk_level.changed` events | Alert trigger source |
| `ScoreScheduler` | `internal/service/scoring/scheduler.go` | Periodic scoring + change detection | Hook alert evaluation after scoring |
| `CustomerEventRepository` | `internal/repository/customer_event.go` | Event storage + querying | Query trigger events |
| `HealthScoreRepository` | `internal/repository/health_score.go` | Current + historical scores | Score comparison for alerts |

### Existing Tables (already migrated)

| Table | Migration | Key Columns |
|-------|-----------|-------------|
| `alert_rules` | 000009 | org_id, name, trigger_type, conditions (JSONB), channel, recipients (JSONB), is_active |
| `alert_history` | 000009 | org_id, alert_rule_id, customer_id, trigger_data (JSONB), channel, status (sent/failed/pending), sent_at, error_message |
| `health_scores` | 000007 | org_id, customer_id, overall_score (0-100), risk_level (green/yellow/red), factors (JSONB) |
| `health_score_history` | 000007 | Append-only history of health_scores |
| `customer_events` | 000006 | org_id, customer_id, event_type, source, occurred_at, data (JSONB) |
| `customers` | 000005 | org_id, external_id, source, email (CITEXT), name, company_name, mrr_cents |

## Dependency Graph

```
#138 SendGrid Integration ──► #139 Email Templates ──┐
                                                      ├──► #141 Score Drop Triggers ──► #143 Delivery Tracking
#140 Alert Rule Engine ───────────────────────────────┘          │
                                                                 ├──► #142 Alert Management UI ──► #144 Notification Preferences UI
                                                                 │
                                                                 └──► #145 In-App Notifications
```

## Execution Phases

Issues are grouped into phases based on dependency chains. Issues within the same phase can be worked on in parallel.

---

### Phase 1 — SendGrid Enhancement & Alert History Repository

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#138](https://github.com/subculture-collective/pulse-score/issues/138) | Set up SendGrid integration and email service | high | `internal/service/email.go`, `internal/repository/alert_history.go` |

**Details:**

The SendGrid integration already exists in `internal/service/sendgrid.go`. This issue extends it into a generic, retry-capable email service abstraction.

1. Create `internal/service/email.go` — `EmailService` interface:
    ```go
    type EmailService interface {
        SendEmail(ctx context.Context, params SendEmailParams) error
    }

    type SendEmailParams struct {
        To       string
        Subject  string
        HTMLBody string
        TextBody string
    }
    ```

2. Extend `SendGridEmailService` in `internal/service/sendgrid.go`:
    - Add `SendEmail(ctx, params SendEmailParams) error` method implementing the `EmailService` interface
    - Add retry logic: up to 3 attempts with exponential backoff (1s, 2s, 4s)
    - Only retry on transient errors (5xx status codes, network timeouts)
    - Add `SendAlertEmail(ctx, params AlertEmailParams) error` convenience method
    - Log all send attempts, successes, and failures via `slog`

3. Create `internal/repository/alert_history.go` — `AlertHistoryRepository`:
    ```go
    type AlertHistory struct {
        ID           uuid.UUID      `json:"id"`
        OrgID        uuid.UUID      `json:"org_id"`
        AlertRuleID  uuid.UUID      `json:"alert_rule_id"`
        CustomerID   *uuid.UUID     `json:"customer_id,omitempty"`
        TriggerData  map[string]any `json:"trigger_data"`
        Channel      string         `json:"channel"`
        Status       string         `json:"status"` // sent, failed, pending
        SentAt       *time.Time     `json:"sent_at,omitempty"`
        ErrorMessage string         `json:"error_message,omitempty"`
        CreatedAt    time.Time      `json:"created_at"`
    }
    ```
    - `Create(ctx, history *AlertHistory) error` — insert new history record
    - `UpdateStatus(ctx, id uuid.UUID, status string, errorMsg string) error` — update delivery status
    - `ListByOrg(ctx, orgID uuid.UUID, limit, offset int) ([]*AlertHistory, int, error)` — paginated list with total count
    - `ListByRule(ctx, ruleID uuid.UUID, limit, offset int) ([]*AlertHistory, error)` — filter by rule
    - `GetLastAlertForRule(ctx, ruleID, customerID uuid.UUID) (*AlertHistory, error)` — for deduplication/cooldown
    - `CountByStatus(ctx, orgID uuid.UUID) (map[string]int, error)` — for dashboard metrics

4. Wire `AlertHistoryRepository` in `cmd/api/main.go`

**Tests:**
- `internal/service/sendgrid_test.go` — retry logic (mock HTTP client), dev mode logging, error handling for 4xx vs 5xx
- `internal/repository/alert_history_test.go` — CRUD operations, pagination, deduplication query, status updates

**Acceptance criteria:**

- [ ] `EmailService` interface defined
- [ ] `SendEmail` method with retry logic (3 attempts, exponential backoff)
- [ ] Only retries on transient failures (5xx, timeouts)
- [ ] `AlertHistoryRepository` with Create, UpdateStatus, ListByOrg, GetLastAlertForRule
- [ ] Dev mode logs to stdout instead of sending
- [ ] Tests with mocked HTTP client

---

### Phase 2a — Email Templates

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#139](https://github.com/subculture-collective/pulse-score/issues/139) | Create email templates for alerts | high | `internal/templates/emails/*.html`, `internal/service/email_templates.go` |

**Depends on:** Phase 1 (#138)

**Details:**

1. Create `internal/templates/emails/` directory with template files:
    - `base.html` — base layout with header (logo), content block, footer (unsubscribe link, company info)
    - `score_drop.html` — score drop alert (customer name, old score, new score, delta, top negative factor, link to customer detail)
    - `at_risk.html` — risk level change alert (customer name, previous level, new level, recommended actions, link)
    - `payment_failed.html` — payment failure notification (customer name, amount, failure reason, link)
    - `weekly_digest.html` — weekly summary (total customers, scores improved/declined, at-risk count, top movers table)

2. Create `internal/service/email_templates.go` — template rendering:
    ```go
    type EmailTemplateService struct {
        templates *template.Template
    }

    type ScoreDropEmailData struct {
        CustomerName       string
        CompanyName        string
        OldScore           int
        NewScore           int
        Delta              int
        TopNegativeFactor  string
        CustomerDetailURL  string
        UnsubscribeURL     string
    }

    type RiskChangeEmailData struct {
        CustomerName     string
        CompanyName      string
        PreviousLevel    string
        NewLevel         string
        Score            int
        CustomerDetailURL string
        UnsubscribeURL   string
    }

    type PaymentFailedEmailData struct {
        CustomerName     string
        CompanyName      string
        Amount           string
        FailureReason    string
        CustomerDetailURL string
        UnsubscribeURL   string
    }

    type WeeklyDigestEmailData struct {
        OrgName          string
        TotalCustomers   int
        AtRiskCount      int
        ImprovedCount    int
        DeclinedCount    int
        TopMovers        []CustomerScoreChange
        DashboardURL     string
        UnsubscribeURL   string
    }
    ```
    - `RenderScoreDrop(data ScoreDropEmailData) (html string, text string, error)`
    - `RenderRiskChange(data RiskChangeEmailData) (html string, text string, error)`
    - `RenderPaymentFailed(data PaymentFailedEmailData) (html string, text string, error)`
    - `RenderWeeklyDigest(data WeeklyDigestEmailData) (html string, text string, error)`

3. Template design requirements:
    - Inline CSS only (email clients don't support `<style>` blocks reliably)
    - Responsive: single-column layout, max-width 600px
    - Color-coded score badges (green/yellow/red)
    - Plain text fallback for each template
    - Go `html/template` with layout inheritance via `{{ template "base" . }}`
    - Embed templates via `//go:embed` directive

**Tests:** `internal/service/email_templates_test.go` — each template renders correctly with sample data, plain text fallback, no missing variable errors, HTML escaping of user-provided data

**Acceptance criteria:**

- [ ] Four HTML email templates created (score_drop, at_risk, payment_failed, weekly_digest)
- [ ] Base layout with header, content block, footer
- [ ] Responsive design (mobile-friendly, inline CSS)
- [ ] Plain text fallback for each template
- [ ] Templates render correctly with Go html/template
- [ ] User-provided data is HTML-escaped
- [ ] Templates embedded via `//go:embed`
- [ ] Tests verify rendering for each template

---

### Phase 2b — Alert Rule Evaluation Engine

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#140](https://github.com/subculture-collective/pulse-score/issues/140) | Build alert rule evaluation engine | critical | `internal/service/alert_engine.go`, `internal/service/alert_scheduler.go` |

**Depends on:** Phase 1 (#138)

**Details:**

1. Create `internal/service/alert_engine.go` — `AlertEngine`:
    ```go
    type AlertEngine struct {
        alertRules   *repository.AlertRuleRepository
        alertHistory *repository.AlertHistoryRepository
        healthScores *repository.HealthScoreRepository
        customers    *repository.CustomerRepository
        events       *repository.CustomerEventRepository
    }

    type AlertMatch struct {
        Rule        *repository.AlertRule
        Customer    *repository.Customer
        TriggerData map[string]any
    }
    ```

    - `EvaluateAll(ctx, orgID uuid.UUID) ([]AlertMatch, error)` — evaluate all active rules for an org
    - `EvaluateRule(ctx, rule *AlertRule, orgID uuid.UUID) ([]AlertMatch, error)` — evaluate a single rule
    - Rule type implementations:
      - **`score_below`**: Query `health_scores` for customers where `overall_score < threshold`
        - Conditions: `{"threshold": 40}`
      - **`score_drop`**: Compare current score vs score from N days ago in `health_score_history`
        - Conditions: `{"points": 10, "days": 7}` — fire when score drops ≥ 10 points in 7 days
      - **`risk_change`**: Query `customer_events` for recent `risk_level.changed` events
        - Conditions: `{"from": "green", "to": "yellow"}` or `{"to": "red"}` (any transition to red)
      - **`event_trigger`**: Query `customer_events` for specific event types
        - Conditions: `{"event_type": "payment.failed"}`

    - **Deduplication**: Before firing, check `AlertHistoryRepository.GetLastAlertForRule(ruleID, customerID)` — if an alert was sent within the cooldown period (default 24h, configurable via `conditions.cooldown_hours`), skip
    - Return `[]AlertMatch` — list of (rule, customer, trigger_data) tuples ready for delivery

2. Create `internal/service/alert_scheduler.go` — `AlertScheduler`:
    ```go
    type AlertScheduler struct {
        engine       *AlertEngine
        emailService *SendGridEmailService
        templates    *EmailTemplateService
        history      *repository.AlertHistoryRepository
        connections  *repository.IntegrationConnectionRepository
        interval     time.Duration
    }
    ```
    - `Start(ctx context.Context)` — ticker loop, runs every 15 minutes (configurable)
    - `RunOnce(ctx context.Context)` — single evaluation pass across all orgs
    - For each org:
      1. Call `engine.EvaluateAll(ctx, orgID)` → get matches
      2. For each match, create `alert_history` record with status `pending`
      3. Render email template based on trigger type
      4. Send email via `EmailService.SendEmail()`
      5. Update `alert_history` status to `sent` (or `failed` with error message)
    - Error handling: individual alert failures don't block other alerts
    - Logging: log each evaluation pass summary (rules evaluated, alerts fired, emails sent, failures)

3. Add configuration to `internal/config/config.go`:
    ```go
    type AlertConfig struct {
        EvalIntervalMin   int     `env:"ALERT_EVAL_INTERVAL_MIN" envDefault:"15"`
        DefaultCooldownHr int     `env:"ALERT_DEFAULT_COOLDOWN_HR" envDefault:"24"`
    }
    ```

4. Wire in `cmd/api/main.go`:
    - Create `AlertEngine` with repos
    - Create `AlertScheduler` with engine + email service
    - Start in background: `go alertScheduler.Start(bgCtx)` alongside `scoreScheduler.Start(bgCtx)`

**Tests:** `internal/service/alert_engine_test.go` —
- `score_below`: customer at 35 with threshold 40 → match; customer at 75 → no match
- `score_drop`: customer dropped 15 points in 7 days → match; customer dropped 5 → no match
- `risk_change`: green→yellow transition → match based on conditions; yellow→green → no match for to=red rule
- `event_trigger`: payment.failed event → match
- Deduplication: alert sent 2 hours ago with 24h cooldown → skip; alert sent 25 hours ago → fire
- Batch: multiple rules, multiple customers, correct set of matches

`internal/service/alert_scheduler_test.go` — RunOnce processes orgs, sends emails, records history, handles failures gracefully

**Acceptance criteria:**

- [ ] All four rule types implemented (`score_below`, `score_drop`, `risk_change`, `event_trigger`)
- [ ] Batch evaluation processes all customers per org
- [ ] Cooldown prevents duplicate alerts (configurable, default 24h)
- [ ] Alert history recorded for every alert attempt
- [ ] Scheduled execution every 15 minutes (configurable)
- [ ] Individual failures don't block other alerts
- [ ] Tests cover each rule type, deduplication, and edge cases

---

### Phase 3 — Score Drop Triggers (Integration with Scoring Engine)

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#141](https://github.com/subculture-collective/pulse-score/issues/141) | Implement score drop alert triggers | high | `internal/service/alert_engine.go` (extend), `internal/service/scoring/scheduler.go` (hook) |

**Depends on:** Phase 2b (#140)

**Details:**

1. Enhance `score_drop` rule evaluation in `AlertEngine`:
    - Query `health_score_history` for the customer's score at T minus N days
    - Calculate delta: `current_score - historical_score`
    - If `|delta| >= rule.conditions.points` and delta is negative → match
    - Trigger data payload:
      ```json
      {
        "customer_id": "uuid",
        "old_score": 72,
        "new_score": 55,
        "delta": -17,
        "days": 7,
        "biggest_contributing_factor": "payment_recency",
        "risk_level": "yellow"
      }
      ```

2. Add helper to `HealthScoreRepository`:
    - `GetScoreAtTime(ctx, customerID, orgID uuid.UUID, at time.Time) (*HealthScore, error)` — get the closest score to the given timestamp from `health_score_history`

3. Add optional real-time hook in `ScoreScheduler.calculateAndStore()`:
    - After `changeDetector.DetectAndRecord()`, optionally trigger alert evaluation for the specific customer
    - Add an `AlertCallback` function field to `ScoreScheduler`:
      ```go
      type ScoreScheduler struct {
          // ... existing fields
          alertCallback func(ctx context.Context, customerID, orgID uuid.UUID)
      }
      ```
    - In `calculateAndStore`, after change detection: if `alertCallback != nil`, call it
    - This enables near-real-time alerting after score changes (in addition to scheduled evaluation)

4. Wire callback in `cmd/api/main.go`:
    ```go
    scoreScheduler.SetAlertCallback(func(ctx context.Context, customerID, orgID uuid.UUID) {
        if matches, err := alertEngine.EvaluateForCustomer(ctx, customerID, orgID); err == nil {
            alertScheduler.ProcessMatches(ctx, matches)
        }
    })
    ```

5. Add `EvaluateForCustomer(ctx, customerID, orgID uuid.UUID) ([]AlertMatch, error)` to `AlertEngine` — evaluate all active rules for a single customer (used by real-time hook)

**Tests:**
- `internal/repository/health_score_test.go` — `GetScoreAtTime` returns closest historical score
- `internal/service/alert_engine_test.go` — score drop detection accuracy, custom thresholds, biggest contributing factor extraction
- Integration: score recalculation triggers alert evaluation → email sent for significant drop

**Acceptance criteria:**

- [ ] Score drops detected accurately using historical comparison
- [ ] Custom thresholds (points, days) respected from rule conditions
- [ ] Alert includes context: old_score, new_score, delta, biggest_contributing_factor
- [ ] Risk boundary crossing detected (e.g., green → yellow)
- [ ] Real-time hook fires after score recalculation
- [ ] `EvaluateForCustomer` evaluates all rules for one customer
- [ ] Tests cover various drop scenarios and edge cases

---

### Phase 4a — Alert Management UI

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#142](https://github.com/subculture-collective/pulse-score/issues/142) | Build alert management UI | high | `web/src/pages/AlertsPage.tsx`, `web/src/components/AlertRuleForm.tsx`, `web/src/components/AlertHistoryTable.tsx`, `web/src/lib/api.ts`, `web/src/pages/settings/SettingsPage.tsx` |

**Depends on:** Phase 2b (#140 — API endpoints available)

**Details:**

1. Add "Alerts" tab to Settings page in `web/src/pages/settings/SettingsPage.tsx`:
    ```tsx
    const tabs = [
      // ...existing tabs
      { path: "alerts", label: "Alerts" },
    ];
    // Add route:
    <Route path="alerts" element={<AlertsTab />} />
    ```

2. Create `web/src/pages/settings/AlertsTab.tsx` — main alerts settings tab:
    - Two sub-sections: "Alert Rules" and "Alert History"
    - Toggle between sections via sub-tabs or toggle buttons

3. Create `web/src/components/AlertRuleList.tsx` — alert rules list:
    - Table columns: Name, Type (badge), Condition (human-readable), Enabled (toggle), Last Triggered, Actions (edit, delete)
    - "Create Rule" button → opens `AlertRuleForm`
    - Enable/disable toggle calls PATCH `/alerts/rules/:id` with `{ is_active: bool }`
    - Delete confirmation dialog

4. Create `web/src/components/AlertRuleForm.tsx` — create/edit form:
    - Mode: create or edit (pre-fills existing values)
    - Fields:
      - **Name** — text input (required)
      - **Description** — textarea (optional)
      - **Trigger Type** — select: Score Drop, Risk Level Change, Payment Failed
      - **Conditions** — dynamic fields based on trigger type:
        - `score_drop` / `score_below`: threshold (number), timeframe in days (number)
        - `risk_change`: from level (select), to level (select)
        - `payment_failed`: no extra conditions
      - **Recipients** — multi-email input (tag-style, validated)
      - **Active** — checkbox
    - Submit: POST for create, PATCH for update
    - Validation: name required, at least 1 recipient, valid email addresses

5. Create `web/src/components/AlertHistoryTable.tsx` — alert history:
    - Table columns: Timestamp, Rule Name, Customer, Status (badge: sent/failed/pending), Channel
    - Status badges: green for sent, red for failed, yellow for pending
    - Pagination controls
    - Filter by status (all, sent, failed, pending)

6. Add API functions to `web/src/lib/api.ts`:
    ```ts
    export interface AlertRule {
      id: string;
      name: string;
      description: string;
      trigger_type: string;
      conditions: Record<string, any>;
      channel: string;
      recipients: string[];
      is_active: boolean;
      created_at: string;
      updated_at: string;
    }

    export interface AlertHistoryItem {
      id: string;
      alert_rule_id: string;
      customer_id: string;
      trigger_data: Record<string, any>;
      channel: string;
      status: string;
      sent_at: string | null;
      error_message: string;
      created_at: string;
    }

    export const alertsApi = {
      listRules: () => api.get<AlertRule[]>("/alerts/rules"),
      getRule: (id: string) => api.get<AlertRule>(`/alerts/rules/${id}`),
      createRule: (data: Partial<AlertRule>) => api.post<AlertRule>("/alerts/rules", data),
      updateRule: (id: string, data: Partial<AlertRule>) => api.patch<AlertRule>(`/alerts/rules/${id}`, data),
      deleteRule: (id: string) => api.delete(`/alerts/rules/${id}`),
      listHistory: (params?: { limit?: number; offset?: number; status?: string }) =>
        api.get<{ items: AlertHistoryItem[]; total: number }>("/alerts/history", { params }),
    };
    ```

7. Add backend handler for alert history to `internal/handler/alert_rule.go`:
    - `ListHistory()` — `GET /api/v1/alerts/history?limit=20&offset=0&status=sent`
    - Wire route in `cmd/api/main.go`

**Tests:**
- Frontend: component tests for AlertRuleForm validation, AlertRuleList rendering, AlertHistoryTable pagination
- Backend: `internal/handler/alert_rule_test.go` — ListHistory endpoint with pagination and filtering

**Acceptance criteria:**

- [ ] "Alerts" tab added to Settings page
- [ ] Rules list shows all configured rules with enable/disable toggle
- [ ] Create form with type-specific condition fields
- [ ] Edit and delete rules work via API
- [ ] Alert history table with status badges and pagination
- [ ] API client functions for all alert endpoints
- [ ] Tests cover CRUD operations and form validation

---

### Phase 4b — Email Delivery Tracking

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#143](https://github.com/subculture-collective/pulse-score/issues/143) | Implement email delivery tracking | medium | `internal/handler/webhook_sendgrid.go`, `internal/repository/alert_history.go` (extend), migration |

**Depends on:** Phase 1 (#138), Phase 2b (#140)

**Details:**

1. Create migration `000016_add_delivery_tracking.up.sql`:
    ```sql
    ALTER TABLE alert_history
        ADD COLUMN delivered_at TIMESTAMPTZ,
        ADD COLUMN opened_at   TIMESTAMPTZ,
        ADD COLUMN clicked_at  TIMESTAMPTZ,
        ADD COLUMN bounced_at  TIMESTAMPTZ,
        ADD COLUMN sendgrid_message_id VARCHAR(255);

    CREATE INDEX idx_alert_history_sendgrid_msg ON alert_history (sendgrid_message_id)
        WHERE sendgrid_message_id IS NOT NULL;
    ```
    And `000016_add_delivery_tracking.down.sql`:
    ```sql
    ALTER TABLE alert_history
        DROP COLUMN IF EXISTS delivered_at,
        DROP COLUMN IF EXISTS opened_at,
        DROP COLUMN IF EXISTS clicked_at,
        DROP COLUMN IF EXISTS bounced_at,
        DROP COLUMN IF EXISTS sendgrid_message_id;
    ```

2. Update `AlertHistory` struct and `SendGridEmailService`:
    - Capture `X-Message-Id` response header from SendGrid after sending → store in `alert_history.sendgrid_message_id`
    - Add to `AlertHistoryRepository`: `UpdateDeliveryStatus(ctx, sendgridMsgID string, event string, timestamp time.Time) error`

3. Create `internal/handler/webhook_sendgrid.go` — SendGrid event webhook:
    ```go
    type SendGridWebhookHandler struct {
        history      *repository.AlertHistoryRepository
        webhookKey   string // SendGrid webhook verification key
    }
    ```
    - `HandleEvents()` — `POST /api/v1/webhooks/sendgrid`
    - Parse SendGrid event payload (array of events)
    - **Signature verification**: Validate `X-Twilio-Email-Event-Webhook-Signature` header using the webhook signing key
    - Supported events: `delivered`, `bounce`, `open`, `click`, `spam_report`
    - Map events to `alert_history` columns:
      - `delivered` → set `delivered_at`
      - `bounce` → set `bounced_at`, update status to `failed`
      - `open` → set `opened_at`
      - `click` → set `clicked_at`
      - `spam_report` → set status to `failed`, log warning

4. Add config:
    ```go
    type SendGridConfig struct {
        // ...existing
        WebhookVerifyKey string `env:"SENDGRID_WEBHOOK_VERIFY_KEY"`
    }
    ```

5. Wire route in `cmd/api/main.go` — **no auth middleware** (webhook is authenticated via signature):
    ```go
    r.Post("/webhooks/sendgrid", sendgridWebhookHandler.HandleEvents)
    ```

**Tests:** `internal/handler/webhook_sendgrid_test.go` — valid signature accepted, invalid signature rejected (403), each event type updates correct column, unknown events ignored, malformed payload returns 400

**Acceptance criteria:**

- [ ] SendGrid webhook endpoint processes events
- [ ] Signature verification prevents spoofing
- [ ] Alert history updated with delivery timestamps (delivered_at, opened_at, clicked_at, bounced_at)
- [ ] Bounce handling marks alert as failed
- [ ] SendGrid message ID stored on send for correlation
- [ ] Tests cover webhook event processing and signature verification

---

### Phase 5a — Notification Preferences UI

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#144](https://github.com/subculture-collective/pulse-score/issues/144) | Build alert notification preferences UI | medium | `internal/repository/notification_preferences.go`, `internal/service/notification_preferences.go`, `internal/handler/notification_preferences.go`, `web/src/components/NotificationPreferences.tsx`, migration |

**Depends on:** Phase 4a (#142)

**Details:**

1. Create migration `000017_create_notification_preferences.up.sql`:
    ```sql
    CREATE TABLE notification_preferences (
        id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
        org_id         UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
        alert_type     VARCHAR(50) NOT NULL,  -- score_drop, risk_change, payment_failed, digest
        channel_email  BOOLEAN NOT NULL DEFAULT true,
        channel_inapp  BOOLEAN NOT NULL DEFAULT true,
        frequency      VARCHAR(20) NOT NULL DEFAULT 'immediate'
                       CHECK (frequency IN ('immediate', 'daily', 'weekly', 'muted')),
        created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        CONSTRAINT notification_prefs_unique UNIQUE (user_id, org_id, alert_type)
    );

    CREATE INDEX idx_notification_prefs_user ON notification_preferences (user_id, org_id);

    CREATE TRIGGER set_notification_prefs_updated_at
        BEFORE UPDATE ON notification_preferences
        FOR EACH ROW
        EXECUTE FUNCTION trigger_set_updated_at();
    ```

2. Create `internal/repository/notification_preferences.go`:
    ```go
    type NotificationPreference struct {
        ID           uuid.UUID
        UserID       uuid.UUID
        OrgID        uuid.UUID
        AlertType    string   // score_drop, risk_change, payment_failed, digest
        ChannelEmail bool
        ChannelInApp bool
        Frequency    string   // immediate, daily, weekly, muted
        CreatedAt    time.Time
        UpdatedAt    time.Time
    }
    ```
    - `GetByUser(ctx, userID, orgID uuid.UUID) ([]*NotificationPreference, error)`
    - `Upsert(ctx, pref *NotificationPreference) error` — INSERT ON CONFLICT UPDATE
    - `ShouldNotify(ctx, userID, orgID uuid.UUID, alertType, channel string) (bool, error)` — check if user wants this notification

3. Create `internal/service/notification_preferences.go`:
    - `Get(ctx, userID, orgID uuid.UUID) ([]*repository.NotificationPreference, error)` — return preferences (create defaults if none exist)
    - `Update(ctx, userID, orgID uuid.UUID, prefs []UpdatePrefRequest) error` — batch update
    - Default preferences: email + immediate for critical (score_drop, payment_failed), email + daily for informational (digest)

4. Create `internal/handler/notification_preferences.go`:
    - `Get()` — `GET /api/v1/users/me/notification-preferences`
    - `Update()` — `PUT /api/v1/users/me/notification-preferences`

5. Integrate preferences into `AlertScheduler`:
    - Before sending an email, check `NotificationPreferenceRepository.ShouldNotify(userID, orgID, alertType, "email")`
    - If frequency is `muted`, skip
    - If frequency is `daily` or `weekly`, queue for digest instead of immediate send

6. Create `web/src/components/NotificationPreferences.tsx`:
    - Table layout: rows = alert types, columns = Channel (Email, In-App), Frequency (dropdown)
    - Alert types: Score Drop, Risk Level Change, Payment Failed, Weekly Digest
    - Frequency options: Immediate, Daily Digest, Weekly Digest, Muted
    - Save button → PUT to API
    - Toast on success

7. Add to Settings page as a section within the Profile or a dedicated "Notifications" tab

**Tests:**
- `internal/repository/notification_preferences_test.go` — upsert, defaults, ShouldNotify logic
- `internal/service/notification_preferences_test.go` — default creation, update validation
- Frontend: component renders current preferences, saves changes

**Acceptance criteria:**

- [ ] Notification preferences table created with per-user, per-alert-type settings
- [ ] Preferences form renders with current settings
- [ ] Channel and frequency selections saved
- [ ] Mute function prevents alert delivery
- [ ] Backend respects preferences when sending alerts
- [ ] Default preferences created for new users
- [ ] Tests cover preference save, retrieval, and ShouldNotify logic

---

### Phase 5b — In-App Notification System

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#145](https://github.com/subculture-collective/pulse-score/issues/145) | Implement in-app notification system | high | `internal/repository/notification.go`, `internal/service/notification.go`, `internal/handler/notification.go`, `web/src/components/NotificationBell.tsx`, `web/src/components/NotificationDropdown.tsx`, migration |

**Depends on:** Phase 2b (#140), Phase 5a (#144 — preferences)

**Details:**

1. Create migration `000018_create_notifications.up.sql`:
    ```sql
    CREATE TABLE notifications (
        id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
        org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
        type        VARCHAR(50) NOT NULL,     -- score_drop, risk_change, payment_failed, system
        title       VARCHAR(255) NOT NULL,
        body        TEXT NOT NULL,
        link        VARCHAR(500),             -- relative URL to navigate to
        is_read     BOOLEAN NOT NULL DEFAULT false,
        created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

    CREATE INDEX idx_notifications_user_unread ON notifications (user_id, org_id, is_read, created_at DESC);
    CREATE INDEX idx_notifications_user_created ON notifications (user_id, org_id, created_at DESC);
    ```

2. Create `internal/repository/notification.go`:
    ```go
    type Notification struct {
        ID        uuid.UUID
        UserID    uuid.UUID
        OrgID     uuid.UUID
        Type      string
        Title     string
        Body      string
        Link      string
        IsRead    bool
        CreatedAt time.Time
    }
    ```
    - `Create(ctx, notif *Notification) error`
    - `List(ctx, userID, orgID uuid.UUID, unreadOnly bool, limit, offset int) ([]*Notification, error)`
    - `CountUnread(ctx, userID, orgID uuid.UUID) (int, error)`
    - `MarkRead(ctx, id, userID uuid.UUID) error`
    - `MarkAllRead(ctx, userID, orgID uuid.UUID) error`

3. Create `internal/service/notification.go`:
    - `Create(ctx, userID, orgID uuid.UUID, notifType, title, body, link string) error` — create after checking preferences
    - `List(ctx, userID, orgID uuid.UUID, unreadOnly bool, limit, offset int) ([]*repository.Notification, error)`
    - `CountUnread(ctx, userID, orgID uuid.UUID) (int, error)`
    - `MarkRead(ctx, id, userID uuid.UUID) error`
    - `MarkAllRead(ctx, userID, orgID uuid.UUID) error`

4. Create `internal/handler/notification.go`:
    - `List()` — `GET /api/v1/notifications?unread=true&limit=20&offset=0`
    - `CountUnread()` — `GET /api/v1/notifications/unread-count`
    - `MarkRead()` — `PUT /api/v1/notifications/:id/read`
    - `MarkAllRead()` — `PUT /api/v1/notifications/read-all`

5. Wire routes in `cmd/api/main.go`:
    ```go
    r.Route("/notifications", func(r chi.Router) {
        r.Get("/", notificationHandler.List)
        r.Get("/unread-count", notificationHandler.CountUnread)
        r.Put("/{id}/read", notificationHandler.MarkRead)
        r.Put("/read-all", notificationHandler.MarkAllRead)
    })
    ```

6. Integrate into `AlertScheduler`:
    - After sending email alert, also create in-app notification for each recipient who is a user in the org
    - Check `notification_preferences.channel_inapp` before creating
    - Notification body: short summary (e.g., "Customer Acme Corp's health score dropped from 72 to 55")
    - Link: `/customers/{customer_id}` for customer-specific alerts, `/dashboard` for digests

7. Create `web/src/components/NotificationBell.tsx`:
    - Bell icon in Header component
    - Badge showing unread count (hidden when 0)
    - Poll `GET /api/v1/notifications/unread-count` every 30 seconds
    - Click opens `NotificationDropdown`

8. Create `web/src/components/NotificationDropdown.tsx`:
    - Lists recent notifications (last 20)
    - Each item: icon (by type), title, body (truncated), timestamp (relative: "2h ago")
    - Unread items have a visual indicator (blue dot or bold text)
    - Click notification: mark as read + navigate to link
    - "Mark All Read" button at top
    - "View All" link (optional, for future full page)

9. Add `NotificationBell` to `web/src/components/Header.tsx`

10. Add API functions to `web/src/lib/api.ts`:
    ```ts
    export interface NotificationItem {
        id: string;
        type: string;
        title: string;
        body: string;
        link: string | null;
        is_read: boolean;
        created_at: string;
    }

    export const notificationsApi = {
        list: (params?: { unread?: boolean; limit?: number; offset?: number }) =>
            api.get<NotificationItem[]>("/notifications", { params }),
        countUnread: () => api.get<{ count: number }>("/notifications/unread-count"),
        markRead: (id: string) => api.put(`/notifications/${id}/read`),
        markAllRead: () => api.put("/notifications/read-all"),
    };
    ```

**Tests:**
- `internal/repository/notification_test.go` — CRUD, CountUnread, MarkRead, MarkAllRead
- `internal/handler/notification_test.go` — all endpoints, pagination, unread filter
- Frontend: NotificationBell renders count, dropdown lists items, mark read works

**Acceptance criteria:**

- [ ] Notifications table created
- [ ] Bell icon shows unread count in header
- [ ] Dropdown lists recent notifications
- [ ] Mark as read and mark all read work
- [ ] Click notification navigates to relevant page
- [ ] Polling fetches new notifications every 30 seconds
- [ ] In-app notifications created alongside email alerts
- [ ] Preferences respected (channel_inapp check)
- [ ] Tests cover notification lifecycle

---

## Implementation Summary

| Phase | Issues | Parallel? | Key Deliverables |
|-------|--------|-----------|------------------|
| **1** | #138 | — | Email service interface, retry logic, AlertHistoryRepository |
| **2a** | #139 | Yes (with 2b) | HTML email templates (4 types), template rendering service |
| **2b** | #140 | Yes (with 2a) | Alert evaluation engine (4 rule types), alert scheduler |
| **3** | #141 | — | Score drop triggers, real-time hook into score scheduler |
| **4a** | #142 | Yes (with 4b) | Alert management UI (rules list, form, history table) |
| **4b** | #143 | Yes (with 4a) | SendGrid webhook handler, delivery tracking columns |
| **5a** | #144 | Yes (with 5b) | Notification preferences table, UI, backend integration |
| **5b** | #145 | Yes (with 5a) | In-app notifications (bell, dropdown, API, polling) |

## New Files Summary

### Backend (Go)
| File | Purpose |
|------|---------|
| `internal/service/email.go` | `EmailService` interface |
| `internal/service/email_templates.go` | Template rendering service |
| `internal/service/alert_engine.go` | Rule evaluation engine |
| `internal/service/alert_scheduler.go` | Scheduled alert evaluation + delivery |
| `internal/repository/alert_history.go` | AlertHistory CRUD + deduplication queries |
| `internal/repository/notification_preferences.go` | Per-user notification preferences |
| `internal/repository/notification.go` | In-app notifications storage |
| `internal/service/notification_preferences.go` | Preferences business logic |
| `internal/service/notification.go` | Notification creation + management |
| `internal/handler/webhook_sendgrid.go` | SendGrid event webhook |
| `internal/handler/notification_preferences.go` | Preferences API endpoints |
| `internal/handler/notification.go` | Notifications API endpoints |
| `internal/templates/emails/base.html` | Base email layout |
| `internal/templates/emails/score_drop.html` | Score drop alert template |
| `internal/templates/emails/at_risk.html` | Risk change alert template |
| `internal/templates/emails/payment_failed.html` | Payment failure template |
| `internal/templates/emails/weekly_digest.html` | Weekly digest template |

### Migrations
| File | Purpose |
|------|---------|
| `migrations/000016_add_delivery_tracking.{up,down}.sql` | Delivery status columns on alert_history |
| `migrations/000017_create_notification_preferences.{up,down}.sql` | User notification preferences |
| `migrations/000018_create_notifications.{up,down}.sql` | In-app notifications table |

### Frontend (React/TypeScript)
| File | Purpose |
|------|---------|
| `web/src/pages/settings/AlertsTab.tsx` | Alerts settings tab |
| `web/src/components/AlertRuleList.tsx` | Alert rules list with enable/disable |
| `web/src/components/AlertRuleForm.tsx` | Create/edit alert rule form |
| `web/src/components/AlertHistoryTable.tsx` | Alert history with status badges |
| `web/src/components/NotificationPreferences.tsx` | Notification preferences form |
| `web/src/components/NotificationBell.tsx` | Bell icon with unread count |
| `web/src/components/NotificationDropdown.tsx` | Notification list dropdown |

### Existing Files to Modify
| File | Change |
|------|--------|
| `internal/service/sendgrid.go` | Add `SendEmail()`, retry logic, capture message ID |
| `internal/service/scoring/scheduler.go` | Add `alertCallback` hook |
| `internal/handler/alert_rule.go` | Add `ListHistory()` endpoint |
| `internal/config/config.go` | Add `AlertConfig` section |
| `cmd/api/main.go` | Wire AlertEngine, AlertScheduler, NotificationHandler, webhook route |
| `web/src/pages/settings/SettingsPage.tsx` | Add "Alerts" tab |
| `web/src/components/Header.tsx` | Add `NotificationBell` |
| `web/src/lib/api.ts` | Add `alertsApi` and `notificationsApi` |

## Configuration

New environment variables:

| Variable | Default | Purpose |
|----------|---------|---------|
| `ALERT_EVAL_INTERVAL_MIN` | `15` | Alert evaluation frequency in minutes |
| `ALERT_DEFAULT_COOLDOWN_HR` | `24` | Default cooldown between duplicate alerts |
| `SENDGRID_WEBHOOK_VERIFY_KEY` | — | SendGrid webhook signature verification key |
