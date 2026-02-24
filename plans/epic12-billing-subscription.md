# Execution Plan: Epic 12 — Billing & Subscription (#20)

## Overview

**Epic:** [#20 — Billing & Subscription](https://github.com/subculture-collective/pulse-score/issues/20)  
**Sub-issues:** #153–#161 (9 issues)  
**Scope:** Implement PulseScore-owned Stripe billing (separate from Stripe data integration): plan catalog + Checkout + billing webhooks + subscription tracking + feature gating + pricing/checkout/subscription management UI + Stripe Customer Portal.

---

## Current State (What already exists)

- **Organization billing fields already exist:** `organizations.plan` and `organizations.stripe_customer_id` in migration `000002_create_organizations.up.sql`.
- **Stripe integration is already implemented for customer-data sync**, not SaaS billing:
  - OAuth + sync endpoints in `internal/handler/integration_stripe.go` and `cmd/api/main.go`
  - Webhook endpoint `POST /api/v1/webhooks/stripe` handled by `internal/service/stripe_webhook.go`
  - Stripe subscription/payment repositories currently model **customer telemetry data** (`stripe_subscriptions`, `stripe_payments`) in `internal/repository/stripe_data.go`
- **Frontend billing surface is a placeholder** in `web/src/pages/settings/BillingTab.tsx`.
- **Landing pricing section exists** in `web/src/components/landing/PricingSection.tsx`, but currently static and not wired to checkout flow.

### Key Gap

Epic 12 requires a **new billing domain** for PulseScore product subscriptions. This must remain isolated from existing Stripe data-integration services to avoid token/config confusion.

---

## Architecture Guardrails for Epic 12

1. **Separate Stripe configs for billing vs integration**
   - Keep existing `StripeConfig` for integration OAuth/data sync untouched.
   - Add billing-specific config/env (e.g., `STRIPE_BILLING_SECRET_KEY`, `STRIPE_BILLING_PUBLISHABLE_KEY`, `STRIPE_BILLING_WEBHOOK_SECRET`).

2. **Separate webhook endpoint**
   - Keep `POST /api/v1/webhooks/stripe` for integration events.
   - Add `POST /api/v1/webhooks/stripe-billing` for billing lifecycle events.

3. **Dedicated billing service layer**
   - Create billing domain under `internal/service/billing/*` and `internal/repository/*billing*`.

4. **Plan limits are enforced server-side**
   - UI hints are informational only.
   - Middleware/service checks are the source of truth for customer/integration limits.

---

## Dependency Graph

```text
#153 Billing plans catalog/config
   ├──► #154 Checkout session endpoint
   ├──► #157 Feature-gating limits
   └──► #158 Pricing page

#154 Checkout
   └──► #155 Billing webhook handler

#155 Billing webhook handler
   └──► #156 Subscription tracking model/service

#156 Subscription tracking
   ├──► #157 Feature gating middleware
   ├──► #160 Subscription management page
   └──► #161 Portal session endpoint

#158 Pricing page
   └──► #159 Frontend checkout flow

#154 + #159 + #161 + #156
   └──► #160 Subscription management page
```

---

## Execution Phases

### Phase 1 — Billing Foundations (Plan catalog + subscription schema)

| Issue | Title | Priority | Files to Create/Modify |
|------|------|------|------|
| [#153](https://github.com/subculture-collective/pulse-score/issues/153) | Configure Stripe billing products and prices | critical | `internal/billing/plans.go`, `internal/config/config.go`, `.env.example`, docs under `docs/` |
| [#156](https://github.com/subculture-collective/pulse-score/issues/156) | Implement subscription tracking model | high | `migrations/000021_*`, `internal/repository/org_subscription.go`, `internal/service/billing/subscription.go` |

#### #153 Implementation details

1. Create `internal/billing/plans.go` as canonical plan definitions:
   - `free`, `growth`, `scale`
   - Monthly + annual Stripe price IDs
   - Limits: `customer_limit`, `integration_limit`
   - Feature flags (e.g., playbooks, ai insights)
2. Add billing Stripe config section in `internal/config/config.go`:
   - `BillingStripeSecretKey`
   - `BillingStripePublishableKey`
   - `BillingStripeWebhookSecret`
   - Optional: `BillingStripePortalReturnURL`
3. Add startup validation so app fails fast if billing keys are missing in non-dev environments.
4. Add a small verification utility/service to ensure configured Stripe price IDs exist and carry required metadata (`tier`, `customer_limit`, `integration_limit`).

**Tests**
- Unit tests for plan lookup and limit access (`GetPlanByTier`, `GetLimits`, annual/monthly mapping)
- Config loading tests for new billing env vars

#### #156 Implementation details

1. Add new migration for org-level subscription state (recommended table `org_subscriptions`):
   - `org_id` (unique)
   - `stripe_subscription_id`
   - `stripe_customer_id`
   - `plan_tier`
   - `status`
   - `current_period_start`, `current_period_end`
   - `cancel_at_period_end`
   - `created_at`, `updated_at`
2. Add repository `internal/repository/org_subscription.go` with:
   - `UpsertByOrg`
   - `GetByOrg`
   - `GetByStripeSubscriptionID`
3. Add service `internal/service/billing/subscription.go` with query helpers:
   - `GetCurrentPlan(orgID)`
   - `IsActive(orgID)`
   - `GetUsageLimits(orgID)`
4. Default new organizations to Free tier if no subscription row exists.

**Tests**
- Repository CRUD/upsert tests
- Service tests for default free fallback and status transitions

---

### Phase 2 — Checkout + Portal backend APIs

| Issue | Title | Priority | Files to Create/Modify |
|------|------|------|------|
| [#154](https://github.com/subculture-collective/pulse-score/issues/154) | Implement Stripe Checkout session creation | critical | `internal/service/billing/checkout.go`, `internal/handler/billing.go`, `cmd/api/main.go` |
| [#161](https://github.com/subculture-collective/pulse-score/issues/161) | Implement Stripe Customer Portal redirect | high | `internal/service/billing/portal.go`, `internal/handler/billing.go`, `cmd/api/main.go` |

#### #154 Implementation details

1. Add `POST /api/v1/billing/checkout` (admin-gated).
2. Request: `{ priceId: string, annual: boolean }` (or `{ tier, cycle }` mapped server-side to configured price ID).
3. Ensure org has Stripe customer:
   - Reuse `organizations.stripe_customer_id` if present.
   - Create new Stripe customer if absent and persist back.
4. Create Stripe Checkout Session (`mode=subscription`) with metadata:
   - `org_id`, `user_id`, `tier`, `cycle`.
5. Return `{ url }` for frontend redirect.

**Tests**
- Handler tests for validation/authz
- Service tests for customer-creation path and session creation path

#### #161 Implementation details

1. Add `POST /api/v1/billing/portal-session`.
2. Ensure Stripe customer exists before creating portal session.
3. Return hosted portal URL with return path `/settings/billing`.
4. Expose in billing service for UI integration.

**Tests**
- Portal session created with right customer
- Error path when no customer + Stripe creation failure

---

### Phase 3 — Billing webhook and subscription state sync

| Issue | Title | Priority | Files to Create/Modify |
|------|------|------|------|
| [#155](https://github.com/subculture-collective/pulse-score/issues/155) | Implement billing webhook handler | critical | `internal/service/billing/webhook.go`, `internal/handler/webhook_billing.go`, `cmd/api/main.go` |

#### #155 Implementation details

1. Add public endpoint `POST /api/v1/webhooks/stripe-billing`.
2. Verify billing webhook signatures with billing webhook secret (not integration webhook secret).
3. Handle events:
   - `checkout.session.completed`
   - `customer.subscription.created`
   - `customer.subscription.updated`
   - `customer.subscription.deleted`
   - `invoice.payment_succeeded`
   - `invoice.payment_failed`
4. Update `org_subscriptions` and `organizations.plan` in one transaction where possible.
5. On `payment_failed`, flag status and trigger notification hooks.
6. Add idempotency guard (event ID dedupe table or durable event log) to safely replay events.

**Tests**
- Signature verification success/failure
- Idempotency checks
- Event-specific state transition tests

---

### Phase 4 — Enforce plan limits in backend

| Issue | Title | Priority | Files to Create/Modify |
|------|------|------|------|
| [#157](https://github.com/subculture-collective/pulse-score/issues/157) | Implement feature gating middleware | critical | `internal/middleware/feature_gate.go`, `internal/service/billing/limits.go`, selected handlers in `internal/handler/*` |

#### #157 Implementation details

1. Implement tier limit checks for:
   - Customer creation operations (max customers)
   - Integration connection operations (max integrations)
2. Add feature access checks:
   - `CanAccess(orgID, featureName)`
3. Return `402 Payment Required` with structured payload:
   - current plan
   - limit reached
   - recommended upgrade tier
4. Apply middleware/checks to relevant routes/handlers:
   - customer create/upsert paths
   - integration connect endpoints
5. Ensure downgrade behavior enforces Free limits for **new actions** while preserving existing over-limit data until user acts.

**Tests**
- Per-tier customer/integration limit tests
- Feature gate allow/deny matrix tests
- Proper 402 payload contract tests

---

### Phase 5 — Pricing page + checkout UX

| Issue | Title | Priority | Files to Create/Modify |
|------|------|------|------|
| [#158](https://github.com/subculture-collective/pulse-score/issues/158) | Build pricing page component | high | `web/src/pages/PricingPage.tsx`, `web/src/lib/api.ts`, `web/src/components/landing/PricingSection.tsx` |
| [#159](https://github.com/subculture-collective/pulse-score/issues/159) | Build checkout flow and success handling | high | `web/src/hooks/useCheckout.ts`, `web/src/pages/settings/BillingTab.tsx`, router integration |

#### #158 Implementation details

1. Create a reusable pricing component/page with Free/Growth/Scale comparison.
2. Add monthly/annual toggle and savings badge.
3. Highlight current plan for authenticated orgs.
4. CTA buttons call checkout hook/API for paid plans.
5. Optionally replace static landing `PricingSection` internals with shared pricing model to avoid drift.

**Tests**
- Toggle behavior tests
- Current plan highlighting tests
- CTA callback wiring tests

#### #159 Implementation details

1. Implement `useCheckout` hook:
   - Loading state
   - API call to `/billing/checkout`
   - Redirect to Stripe URL
2. Handle return states in billing settings page:
   - `?checkout=success` → success toast + refresh subscription
   - `?checkout=cancelled` → informative notice
3. Refresh org subscription data after successful return.

**Tests**
- Hook success/failure/loading tests
- Success/cancel query param behavior tests

---

### Phase 6 — Subscription management UI

| Issue | Title | Priority | Files to Create/Modify |
|------|------|------|------|
| [#160](https://github.com/subculture-collective/pulse-score/issues/160) | Build subscription management page | high | `web/src/components/billing/SubscriptionManager.tsx`, `web/src/pages/settings/BillingTab.tsx`, `web/src/lib/api.ts`, backend billing handler for `GET /billing/subscription` |

#### #160 Implementation details

1. Replace placeholder `BillingTab` with real manager component.
2. Add `GET /api/v1/billing/subscription` backend response:
   - current tier, status, billing cycle, renewal date
   - usage counters (customers/integrations) + limits
3. UI actions:
   - Change plan (launch checkout)
   - Cancel at period end (confirmation modal)
   - Open Stripe portal (`POST /billing/portal-session`)
4. Ensure plan/usage is consistent with middleware enforcement logic.

**Tests**
- Component rendering with active/canceled/past_due states
- Cancel confirmation flow tests
- Portal redirect action tests

---

## API Surface to Add (Epic 12)

### Public
- `POST /api/v1/webhooks/stripe-billing`

### Protected (admin where appropriate)
- `POST /api/v1/billing/checkout`
- `POST /api/v1/billing/portal-session`
- `GET /api/v1/billing/subscription`
- `POST /api/v1/billing/cancel` (if cancellation endpoint is implemented explicitly)

---

## Suggested PR Slicing

1. **PR-1:** #153 + config/env + plan catalog tests
2. **PR-2:** #156 schema/repository/service foundation
3. **PR-3:** #154 checkout endpoint + customer creation
4. **PR-4:** #155 billing webhook + idempotency
5. **PR-5:** #157 feature gating middleware
6. **PR-6:** #158 pricing page + shared plan presentation
7. **PR-7:** #159 checkout UX + success/cancel handling
8. **PR-8:** #161 portal endpoint
9. **PR-9:** #160 subscription management page + usage meters

---

## Risks & Mitigations

1. **Confusing Stripe integration keys with billing keys**  
   _Mitigation:_ strict config separation + explicit naming (`STRIPE_BILLING_*`).

2. **Event replay / duplicate webhook side effects**  
   _Mitigation:_ durable idempotency key storage by Stripe event ID.

3. **Plan drift between backend and frontend pricing copy**  
   _Mitigation:_ central shared plan model or API-driven pricing payload.

4. **Incorrect limit enforcement on downgrade**  
   _Mitigation:_ define policy up-front (block new actions only) and encode in tests.

---

## Definition of Done (Epic 12)

Epic #20 is complete when all sub-issues #153–#161 are merged and validated end-to-end:

- Checkout works from pricing/settings to Stripe and back
- Billing webhook updates org subscription state reliably in near real-time
- Feature limits are enforced for customer/integration actions
- Billing settings page shows current plan + usage + renewal details
- Stripe Customer Portal opens successfully for payment/invoice self-service
- Test coverage exists for all new handlers/services/middleware and key UI states
