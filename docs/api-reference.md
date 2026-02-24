# PulseScore API Reference

This reference documents the public REST API surface under `/api/v1`.

- Base URL (local): `http://localhost:8080/api/v1`
- Content type: `application/json`
- Auth scheme: `Authorization: Bearer <access_token>` for protected endpoints
- Source of truth: `docs/openapi.yaml` + implemented routes in `cmd/api/main.go`

## Authentication

### JWT flow

1. `POST /auth/register` or `POST /auth/login` returns:
   - `access_token` (short-lived)
   - `refresh_token` (long-lived)
2. Include the access token in protected requests:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

3. When access token expires, call `POST /auth/refresh` with the refresh token to rotate both tokens.

### Token refresh example

**Request**

```json
{
  "refresh_token": "rt_123..."
}
```

**Response (200)**

```json
{
  "access_token": "eyJ...new",
  "refresh_token": "rt_456...new",
  "expires_in": 900,
  "user": {
    "id": "3d9d3d07-8ca4-4d84-bf1f-3fd95b874be6",
    "email": "owner@acme.com",
    "role": "admin"
  }
}
```

## Pagination

PulseScore currently uses offset/page-style pagination patterns:

1. **Page-based** (customers/events): `page`, `per_page`
2. **Offset-based** (alerts/notifications): `limit`, `offset`

### Page-based example

`GET /customers?page=2&per_page=20&sort=mrr&order=desc`

### Offset-based example

`GET /alerts/history?status=triggered&limit=25&offset=50`

## Rate limits

### API request limit

Global per-IP request limit is enforced via middleware:

- Default: **100 requests/minute** (`RATE_LIMIT_RPM`)

### Per-tier product limits

| Plan | Customer limit | Integration limit |
| --- | ---: | ---: |
| `free` | 10 | 1 |
| `growth` | 250 | 3 |
| `scale` | unlimited | unlimited |

## Error responses

Common error payload:

```json
{
  "error": "message"
}
```

Common HTTP statuses:

- `400` invalid request body, invalid query/path parameter, missing webhook signature
- `401` unauthorized (missing/invalid JWT)
- `403` forbidden (insufficient role, e.g. admin-only endpoints)
- `404` resource not found
- `409` conflict (e.g. registration/invitation conflict)
- `422` validation error
- `429` rate limited
- `500` internal server error

---

## Health

### GET `/healthz`
- **Auth required:** No
- **Description:** Liveness probe.

**Response (200)**

```json
{ "status": "ok" }
```

### GET `/readyz`
- **Auth required:** No
- **Description:** Readiness probe (includes DB health).

**Response (200/503)**

```json
{ "status": "ok", "db": "connected" }
```

---

## Auth

### POST `/auth/register`
- **Auth required:** No
- **Description:** Register user + initial organization.

**Request**

```json
{
  "name": "Acme Inc",
  "email": "owner@acme.com",
  "password": "StrongPassword123!"
}
```

**Response (201)**

```json
{
  "access_token": "eyJ...",
  "refresh_token": "rt_123...",
  "user": {
    "id": "3d9d3d07-8ca4-4d84-bf1f-3fd95b874be6",
    "email": "owner@acme.com",
    "role": "admin"
  },
  "organization": {
    "id": "1f0d2f47-5f0b-4e61-a929-b81f16431ba4",
    "name": "Acme Inc"
  }
}
```

### POST `/auth/login`
- **Auth required:** No
- **Description:** Login with email/password.

**Request**

```json
{
  "email": "owner@acme.com",
  "password": "StrongPassword123!"
}
```

**Response (200)**

```json
{
  "access_token": "eyJ...",
  "refresh_token": "rt_123...",
  "user": {
    "id": "3d9d3d07-8ca4-4d84-bf1f-3fd95b874be6",
    "email": "owner@acme.com",
    "role": "admin"
  }
}
```

### POST `/auth/refresh`
- **Auth required:** No
- **Description:** Rotate access/refresh tokens.

**Request**

```json
{ "refresh_token": "rt_123..." }
```

**Response (200)**

```json
{
  "access_token": "eyJ...new",
  "refresh_token": "rt_456...new"
}
```

### POST `/auth/password-reset/request`
- **Auth required:** No
- **Description:** Request reset email.

**Request**

```json
{ "email": "owner@acme.com" }
```

**Response (200)**

```json
{ "message": "If that email exists, a reset link has been sent." }
```

### POST `/auth/password-reset/complete`
- **Auth required:** No
- **Description:** Complete password reset.

**Request**

```json
{
  "token": "reset_token_value",
  "new_password": "N3wStrongPassword!"
}
```

**Response (200)**

```json
{ "message": "Password reset successful" }
```

---

## Customers

### GET `/customers`
- **Auth required:** Yes (JWT)
- **Description:** List customers with filters.
- **Query params:** `page`, `per_page`, `sort`, `order`, `risk`, `search`, `source`

**Response (200)**

```json
{
  "customers": [
    {
      "id": "3f4f5b8d-1ad4-4748-bf40-55b8c8b8e29d",
      "name": "Acme Corp",
      "source": "stripe",
      "mrr": 12500,
      "health_score": 78,
      "risk_level": "medium"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 1
  }
}
```

### GET `/customers/{id}`
- **Auth required:** Yes (JWT)
- **Description:** Retrieve one customer detail.

**Response (200)**

```json
{
  "id": "3f4f5b8d-1ad4-4748-bf40-55b8c8b8e29d",
  "name": "Acme Corp",
  "mrr": 12500,
  "health_score": 78,
  "risk_level": "medium",
  "last_activity_at": "2026-02-20T10:15:00Z"
}
```

### GET `/customers/{id}/events`
- **Auth required:** Yes (JWT)
- **Description:** Customer event timeline.
- **Query params:** `page`, `per_page`, `type`, `from`, `to`

**Response (200)**

```json
{
  "events": [
    {
      "id": "4b4355f5-63f3-4a88-bf31-c4f1c99b4f6b",
      "type": "payment_failed",
      "occurred_at": "2026-02-21T08:00:00Z",
      "metadata": { "invoice_id": "in_123" }
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 1
  }
}
```

---

## Health Scores

### GET `/dashboard/summary`
- **Auth required:** Yes (JWT)
- **Description:** Dashboard KPI summary.

**Response (200)**

```json
{
  "total_customers": 42,
  "average_health_score": 74,
  "at_risk_customers": 6,
  "healthy_customers": 24
}
```

### GET `/dashboard/score-distribution`
- **Auth required:** Yes (JWT)
- **Description:** High/medium/low distribution.

**Response (200)**

```json
{
  "high": 24,
  "medium": 12,
  "low": 6
}
```

### GET `/scoring/risk-distribution`
- **Auth required:** Yes (JWT)
- **Description:** Risk bucket distribution from scoring engine.

**Response (200)**

```json
{
  "low": 24,
  "medium": 12,
  "high": 6
}
```

### GET `/scoring/histogram`
- **Auth required:** Yes (JWT)
- **Description:** Histogram of score buckets.

**Response (200)**

```json
{
  "bins": [
    { "range": "0-20", "count": 1 },
    { "range": "21-40", "count": 3 },
    { "range": "41-60", "count": 9 },
    { "range": "61-80", "count": 17 },
    { "range": "81-100", "count": 12 }
  ]
}
```

### POST `/scoring/customers/{id}/recalculate`
- **Auth required:** Yes (JWT)
- **Description:** Trigger immediate score recalculation for one customer.

**Request**

```json
{}
```

**Response (200)**

```json
{ "status": "recalculation_scheduled" }
```

### GET `/scoring/config`
- **Auth required:** Yes (JWT + admin)
- **Description:** Read scoring configuration.

**Response (200)**

```json
{
  "weights": {
    "payment_recency": 0.25,
    "mrr_trend": 0.25,
    "failed_payments": 0.2,
    "support_tickets": 0.15,
    "engagement": 0.15
  },
  "change_delta": 10
}
```

### PUT `/scoring/config`
- **Auth required:** Yes (JWT + admin)
- **Description:** Update scoring config.

**Request**

```json
{
  "weights": {
    "payment_recency": 0.3,
    "mrr_trend": 0.2,
    "failed_payments": 0.2,
    "support_tickets": 0.15,
    "engagement": 0.15
  },
  "change_delta": 8
}
```

**Response (200)**

```json
{
  "status": "updated"
}
```

---

## Integrations

### Generic provider endpoints

### GET `/integrations`
- **Auth required:** Yes (JWT)
- **Description:** List connected integrations.

**Response (200)**

```json
{
  "integrations": [
    { "provider": "stripe", "status": "connected" },
    { "provider": "hubspot", "status": "disconnected" }
  ]
}
```

### GET `/integrations/{provider}/status`
- **Auth required:** Yes (JWT + admin)
- **Description:** Get status for provider (e.g. `stripe`, `hubspot`, `intercom`).

**Response (200)**

```json
{
  "provider": "stripe",
  "status": "connected",
  "last_sync_at": "2026-02-24T20:05:00Z"
}
```

### POST `/integrations/{provider}/sync`
- **Auth required:** Yes (JWT + admin)
- **Description:** Trigger sync.

**Request**

```json
{}
```

**Response (200)**

```json
{ "status": "sync_started" }
```

### DELETE `/integrations/{provider}`
- **Auth required:** Yes (JWT + admin)
- **Description:** Disconnect provider.

**Response (200)**

```json
{ "status": "disconnected" }
```

### Stripe-specific routes

- `GET /integrations/stripe/connect` (admin; starts OAuth)
- `GET /integrations/stripe/callback` (admin; OAuth callback)
- `GET /integrations/stripe/status` (admin)
- `DELETE /integrations/stripe` (admin)
- `POST /integrations/stripe/sync` (admin)

### HubSpot-specific routes

- `GET /integrations/hubspot/connect` (admin; starts OAuth)
- `GET /integrations/hubspot/callback` (admin; OAuth callback)
- `GET /integrations/hubspot/status` (admin)
- `DELETE /integrations/hubspot` (admin)
- `POST /integrations/hubspot/sync` (admin)

### Intercom-specific routes

- `GET /integrations/intercom/connect` (admin; starts OAuth)
- `GET /integrations/intercom/callback` (admin; OAuth callback)
- `GET /integrations/intercom/status` (admin)
- `DELETE /integrations/intercom` (admin)
- `POST /integrations/intercom/sync` (admin)

### Integration webhooks (public; signature-verified)

- `POST /webhooks/stripe`
- `POST /webhooks/hubspot`
- `POST /webhooks/intercom`

**Webhook response (200)**

```json
{ "status": "ok" }
```

---

## Alerts

### GET `/alerts/rules`
- **Auth required:** Yes (JWT + admin)
- **Description:** List alert rules.

**Response (200)**

```json
{
  "rules": [
    {
      "id": "a1885638-870c-4070-ac53-f8de157e7a93",
      "name": "MRR drop > 20%",
      "enabled": true,
      "severity": "high"
    }
  ]
}
```

### POST `/alerts/rules`
- **Auth required:** Yes (JWT + admin)
- **Description:** Create alert rule.

**Request**

```json
{
  "name": "MRR drop > 20%",
  "description": "Detects sudden revenue drop",
  "enabled": true,
  "severity": "high",
  "conditions": [
    { "field": "mrr_change_pct", "operator": "lte", "value": -20 }
  ]
}
```

**Response (201/200)**

```json
{
  "id": "a1885638-870c-4070-ac53-f8de157e7a93",
  "name": "MRR drop > 20%",
  "enabled": true
}
```

### GET `/alerts/rules/{id}`
- **Auth required:** Yes (JWT + admin)
- **Description:** Get one alert rule.

**Response (200)**

```json
{
  "id": "a1885638-870c-4070-ac53-f8de157e7a93",
  "name": "MRR drop > 20%",
  "enabled": true,
  "severity": "high"
}
```

### PATCH `/alerts/rules/{id}`
- **Auth required:** Yes (JWT + admin)
- **Description:** Update rule fields.

**Request**

```json
{
  "enabled": false
}
```

**Response (200)**

```json
{
  "id": "a1885638-870c-4070-ac53-f8de157e7a93",
  "enabled": false
}
```

### DELETE `/alerts/rules/{id}`
- **Auth required:** Yes (JWT + admin)
- **Description:** Delete alert rule.

**Response (200)**

```json
{ "status": "deleted" }
```

### GET `/alerts/rules/{id}/history`
- **Auth required:** Yes (JWT + admin)
- **Description:** List history entries for a rule.
- **Query params:** `limit`, `offset`

**Response (200)**

```json
{
  "history": [
    {
      "id": "0d7d8a8c-6efe-491a-a737-737f2b7f74c9",
      "status": "triggered",
      "triggered_at": "2026-02-24T13:00:00Z"
    }
  ]
}
```

### GET `/alerts/history`
- **Auth required:** Yes (JWT)
- **Description:** List org-wide alert history.
- **Query params:** `status`, `limit`, `offset`

**Response (200)**

```json
{
  "history": [
    {
      "id": "0d7d8a8c-6efe-491a-a737-737f2b7f74c9",
      "rule_id": "a1885638-870c-4070-ac53-f8de157e7a93",
      "status": "triggered"
    }
  ],
  "total": 1,
  "limit": 25,
  "offset": 0
}
```

### GET `/alerts/stats`
- **Auth required:** Yes (JWT)
- **Description:** Aggregated counts by status.

**Response (200)**

```json
{
  "triggered": 12,
  "sent": 11,
  "dismissed": 4
}
```

---

## Billing

### GET `/billing/subscription`
- **Auth required:** Yes (JWT)
- **Description:** Current plan, status, usage, and features.

**Response (200)**

```json
{
  "tier": "growth",
  "status": "active",
  "billing_cycle": "monthly",
  "renewal_date": "2026-03-24T00:00:00Z",
  "cancel_at_period_end": false,
  "usage": {
    "customers": { "used": 42, "limit": 250 },
    "integrations": { "used": 2, "limit": 3 }
  },
  "features": {
    "playbooks": true,
    "ai_insights": false
  }
}
```

### POST `/billing/checkout`
- **Auth required:** Yes (JWT + admin)
- **Description:** Create Stripe Checkout session for paid plan.

**Request**

```json
{
  "tier": "growth",
  "cycle": "monthly"
}
```

(or explicit price)

```json
{
  "priceId": "price_123",
  "annual": false
}
```

**Response (200)**

```json
{
  "url": "https://checkout.stripe.com/c/pay/cs_test_..."
}
```

### POST `/billing/portal-session`
- **Auth required:** Yes (JWT + admin)
- **Description:** Create Stripe customer portal session.

**Request**

```json
{}
```

**Response (200)**

```json
{
  "url": "https://billing.stripe.com/p/session/test_..."
}
```

### POST `/billing/cancel`
- **Auth required:** Yes (JWT + admin)
- **Description:** Schedule cancellation at period end.

**Request**

```json
{}
```

**Response (200)**

```json
{
  "status": "cancel_at_period_end"
}
```

### POST `/webhooks/stripe-billing`
- **Auth required:** No (Stripe signature required)
- **Required header:** `Stripe-Signature`
- **Description:** Receives Stripe billing lifecycle events.

**Response (200)**

```json
{ "status": "ok" }
```

---

## Additional implemented public endpoints

The following routes are also public API endpoints in the current implementation:

- Organizations: `POST /organizations`, `GET /organizations/current`, `PATCH /organizations/current`
- Users: `GET /users/me`, `PATCH /users/me`
- Members: `GET /members`, `PATCH /members/{id}/role`, `DELETE /members/{id}`
- Invitations: `POST /invitations/accept`, `GET /invitations`, `POST /invitations`, `DELETE /invitations/{id}`
- Notifications: `GET /notifications/preferences`, `PATCH /notifications/preferences`, `GET /notifications`, `GET /notifications/unread-count`, `POST /notifications/{id}/read`, `POST /notifications/read-all`
- Onboarding: `GET /onboarding/status`, `PATCH /onboarding/status`, `POST /onboarding/complete`, `POST /onboarding/reset`, `GET /onboarding/analytics`
- Other webhooks: `POST /webhooks/sendgrid`

For full schema-level details (components, field constraints, and shared response objects), see `docs/openapi.yaml`.
