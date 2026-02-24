Plan: PulseScore Development Roadmap via GitHub Issues
Create ~211 task issues organized under 23 epic issues and 1 master roadmap issue in subculture-collective/pulse-score. Each sub-issue is scoped for a coding agent with detailed descriptions, goals, and acceptance criteria. The master roadmap lists every issue in dependency-respecting chronological order (flat, not grouped by epic). All hosting targets your VPS with Docker + Nginx + SSL (no Vercel).

Structure
1 Roadmap issue — flat chronological list of all ~211 issues
23 Epic issues — each lists its child issues in the body
~211 Task issues — each has description, goals, technical details, acceptance criteria, dependencies
Labels: phase:mvp, phase:expansion, phase:defensibility, type:epic, type:task, area:backend, area:frontend, area:infra, area:docs, area:integration, priority:critical/high/medium/low
Milestones: Phase 1: MVP, Phase 2: Expansion, Phase 3: Defensibility
Issue Template (each sub-issue follows this)
Description — What to build and why, with technical context
Goals — Specific deliverables
Technical Details — Files to create, packages to use, patterns to follow
Acceptance Criteria — Testable checkboxes including "tests written and passing"
Dependencies — Blocked by #N links

PHASE 1: MVP (15 Epics, ~143 issues)
E1: Project Foundation & Infrastructure (14 issues)

Initialize Go module with standard layout (cmd/api/, internal/, pkg/, Makefile)
Set up PostgreSQL with Docker Compose for local dev
Initialize React/TypeScript project (Vite + TailwindCSS v4)
Set up Go HTTP server with Chi router + base middleware (logging, recovery, request ID)
Set up database migration framework (golang-migrate)
Configure environment variable management (config package, .env.example)
Set up GitHub Actions CI (Go lint/test/build, React lint/test/build)
Create multi-stage Dockerfile for Go API
Create Dockerfile for React frontend with Nginx (SPA-routing)
Create production Docker Compose (PostgreSQL + API + frontend)
Set up Nginx reverse proxy with SSL (Let's Encrypt/Certbot)
Create VPS deployment scripts (initial setup + deploy command)
Implement health check endpoints (/healthz, /readyz)
Configure CORS, security headers, and rate limiting middleware
E2: Database Schema & Migrations (10 issues)

Organizations table migration
Users + user_organizations join table migration
Integration_connections table migration (OAuth tokens, encrypted)
Customers table migration (tenant-scoped, external_id, MRR, metadata JSONB)
Customer_events table migration (event log, typed, timestamped)
Health_scores + health_score_history tables migration
Stripe-specific data tables (subscriptions, payments)
Alert_rules + alert_history tables migration
Database seed script (1 org, 3 users, 50 customers, events, scores)
Database connection pool + query helpers (pgxpool, repository pattern)
E3: Authentication & Multi-tenancy (15 issues)

User registration endpoint (bcrypt, email validation, creates org)
User login endpoint (JWT access + refresh tokens)
JWT authentication middleware
Refresh token endpoint
Multi-tenant isolation middleware (org_id scoping)
Organization creation flow (auto on registration)
Team member invitation endpoint (generate token, RBAC-gated)
Invite email sending via SendGrid
Invite acceptance endpoint (create/link user to org)
RBAC middleware (owner/admin/member)
Password reset flow (request + completion endpoints)
User profile API (GET/PATCH)
Login page (React)
Registration page (React)
Auth state management + protected routes (React context, token refresh)
E4: Stripe Data Integration (12 issues)

Stripe OAuth connect flow backend (initiate + callback)
Stripe connection UI in settings (React)
Stripe customer data sync service
Stripe subscription data sync service
Stripe payment/invoice data sync service
Stripe webhook handler (signature verification, event routing)
MRR calculation service (normalize annual → monthly)
Failed payment detection and tracking
Payment recency calculation per customer
Initial data sync orchestrator (full sync on connect)
Incremental sync scheduler (periodic delta updates)
Connection monitoring + error handling (retry, status updates)
E5: Health Scoring Engine (11 issues)

Scoring configuration model + migration (weights, thresholds, per-org)
Payment recency scoring factor (configurable decay curve)
MRR trend scoring factor (growth/decline over time windows)
Failed payment history scoring factor
Support ticket volume scoring factor (normalized)
Engagement/activity scoring factor (event frequency, graceful degradation)
Weighted score aggregation service (handles missing factors)
Score recalculation scheduler (batch + event-triggered)
Score change detection + history tracking
Customer risk categorization (green/yellow/red thresholds)
Score configuration admin API (GET/PUT with validation)
E6: REST API Layer (10 issues)

Customer list endpoint (paginated, sortable, filterable by risk/search)
Customer detail endpoint (profile + score factors + subscriptions)
Customer timeline/events endpoint (paginated, filterable)
Dashboard summary stats endpoint (customer count, at-risk, MRR, trends)
Health score distribution endpoint (histogram/risk breakdown)
Integration management endpoints (list/disconnect/sync-status)
Organization settings endpoints (GET/PATCH)
User management endpoints (list/update-role/remove members)
Alert rules CRUD endpoints
OpenAPI/Swagger documentation generation
E7: Dashboard Core UI (17 issues)

App shell + sidebar navigation
Responsive navigation (collapsible sidebar, mobile hamburger)
Dashboard overview page with summary stat cards
Health score distribution chart (Recharts)
MRR trend chart (line chart, time range selector)
Customer list page with data table (sort, filter, paginate)
Health score badge/indicator component (reusable, color-coded)
Customer search + filter controls (URL-synced)
Customer detail page layout (header, tabs/sections)
Customer event timeline component
Customer health score history chart
Integration status indicators
Settings page with tabs (Org, Profile, Integrations, Scoring, Billing, Team)
Toast/notification system
Loading skeletons + empty state components
Error boundary + 404 page
Dark mode / theme toggle
E8: HubSpot Integration (8 issues) — OAuth, contact/deal/company sync, data enrichment, webhooks, incremental sync

E9: Intercom Integration (7 issues) — OAuth, conversation/contact sync, ticket volume metrics, webhooks, incremental sync

E10: Email Alerts (8 issues) — SendGrid setup, email templates, alert rule evaluation engine, score drop triggers, alert UI, delivery tracking, preferences

E11: Onboarding Wizard (7 issues) — Multi-step wizard shell, welcome/org setup step, Stripe/HubSpot/Intercom connection steps, "generating scores" preview step, tracking + resume

E12: Billing & Subscription (9 issues) — Stripe billing products setup, checkout via Stripe Checkout, billing webhook handler, subscription tracking, feature gating middleware (Free: 10 customers/1 integration → Scale: unlimited), pricing page, checkout flow, subscription management, Customer Portal redirect

E13: Landing Page (6 issues) — Hero with CTA, features section, pricing comparison, social proof, footer, SEO meta/OG tags + sitemap/robots.txt

E14: Documentation (5 issues) — API reference, quickstart guide, Stripe setup guide, HubSpot/Intercom guides, scoring methodology docs

E15: Stripe App Marketplace (4 issues) — Requirements research, manifest packaging, app install flow, marketplace listing content

PHASE 2: EXPANSION (4 Epics, ~36 issues)
E16: Automated Playbooks (10 issues) — Data model, CRUD API, visual builder UI, trigger: score threshold, trigger: customer event, action: email customer, action: internal alert, action: tag customer, execution engine, execution history UI

E17: Team Collaboration (8 issues) — Account assignment API + UI, customer notes API + UI, team activity feed API + page, @mentions + notifications, account handoff workflow

E18: Additional Integrations (12 issues) — Generic connector interface, Zendesk (OAuth + sync + UI), Salesforce (OAuth + sync + UI), PostHog (API key + sync + UI), generic webhook receiver + config UI, integration health monitoring dashboard

E19: Advanced Pricing & Feature Gating (6 issues) — Growth/Scale tier definitions, granular feature flags system, tier-based gating, upgrade prompts, usage analytics tracking

PHASE 3: DEFENSIBILITY (4 Epics, ~28 issues)
E20: Anonymized Benchmarking (7 issues) — Data model + anonymization pipeline, aggregation service, benchmark calculation (percentiles), comparison dashboard, industry classification, privacy controls, insight notifications

E21: AI-Powered Insights (7 issues) — LLM integration service (GPT-4o-mini), prompt template design, per-customer analysis pipeline, insights UI on detail page, action recommendations, cost tracking + rate limiting, dashboard summary insights

E22: Predictive Churn Models (6 issues) — Feature engineering pipeline, training data preparation, churn model implementation, prediction scoring service, churn indicators in UI, churn forecast dashboard

E23: Integration Marketplace (8 issues) — Connector SDK design, registration/discovery API, marketplace browse UI, installation flow, developer docs + example, community submission, review pipeline, connector analytics

Chronological Order (Master Roadmap)
The roadmap issue lists all ~211 issues flat, ordered by dependency chains:

Order Batch Issues
1-14 Foundation Go setup → React setup (parallel) → PostgreSQL → Chi server → env config → migrations → health checks → CORS/security → DB pool
15-23 Schema All E2 table migrations in order → seed script
24-38 Auth Registration → login → JWT middleware → refresh → multi-tenant → org creation → RBAC → profile → password reset → invitations → frontend auth pages + state management
39-44 CI/CD & Deploy GitHub Actions → Dockerfiles → production compose → Nginx + SSL → VPS deploy scripts (parallel with auth)
45-56 Stripe Integration OAuth → connection UI → customer/subscription/payment sync → webhooks → MRR calc → failed payments → recency → sync orchestrator → scheduler → monitoring
57-67 Health Scoring Config model → 5 scoring factors → aggregation → risk categorization → scheduler → change detection → config API
68-77 API Layer All REST endpoints → OpenAPI docs
78-94 Dashboard UI Shell → nav → shared components → dashboard page → charts → customer list → detail → timeline → settings → dark mode
95-109 HubSpot + Intercom Both integration flows (parallel)
110-117 Email Alerts SendGrid → templates → evaluation → triggers → UI → tracking → preferences
118-124 Onboarding Wizard shell → steps → score generation → tracking
125-133 Billing Products → checkout → webhooks → tracking → gating → pricing page → management
134-139 Landing Page Hero → features → pricing → social proof → footer → SEO
140-144 Docs API ref → quickstart → integration guides → scoring docs
145-148 Stripe Marketplace Research → package → install flow → listing
149-184 Expansion Playbooks → Team Collaboration → Additional Integrations → Advanced Pricing
185-211 Defensibility Benchmarking → AI Insights → Predictive Models → Integration Marketplace
Decisions
VPS deployment (not Vercel) — Docker Compose + Nginx + Certbot on user's VPS
Tests baked in — Each sub-issue acceptance criteria includes "tests written and passing"
Tech stack: Go (Chi) + React/TypeScript (Vite/Tailwind) + PostgreSQL — per the plan doc
All 3 phases covered — MVP through defensibility, ~211 issues total
Issue scoping: Each sub-issue targets 1-4 hours of focused coding agent work with clear inputs/outputs
Further Considerations
Password reset + invite emails: The plan references SendGrid for alerts, but auth emails (password reset, invites) are needed before the alerts epic. Recommendation: Set up SendGrid basics in E3-8 (invite email) and reuse in E10. Already reflected in the plan.

Stripe Connect vs Stripe OAuth: The plan mentions "Stripe Connect" but the use case is reading customer data, not processing payments on behalf. Recommendation: Use Stripe OAuth (standard) for reading data, and a separate Stripe account (direct) for PulseScore's own billing. This is how it's structured in E4 (data) vs E12 (billing).

Frontend component library: No specific UI component library is mentioned. Recommendation: Use shadcn/ui (TailwindCSS-native, widely supported) or keep it custom with Tailwind. This can be decided in E1-3 (React project setup). Which do you prefer?
