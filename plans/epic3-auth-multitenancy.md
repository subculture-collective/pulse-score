# Execution Plan: Epic 3 — Authentication & Multi-tenancy (#12)

## Overview

**Epic:** [#12 — Authentication & Multi-tenancy](https://github.com/onnwee/pulse-score/issues/12)
**Sub-issues:** #56–#70 (15 issues)
**Scope:** User registration/login with JWT tokens, multi-tenant data isolation, RBAC, team invitations, password reset, and React frontend auth pages.

## Current State

The following foundations are already in place:

- **Go API server** with Chi router, structured logging, CORS, rate limiting, security headers
- **Database migrations** for `organizations`, `users`, and `user_organizations` tables (migrations 001–003)
- **Config package** with env-var loading
- **Repository base** with pgxpool
- **React/Vite/TypeScript** project scaffolded under `web/`
- **Health check endpoints** (`/healthz`, `/readyz`)

## Dependency Graph

```
#56 Registration
 └──► #57 Login
       ├──► #58 JWT Middleware ──► #60 Multi-tenant ──► #65 RBAC ──► #62 Invitations ──► #63 Email (SendGrid)
       │                     │                                  │                   └──► #64 Accept Invitation
       │                     │                                  └──► #66 Password Reset (needs #63)
       │                     ├──► #61 Org Creation
       │                     └──► #67 User Profile
       └──► #59 Refresh Token
                                                              ┌──► #68 Login Page ────────┐
                                                              ├──► #69 Registration Page ──┼──► #70 Auth State Mgmt
                                                              └──► #59 (Refresh Token) ────┘
```

## Execution Phases

Issues are grouped into phases based on dependency chains. Issues within the same phase can be worked on in parallel.

---

### Phase 1 — User Registration Endpoint

| Issue                                                  | Title                                | Priority | Files to Create/Modify                                                                                                       |
| ------------------------------------------------------ | ------------------------------------ | -------- | ---------------------------------------------------------------------------------------------------------------------------- |
| [#56](https://github.com/onnwee/pulse-score/issues/56) | Implement user registration endpoint | critical | `internal/service/auth.go`, `internal/handler/auth.go`, `internal/repository/user.go`, `internal/repository/organization.go` |

**Details:**

1. Create `internal/repository/user.go` — `UserRepository` with methods: `Create(ctx, user)`, `GetByEmail(ctx, email)`, `EmailExists(ctx, email)`
2. Create `internal/repository/organization.go` — `OrganizationRepository` with methods: `Create(ctx, org)`, `SlugExists(ctx, slug)`, `AddMember(ctx, userID, orgID, role)`
3. Create `internal/service/auth.go` — `AuthService` with `Register(ctx, req)` method:
    - Validate email format, check uniqueness (case-insensitive via CITEXT)
    - Validate password strength (min 8 chars)
    - Hash password with bcrypt cost 12
    - In a transaction: create user → create org → create user_org (role: owner)
    - Generate JWT access + refresh token pair
4. Create `internal/handler/auth.go` — `AuthHandler` with `Register` handler for `POST /api/v1/auth/register`
5. Add JWT helper package `internal/auth/jwt.go` — token generation/validation (needed by registration response and all subsequent auth work)
6. Wire route in `cmd/api/main.go`
7. Add `golang.org/x/crypto` (bcrypt) and `github.com/golang-jwt/jwt/v5` to `go.mod`
8. Add JWT config fields (`JWT_SECRET`, `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`) to config package

**New config env vars:** `JWT_SECRET`, `JWT_ACCESS_TTL` (default 15m), `JWT_REFRESH_TTL` (default 7d)

**Tests:** `internal/handler/auth_test.go`, `internal/service/auth_test.go` — success, duplicate email (409), invalid email (422), weak password (422)

**Acceptance criteria:**

- [ ] POST /api/v1/auth/register creates user + org in transaction
- [ ] Password hashed with bcrypt (cost 12)
- [ ] Duplicate email returns 409 Conflict
- [ ] Invalid email format returns 422
- [ ] Weak password returns 422 with clear message
- [ ] Response includes JWT access + refresh tokens
- [ ] User automatically gets 'owner' role in new org

---

### Phase 2 — User Login Endpoint

| Issue                                                  | Title                         | Priority | Files to Create/Modify                                                                |
| ------------------------------------------------------ | ----------------------------- | -------- | ------------------------------------------------------------------------------------- |
| [#57](https://github.com/onnwee/pulse-score/issues/57) | Implement user login endpoint | critical | `internal/handler/auth.go`, `internal/service/auth.go`, `internal/repository/user.go` |

**Depends on:** Phase 1 (#56)

**Details:**

1. Add `Login(ctx, req)` method to `AuthService`:
    - Lookup user by email (case-insensitive)
    - Compare password with bcrypt
    - Generate JWT access token (15min) + refresh token (7d)
    - JWT payload: `{user_id, org_id, role, exp, iat}`
    - Include user's default org (first org) in token
    - Track last login timestamp
2. Add `Login` handler to `AuthHandler` for `POST /api/v1/auth/login`
3. Add `UpdateLastLogin(ctx, userID)` to `UserRepository`
4. Implement login-specific rate limiting: 5 attempts per email per 15 minutes
5. Wire route in `cmd/api/main.go`

**Security:**

- Wrong password returns 401 with generic message (no user enumeration)
- Non-existent email returns identical 401 message
- Rate limiting on email to prevent brute force

**Tests:** Login success, wrong password (401), non-existent user (401), rate limit scenarios

**Acceptance criteria:**

- [ ] POST /api/v1/auth/login authenticates correctly
- [ ] Wrong password returns 401 (generic message, no user enumeration)
- [ ] Non-existent email returns 401 (same message as wrong password)
- [ ] JWT tokens generated with correct claims and TTL
- [ ] Refresh token stored/tracked for revocation
- [ ] Rate limiting prevents brute force

---

### Phase 3 — JWT Middleware & Refresh Token (parallel)

| Issue                                                  | Title                                   | Priority | Files to Create/Modify                                                                         |
| ------------------------------------------------------ | --------------------------------------- | -------- | ---------------------------------------------------------------------------------------------- |
| [#58](https://github.com/onnwee/pulse-score/issues/58) | Implement JWT authentication middleware | critical | `internal/middleware/auth.go`, `internal/auth/context.go`                                      |
| [#59](https://github.com/onnwee/pulse-score/issues/59) | Implement refresh token endpoint        | critical | `internal/handler/auth.go`, `internal/service/auth.go`, `internal/repository/refresh_token.go` |

**Depends on:** Phase 2 (#57)

#### #58 — JWT Middleware

1. Create `internal/middleware/auth.go`:
    - Parse `Authorization: Bearer <token>` header
    - Validate signature, expiration, issuer using `internal/auth/jwt.go`
    - Extract claims: `user_id`, `org_id`, `role`
    - Set in context with typed context keys
2. Create `internal/auth/context.go`:
    - Typed context keys (not bare strings) to prevent collisions
    - Helper functions: `GetUserID(ctx)`, `GetOrgID(ctx)`, `GetRole(ctx)`
3. Return 401 for missing/invalid/expired tokens with specific error codes

**Tests:** Valid token, invalid token, expired token, missing token, context helper extraction

#### #59 — Refresh Token Endpoint

1. Create `internal/repository/refresh_token.go` — store/lookup/revoke refresh tokens
2. Create migration for `refresh_tokens` table: `id`, `user_id`, `token_hash`, `expires_at`, `revoked_at`, `created_at`
3. Add `POST /api/v1/auth/refresh` handler:
    - Validate refresh token (not expired, not revoked)
    - Issue new access token (15min) + new refresh token (7d)
    - Invalidate old refresh token (rotation)
4. Wire route in `cmd/api/main.go`

**New migration:** `000010_create_refresh_tokens.{up,down}.sql`

**Tests:** Valid refresh returns new pair, old token invalidated, expired token (401), revoked token (401)

**Acceptance criteria (#58):**

- [ ] Valid token passes through, user context available to handlers
- [ ] Missing/invalid/expired tokens return 401
- [ ] Context helpers extract claims correctly
- [ ] Typed context keys prevent collisions

**Acceptance criteria (#59):**

- [ ] Valid refresh token returns new token pair
- [ ] Old refresh token is invalidated after use
- [ ] Expired/revoked refresh token returns 401

---

### Phase 4 — Multi-tenant Isolation, Org Creation, User Profile (parallel)

| Issue                                                  | Title                                       | Priority | Files to Create/Modify                                                 |
| ------------------------------------------------------ | ------------------------------------------- | -------- | ---------------------------------------------------------------------- |
| [#60](https://github.com/onnwee/pulse-score/issues/60) | Implement multi-tenant isolation middleware | critical | `internal/middleware/tenant.go`                                        |
| [#61](https://github.com/onnwee/pulse-score/issues/61) | Implement organization creation flow        | high     | `internal/handler/organization.go`, `internal/service/organization.go` |
| [#67](https://github.com/onnwee/pulse-score/issues/67) | Implement user profile API                  | medium   | `internal/handler/user.go`, `internal/service/user.go`                 |

**Depends on:** Phase 3 (#58 JWT middleware)

#### #60 — Multi-tenant Middleware

1. Create `internal/middleware/tenant.go`:
    - Default: use `org_id` from JWT claims
    - Optional: `X-Organization-ID` header to switch orgs (for multi-org users)
    - If header provided: verify user is a member of that org via `UserOrganizationRepository`
    - Set `org_id` in context for all downstream handlers/repositories
    - Panic if handler tries to query without `org_id` (safety net)
2. Add `GetUserOrg(ctx, userID, orgID)` to repository layer

**Tests:** Default org from JWT, org switch via header, non-member org switch (403), missing org context error

#### #61 — Organization Creation

1. Create `internal/handler/organization.go` — `POST /api/v1/organizations`
2. Create `internal/service/organization.go`:
    - Auto-generate slug from name (lowercase, hyphenated, unique)
    - Slug collision handling: append number suffix
    - Create organization + user_organizations (role: owner) in transaction
3. Wire route behind JWT middleware

**Tests:** Create org, slug generation, collision handling

#### #67 — User Profile API

1. Create `internal/handler/user.go`:
    - `GET /api/v1/users/me` — return user with org memberships
    - `PATCH /api/v1/users/me` — update first_name, last_name, avatar_url
2. Create `internal/service/user.go` — profile logic
3. Wire routes behind JWT middleware

**Tests:** GET profile, PATCH update, validation errors (422), unauthenticated (401)

---

### Phase 5 — RBAC Middleware

| Issue                                                  | Title                     | Priority | Files to Create/Modify        |
| ------------------------------------------------------ | ------------------------- | -------- | ----------------------------- |
| [#65](https://github.com/onnwee/pulse-score/issues/65) | Implement RBAC middleware | critical | `internal/middleware/rbac.go` |

**Depends on:** Phase 4 (#60 multi-tenant middleware)

**Details:**

1. Create `internal/middleware/rbac.go`:
    - `RequireRole(roles ...string) func(http.Handler) http.Handler`
    - Check user's role in current org from context (set by tenant middleware)
    - Role hierarchy: owner > admin > member (owner has all access, admin has member access)
    - Return 403 with clear error message if insufficient permission
    - Usage: `r.With(RequireRole("admin")).Post("/invite", ...)`

**Tests:** Owner accesses all, admin accesses admin+member, member can't access admin routes, 403 messages

**Acceptance criteria:**

- [ ] Owner can access all endpoints
- [ ] Admin can access admin + member endpoints
- [ ] Member can only access member endpoints
- [ ] 403 returned with clear message for insufficient role
- [ ] Works with Chi middleware chain

---

### Phase 6 — Team Member Invitations

| Issue                                                  | Title                                     | Priority | Files to Create/Modify                                                                                                 |
| ------------------------------------------------------ | ----------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------- |
| [#62](https://github.com/onnwee/pulse-score/issues/62) | Implement team member invitation endpoint | high     | `internal/handler/invitation.go`, `internal/service/invitation.go`, `internal/repository/invitation.go`, new migration |

**Depends on:** Phase 5 (#65 RBAC middleware)

**Details:**

1. Create migration `000011_create_invitations.{up,down}.sql`:
    - `invitations` table: `id`, `org_id`, `email`, `role`, `token` (unique), `status` (pending/accepted/expired), `invited_by`, `expires_at`, `created_at`
2. Create `internal/repository/invitation.go` — CRUD for invitations
3. Create `internal/service/invitation.go`:
    - Generate 32-byte random token (URL-safe base64)
    - Check: can't invite existing members or duplicate pending invitations
    - 7-day expiry
4. Create `internal/handler/invitation.go`:
    - `POST /api/v1/organizations/:org_id/invitations` — create (owner/admin only)
    - `GET /api/v1/invitations` — list pending for org
    - `DELETE /api/v1/invitations/:id` — revoke
5. Wire routes behind JWT + tenant + RBAC middleware

**New migration:** `000011_create_invitations.{up,down}.sql`

**Tests:** Owner/admin can invite, member can't (403), existing member (409), duplicate pending (409), expiry, list, revoke

---

### Phase 7 — Invitation Email & Acceptance (parallel)

| Issue                                                  | Title                                           | Priority | Files to Create/Modify                                             |
| ------------------------------------------------------ | ----------------------------------------------- | -------- | ------------------------------------------------------------------ |
| [#63](https://github.com/onnwee/pulse-score/issues/63) | Implement invitation email sending via SendGrid | high     | `internal/service/email.go`, `internal/service/sendgrid.go`        |
| [#64](https://github.com/onnwee/pulse-score/issues/64) | Implement invitation acceptance endpoint        | high     | `internal/handler/invitation.go`, `internal/service/invitation.go` |

**Depends on:** Phase 6 (#62 invitation endpoint)

#### #63 — Invitation Email (SendGrid)

1. Create `internal/service/email.go` — `EmailService` interface with `SendInvitation(ctx, params)` method
2. Create `internal/service/sendgrid.go` — SendGrid implementation
    - Add `github.com/sendgrid/sendgrid-go` dependency
    - HTML email template with PulseScore branding, org name, inviter name, role, accept link
    - Accept link: `{FRONTEND_URL}/invitations/accept?token={token}`
    - From: `noreply@pulsescore.com` (or configured sender)
    - Error handling: log failure but don't fail invitation creation
    - Dev mode: log email to stdout instead of sending
3. Add config fields: `SENDGRID_API_KEY`, `SENDGRID_FROM_EMAIL`, `FRONTEND_URL`
4. Hook into invitation creation flow

**Tests:** Mock SendGrid, verify email content, dev mode logging

#### #64 — Invitation Acceptance

1. Add `POST /api/v1/invitations/accept` handler:
    - Request: `{token, email, password, first_name, last_name}` (password optional if user exists)
    - Validate token: exists, not expired, not already accepted
    - If user exists: link to org with invited role
    - If new user: create user account, then link to org
    - Mark invitation as accepted
    - Return JWT tokens for immediate login
    - Use transaction for atomicity
2. Wire route (no auth required — public endpoint)

**Tests:** New user accept, existing user accept, expired token (410), already accepted (409), invalid token (404)

---

### Phase 8 — Password Reset Flow

| Issue                                                  | Title                         | Priority | Files to Create/Modify                                                                                         |
| ------------------------------------------------------ | ----------------------------- | -------- | -------------------------------------------------------------------------------------------------------------- |
| [#66](https://github.com/onnwee/pulse-score/issues/66) | Implement password reset flow | high     | `internal/handler/auth.go`, `internal/service/auth.go`, `internal/repository/password_reset.go`, new migration |

**Depends on:** Phase 7 (#63 — SendGrid email service)

**Details:**

1. Create migration `000012_create_password_resets.{up,down}.sql`:
    - `password_resets` table: `id`, `user_id`, `token_hash`, `expires_at`, `used_at`, `created_at`
2. Create `internal/repository/password_reset.go`
3. Add to `AuthService`:
    - `RequestPasswordReset(ctx, email)` — always returns 200 (no email enumeration), generates 32-byte random token, stores hash in DB, sends plain token in email
    - `CompletePasswordReset(ctx, token, newPassword)` — validate token, update password, invalidate token, revoke all refresh tokens
4. Add to `AuthHandler`:
    - `POST /api/v1/auth/password-reset/request`
    - `POST /api/v1/auth/password-reset/complete`
5. Add password reset email template to `EmailService`
6. Rate limit: 3 requests per email per hour
7. Token expires in 1 hour

**New migration:** `000012_create_password_resets.{up,down}.sql`

**Tests:** Happy path, expired token, invalid token, all refresh tokens revoked on password change

---

### Phase 9 — Frontend Auth Pages (parallel)

| Issue                                                  | Title                    | Priority | Files to Create/Modify                                   |
| ------------------------------------------------------ | ------------------------ | -------- | -------------------------------------------------------- |
| [#68](https://github.com/onnwee/pulse-score/issues/68) | Create login page        | critical | `web/src/pages/auth/LoginPage.tsx`, `web/src/lib/api.ts` |
| [#69](https://github.com/onnwee/pulse-score/issues/69) | Create registration page | critical | `web/src/pages/auth/RegisterPage.tsx`                    |

**Depends on:** Phase 2 (#57 login API) and Phase 1 (#56 registration API) — can start as early as after Phase 2, but listed here for logical grouping with the frontend work.

#### #68 — Login Page

1. Create `web/src/pages/auth/LoginPage.tsx`:
    - Form with email + password fields
    - Client-side validation (email format, password required)
    - API call to `POST /api/v1/auth/login`
    - Store tokens in memory (not localStorage)
    - Error states: invalid credentials, rate limited, server error
    - Loading spinner during API call
    - Links to registration and forgot password
    - Responsive, centered card design
2. Create `web/src/lib/api.ts` — API client with base URL config
3. Set up React Router if not already configured

**Tests:** Form validation, submission states, error display

#### #69 — Registration Page

1. Create `web/src/pages/auth/RegisterPage.tsx`:
    - Fields: first_name, last_name, email, password, confirm_password, org_name
    - Validation: email format, password strength (min 8, complexity), passwords match, required fields
    - API call to `POST /api/v1/auth/register`
    - On success: store tokens, redirect to onboarding
    - Password strength indicator (optional)
    - Terms of service checkbox
    - Link to login page
    - Responsive layout
2. Install frontend dependencies if needed (react-router-dom, form library)

**Tests:** Form validation, submission, duplicate email error display

---

### Phase 10 — Auth State Management & Protected Routes

| Issue                                                  | Title                                                | Priority | Files to Create/Modify                                                      |
| ------------------------------------------------------ | ---------------------------------------------------- | -------- | --------------------------------------------------------------------------- |
| [#70](https://github.com/onnwee/pulse-score/issues/70) | Implement auth state management and protected routes | critical | `web/src/contexts/AuthContext.tsx`, `web/src/components/ProtectedRoute.tsx` |

**Depends on:** Phase 9 (#68, #69) and Phase 3 (#59 refresh token endpoint)

**Details:**

1. Create `web/src/contexts/AuthContext.tsx` — `AuthProvider` with:
    - User state, login/logout/refresh functions
    - Access token in memory, refresh token handling
    - Auto-refresh: schedule refresh 1 minute before access token expires
    - On app load: attempt silent refresh to restore session
2. Create `web/src/components/ProtectedRoute.tsx`:
    - Wraps routes requiring auth
    - Redirects unauthenticated users to login
3. Create API interceptor:
    - Attach `Authorization` header to all requests
    - Handle 401 response (trigger refresh or logout)
4. Create `useAuth()` hook
5. Wire into `App.tsx` with route definitions

**Tests:** Auth flow, token refresh, route protection, logout state clearing

---

## Execution Summary

| Phase | Issues        | Parallelizable | Area     | New Migrations                  |
| ----- | ------------- | -------------- | -------- | ------------------------------- |
| 1     | #56           | —              | Backend  | —                               |
| 2     | #57           | —              | Backend  | —                               |
| 3     | #58, #59      | Yes            | Backend  | `000010_create_refresh_tokens`  |
| 4     | #60, #61, #67 | Yes            | Backend  | —                               |
| 5     | #65           | —              | Backend  | —                               |
| 6     | #62           | —              | Backend  | `000011_create_invitations`     |
| 7     | #63, #64      | Yes            | Backend  | —                               |
| 8     | #66           | —              | Backend  | `000012_create_password_resets` |
| 9     | #68, #69      | Yes            | Frontend | —                               |
| 10    | #70           | —              | Frontend | —                               |

**Total:** 15 issues across 10 phases
**New migrations:** 3
**New Go dependencies:** `golang.org/x/crypto` (bcrypt), `github.com/golang-jwt/jwt/v5`, `github.com/sendgrid/sendgrid-go`
**New frontend dependencies:** `react-router-dom` (if not present)

## Critical Path

The longest dependency chain determines the minimum sequential phases:

```
#56 → #57 → #58 → #60 → #65 → #62 → #63 → #66
```

This is an 8-phase critical path through the backend. Frontend work (#68, #69, #70) can be overlapped with phases 3–8 since the API endpoints they need are available after phases 1–2.

**Recommended parallel tracks:**

| Backend Track                               | Frontend Track                  |
| ------------------------------------------- | ------------------------------- |
| Phase 1: #56 Registration                   | —                               |
| Phase 2: #57 Login                          | —                               |
| Phase 3: #58 JWT MW + #59 Refresh           | Phase 9a: #68 Login Page        |
| Phase 4: #60 Tenant + #61 Org + #67 Profile | Phase 9b: #69 Registration Page |
| Phase 5: #65 RBAC                           | —                               |
| Phase 6: #62 Invitations                    | —                               |
| Phase 7: #63 Email + #64 Accept             | Phase 10: #70 Auth State Mgmt   |
| Phase 8: #66 Password Reset                 | —                               |

## Files Created/Modified Summary

### New Files (Backend)

| File                                    | Phase | Purpose                                                       |
| --------------------------------------- | ----- | ------------------------------------------------------------- |
| `internal/auth/jwt.go`                  | 1     | JWT token generation & validation                             |
| `internal/auth/context.go`              | 3     | Typed context keys + helper functions                         |
| `internal/repository/user.go`           | 1     | User database operations                                      |
| `internal/repository/organization.go`   | 1     | Organization database operations                              |
| `internal/repository/refresh_token.go`  | 3     | Refresh token storage & revocation                            |
| `internal/repository/invitation.go`     | 6     | Invitation CRUD operations                                    |
| `internal/repository/password_reset.go` | 8     | Password reset token storage                                  |
| `internal/service/auth.go`              | 1     | Registration, login, token refresh, password reset logic      |
| `internal/service/organization.go`      | 4     | Organization creation + slug generation                       |
| `internal/service/user.go`              | 4     | User profile logic                                            |
| `internal/service/invitation.go`        | 6     | Invitation creation, validation, acceptance                   |
| `internal/service/email.go`             | 7     | EmailService interface                                        |
| `internal/service/sendgrid.go`          | 7     | SendGrid email implementation                                 |
| `internal/handler/auth.go`              | 1     | Auth HTTP handlers (register, login, refresh, password reset) |
| `internal/handler/organization.go`      | 4     | Organization HTTP handlers                                    |
| `internal/handler/user.go`              | 4     | User profile HTTP handlers                                    |
| `internal/handler/invitation.go`        | 6     | Invitation HTTP handlers                                      |
| `internal/middleware/auth.go`           | 3     | JWT validation middleware                                     |
| `internal/middleware/tenant.go`         | 4     | Multi-tenant isolation middleware                             |
| `internal/middleware/rbac.go`           | 5     | Role-based access control middleware                          |

### New Files (Frontend)

| File                                    | Phase | Purpose                            |
| --------------------------------------- | ----- | ---------------------------------- |
| `web/src/lib/api.ts`                    | 9     | API client with auth interceptor   |
| `web/src/pages/auth/LoginPage.tsx`      | 9     | Login form page                    |
| `web/src/pages/auth/RegisterPage.tsx`   | 9     | Registration form page             |
| `web/src/contexts/AuthContext.tsx`      | 10    | Auth state provider + useAuth hook |
| `web/src/components/ProtectedRoute.tsx` | 10    | Route guard component              |

### New Migrations

| File                                                     | Phase | Purpose                |
| -------------------------------------------------------- | ----- | ---------------------- |
| `migrations/000010_create_refresh_tokens.{up,down}.sql`  | 3     | Refresh token tracking |
| `migrations/000011_create_invitations.{up,down}.sql`     | 6     | Team invitations       |
| `migrations/000012_create_password_resets.{up,down}.sql` | 8     | Password reset tokens  |

### Modified Files

| File                        | Phases Modified | Changes                                   |
| --------------------------- | --------------- | ----------------------------------------- |
| `cmd/api/main.go`           | 1–8             | Route wiring, middleware chain setup      |
| `internal/config/config.go` | 1, 7            | JWT config fields, SendGrid config fields |
| `go.mod`                    | 1, 7            | New dependencies                          |
| `web/src/App.tsx`           | 9–10            | Route definitions, AuthProvider wrapping  |
| `web/package.json`          | 9               | react-router-dom dependency               |

## Verification Checklist

After each phase, verify:

- [ ] All new tests pass (`go test ./...` or `npm test`)
- [ ] Existing tests still pass (no regressions)
- [ ] Linting passes (`golangci-lint run` / `npm run lint`)
- [ ] Migrations apply cleanly (up and down)
- [ ] API endpoints testable via curl/httpie
- [ ] No hardcoded secrets — all sensitive values from env vars

## Final Integration Verification

After all phases complete, verify the epic's top-level acceptance criteria:

- [ ] Users can register, login, and receive JWT tokens
- [ ] Refresh token rotation works correctly
- [ ] Organization data is fully isolated between tenants
- [ ] RBAC enforced: members can't access admin routes
- [ ] Team invitations sent via email and accepted
- [ ] Password reset flow works end-to-end
- [ ] React auth pages functional with protected route guards
