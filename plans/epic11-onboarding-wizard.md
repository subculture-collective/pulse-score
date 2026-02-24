# Execution Plan: Epic 11 — Onboarding Wizard (#16)

## Overview

**Epic:** [#16 — Onboarding Wizard](https://github.com/subculture-collective/pulse-score/issues/16)
**Sub-issues:** #146–#152 (7 issues)
**Scope:** Build a resumable, multi-step onboarding flow that takes a new org from welcome/profile setup through integration connection and first score preview, then tracks completion and funnel analytics.

This plan is aligned to the existing layered architecture (Go backend + React frontend) and follows the style used in prior epic plans.

---

## Epic Goals to Satisfy

- Multi-step wizard with progress indicator
- Welcome/org profile setup step
- Stripe, HubSpot, Intercom connection steps (HubSpot/Intercom skippable)
- Initial score generation preview
- Resume from last incomplete step
- Redirect to dashboard when onboarding is complete
- Track onboarding funnel metrics (complete/skip/drop-off/time-per-step)

---

## Dependency Graph

```text
#146 Wizard Shell
   ├──► #147 Welcome & Org Setup
   ├──► #148 Stripe Connect Step
   └──► #149 HubSpot + Intercom Steps
              \
               \
                └──► #150 Score Preview Step

#146 + (#147/#148/#149/#150) ───► #151 Tracking + Resume
#146 + #151 ─────────────────────► #152 Completion Analytics
```

---

## Execution Phases

### Phase 1 — Wizard Foundation

| Issue | Title | Priority | Files to Create/Modify |
|------|------|------|------|
| [#146](https://github.com/subculture-collective/pulse-score/issues/146) | Build multi-step wizard shell component | high | `web/src/components/wizard/WizardShell.tsx`, `web/src/components/wizard/WizardStep.tsx`, `web/src/components/wizard/WizardProgress.tsx`, `web/src/context/onboarding/*`, onboarding route wiring |

**Implementation details:**

1. Create a typed wizard state model (step index, completed/skipped steps, per-step payloads).
2. Build shell primitives:
   - `WizardShell`: layout, navigation controls, guardrails
   - `WizardProgress`: active/complete/upcoming visual states
   - `WizardStep`: render wrapper + per-step validation contract
3. Add step-level validation hooks so `Next` is blocked when invalid.
4. Support route state via query parameter (`/onboarding?step=N`) while preserving in-memory state.
5. Ensure navigation supports Back/Next/Skip/Done by per-step capability flags.

**Tests:**
- Unit/component tests for progress states, navigation transitions, and validation blocking.
- Route query sync tests (`step` param round-trip).

---

### Phase 2 — Core Onboarding Steps (Parallel)

| Issue | Title | Priority | Files to Create/Modify |
|------|------|------|------|
| [#147](https://github.com/subculture-collective/pulse-score/issues/147) | Build welcome and organization setup step | high | `web/src/components/wizard/steps/WelcomeStep.tsx`, `web/src/lib/api.ts` (org profile call), org settings integration |
| [#148](https://github.com/subculture-collective/pulse-score/issues/148) | Build Stripe integration connection step | high | `web/src/components/wizard/steps/StripeConnectStep.tsx`, integration status API wiring |
| [#149](https://github.com/subculture-collective/pulse-score/issues/149) | Build HubSpot and Intercom connection steps | high | `web/src/components/wizard/steps/HubSpotConnectStep.tsx`, `web/src/components/wizard/steps/IntercomConnectStep.tsx`, integration status API wiring |

#### #147 — Welcome + Organization Setup

1. Add onboarding intro content (value proposition + setup expectation).
2. Render org form with:
   - org name (required, pre-filled)
   - industry (select)
   - company size (range/select)
3. Persist form on Next via org update endpoint.
4. Register step payload in wizard context to support resume/review.

**Tests:** form validation (name required), prefill behavior, save-on-next API call success/failure handling.

#### #148 — Stripe Connect Step

1. Explain data usage scope (read-only customer/subscription/payment access).
2. Add Connect CTA to launch OAuth flow.
3. On callback status update, render connected state + account metadata.
4. Support explicit skip path (“connect later”).
5. Add resilient error UI + retry path.

**Tests:** disconnected/connected/error render states, OAuth launch callback handling, skip handling.

#### #149 — HubSpot + Intercom Steps

1. Implement the same UX contract used by Stripe step:
   - explanation
   - connect action
   - confirmation state
   - skip option
2. Keep both steps independently skippable.
3. Show connected account details when available.

**Tests:** both steps fully covered (connected/disconnected/skipped/error), parity checks against Stripe step behavior.

---

### Phase 3 — Score Generation Preview

| Issue | Title | Priority | Files to Create/Modify |
|------|------|------|------|
| [#150](https://github.com/subculture-collective/pulse-score/issues/150) | Build score generation preview step | high | `web/src/components/wizard/steps/ScorePreviewStep.tsx`, sync/score preview API client additions, lightweight chart component |

**Implementation details:**

1. Trigger initial sync when entering step for connected integrations.
2. Track per-integration progress and expose clear statuses.
3. On completion, render:
   - score distribution preview (mini chart)
   - top 3 at-risk customers teaser
4. Handle empty integration scenario gracefully with actionable choice.
5. Add final CTA to dashboard.

**Tests:**
- Sync kickoff on step entry.
- Progress state transitions.
- Preview rendering after completion.
- Empty-state behavior with no integrations.

---

### Phase 4 — Onboarding Tracking and Resume

| Issue | Title | Priority | Files to Create/Modify |
|------|------|------|------|
| [#151](https://github.com/subculture-collective/pulse-score/issues/151) | Implement onboarding tracking and resume | high | migration (`onboarding_status`), `internal/repository/*onboarding*`, `internal/service/*onboarding*`, `internal/handler/*onboarding*`, frontend auth/route guard + settings rerun entry |

**Implementation details:**

1. Add backend persistence for onboarding state (JSONB on org or dedicated table).
2. Track:
   - `completed_steps[]`
   - `skipped_steps[]`
   - `current_step`
   - `completed_at`
3. Add endpoints for status read + step update.
4. On login/app bootstrap:
   - if incomplete, redirect to onboarding at last incomplete step
   - if complete, bypass onboarding
5. Add “Re-run onboarding” action from settings.

**Tests:**
- Status persistence per org.
- Resume behavior across sessions.
- Redirect/bypass logic for complete vs incomplete organizations.
- Re-run pathway resets current_step correctly.

---

### Phase 5 — Onboarding Funnel Analytics

| Issue | Title | Priority | Files to Create/Modify |
|------|------|------|------|
| [#152](https://github.com/subculture-collective/pulse-score/issues/152) | Build onboarding completion analytics | medium | onboarding analytics event writer in backend service layer, repository support for querying/reporting metrics, optional telemetry hooks in wizard client |

**Implementation details:**

1. Emit events for:
   - `step_started`
   - `step_completed`
   - `step_skipped`
   - `onboarding_completed`
   - `onboarding_abandoned`
2. Store events in an existing events table (or dedicated table if needed by query profile).
3. Capture timing per step (`started_at`, `completed_at`, duration derivation).
4. Ensure metrics can compute:
   - step completion rate
   - skip rate
   - average time per step
   - overall completion rate

**Tests:**
- Event emission on each transition type.
- Duration correctness.
- Metric query correctness for completion/skip/time calculations.

---

## Cross-Issue Technical Contracts

To prevent rework between issues, lock these contracts early:

1. **Canonical step IDs** (e.g., `welcome`, `stripe`, `hubspot`, `intercom`, `preview`).
2. **Wizard state schema** shared by frontend + backend status endpoints.
3. **Integration step status model** (`disconnected|connecting|connected|error|skipped`).
4. **Step validation interface** so every step implements `canProceed` consistently.
5. **Analytics event payload shape** (`org_id`, `step_id`, `event_type`, `ts`, optional metadata).

---

## Delivery Checklist by Sub-Issue

- [ ] #146 Wizard shell + progress + nav + validation + tests
- [ ] #147 Welcome/org setup step + save + tests
- [ ] #148 Stripe connect step + skip/error states + tests
- [ ] #149 HubSpot/Intercom steps + skip/error states + tests
- [ ] #150 Score preview step + sync progress + empty state + tests
- [ ] #151 Tracking/resume backend + redirect logic + rerun + tests
- [ ] #152 Analytics event pipeline + metrics correctness + tests

---

## Recommended PR Slicing

To keep implementation manageable and reviewable:

1. **PR-1:** #146 only (foundation)
2. **PR-2:** #147 + #148 + #149 (step UIs)
3. **PR-3:** #150 (preview + sync UX)
4. **PR-4:** #151 (tracking/resume full-stack)
5. **PR-5:** #152 (analytics + instrumentation)

---

## Risks and Mitigations

1. **OAuth callback race conditions** across integration steps
   _Mitigation:_ centralize integration status polling and callback reconciliation in one hook.

2. **Resume logic drift between FE and BE state**
   _Mitigation:_ backend becomes source of truth; frontend context hydrates from backend on entry/login.

3. **Score preview latency causing abandonment**
   _Mitigation:_ explicit progress granularity, optimistic UI copy, timeout fallback with dashboard CTA.

4. **Inconsistent analytics due to skipped transitions**
   _Mitigation:_ emit analytics in a single transition function used by all navigation actions.

---

## Definition of Done (Epic)

Epic #16 is complete when:

- All sub-issues #146–#152 are merged
- Epic acceptance criteria are satisfied end-to-end in a full onboarding walkthrough
- Resume behavior works across logout/login and page refresh
- Dashboard redirect after completion is verified
- Funnel events are queryable for completion/drop-off analysis
