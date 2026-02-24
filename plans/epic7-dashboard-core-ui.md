# Execution Plan: Epic 7 — Dashboard Core UI (#6)

## Overview

**Epic:** [#6 — Dashboard Core UI](https://github.com/subculture-collective/pulse-score/issues/6)
**Sub-issues:** #106–#122 (17 issues)
**Scope:** Build the complete React dashboard including app shell with sidebar navigation, dashboard overview with stat cards and charts, customer list with search/filter/pagination, customer detail pages with timeline and score history, settings pages, shared UI components (toasts, skeletons, error boundaries), and dark mode support.

## Current State

The following foundations are already in place:

### Frontend (web/)
- **React 19** with Vite 7, TypeScript 5.9, TailwindCSS 4, React Router v7
- **Auth context** (`src/contexts/AuthContext.tsx`) — login/logout/register state, JWT token management
- **Protected routes** (`src/components/ProtectedRoute.tsx`) — route guard using auth context
- **Base layout** (`src/components/BaseLayout.tsx`) — minimal shell with top header and nav links (Dashboard, Customers, Settings), no sidebar
- **API client** (`src/lib/api.ts`) — Axios instance with auth interceptor
- **Stripe integration** (`src/lib/stripe.ts`, `src/components/integrations/StripeConnectionCard.tsx`)
- **Auth pages** — `LoginPage.tsx`, `RegisterPage.tsx`
- **Settings pages** — `SettingsPage.tsx`, `StripeCallbackPage.tsx`
- **Routing** — BrowserRouter with `/`, `/auth/login`, `/auth/register`, `/settings`, `/settings/stripe/callback`, and `*` catch-all

### Backend API Endpoints Available
All data-fetching endpoints needed by the dashboard have been implemented (Epic 6):
- `GET /api/v1/dashboard/summary` — stat cards data (total customers, at-risk, MRR, avg score)
- `GET /api/v1/dashboard/score-distribution` — histogram data for score distribution chart
- `GET /api/v1/customers` — paginated, sortable, filterable customer list (?page, ?per_page, ?sort, ?order, ?risk, ?search, ?source)
- `GET /api/v1/customers/{id}` — customer detail with health score, subscriptions, recent events
- `GET /api/v1/customers/{id}/events` — paginated customer event timeline
- `GET /api/v1/integrations` — list all integration connections with status
- `GET /api/v1/integrations/{provider}/status` — individual integration status
- `GET /api/v1/scoring/risk-distribution` — risk level breakdown
- `GET /api/v1/scoring/histogram` — score histogram
- `GET /api/v1/organizations/current` — organization info
- `GET /api/v1/users/me` — current user profile
- `GET /api/v1/members` — team member list
- `GET /api/v1/alerts/rules` — alert rules

### Missing from Current State (needed before starting)
- **Recharts** package — not yet installed, required for chart issues (#109, #110, #116)
- **No sidebar layout** — `BaseLayout.tsx` is a basic top-nav shell, needs to be replaced with a sidebar layout for #106
- **No `DashboardPage`** — current dashboard is an inline placeholder in `App.tsx`
- **No `CustomersPage` or `CustomerDetailPage`**
- **No dark mode** — no ThemeContext, no `dark:` class strategy configured

## Dependency Graph

```
                    ┌─────────────────────────────────────────────────────────┐
                    │                     PHASE 1                             │
                    │            Foundation & Shared Components                │
                    │                                                         │
                    │  #106 App Shell + Sidebar ◄── #107 Responsive Nav       │
                    │  #112 Health Score Badge                                │
                    │  #119 Toast/Notification System                         │
                    │  #120 Loading Skeletons + Empty States                  │
                    │  #121 Error Boundary + 404 Page                         │
                    │  #122 Dark Mode / Theme Toggle                          │
                    └───────────┬─────────────────────────────────────────────┘
                                │
                    ┌───────────▼─────────────────────────────────────────────┐
                    │                     PHASE 2                             │
                    │               Dashboard Overview                        │
                    │                                                         │
                    │  #108 Dashboard Overview + Stat Cards                   │
                    │  #109 Health Score Distribution Chart                   │
                    │  #110 MRR Trend Chart                                   │
                    └───────────┬─────────────────────────────────────────────┘
                                │
                    ┌───────────▼─────────────────────────────────────────────┐
                    │                     PHASE 3                             │
                    │              Customer List & Detail                      │
                    │                                                         │
                    │  #111 Customer List + Data Table ──► #113 Filters (URL) │
                    │  #114 Customer Detail Page ──► #115 Event Timeline      │
                    │                              ──► #116 Score History     │
                    └───────────┬─────────────────────────────────────────────┘
                                │
                    ┌───────────▼─────────────────────────────────────────────┐
                    │                     PHASE 4                             │
                    │           Settings & Integration Status                  │
                    │                                                         │
                    │  #117 Integration Status Indicators                     │
                    │  #118 Settings Page with Tabs                           │
                    └─────────────────────────────────────────────────────────┘
```

## Execution Phases

Issues are grouped into phases based on dependency chains. Issues within the same phase can be worked on in parallel unless noted otherwise.

---

### Phase 0 — Prerequisites (setup before any issues)

**Goal:** Install missing dependencies and configure tooling.

1. Install Recharts:
   ```bash
   cd web && npm install recharts
   ```
2. Verify TailwindCSS dark mode uses `class` strategy (Tailwind v4 uses CSS-based dark mode by default; confirm or configure `@custom-variant dark (&:where(.dark, .dark *))` in `src/index.css`)
3. Install any icon library if not present (e.g., `lucide-react` for sidebar/nav icons)

---

### Phase 1 — Foundation & Shared Components (can be parallelized)

All Phase 1 issues are either independent or have a single dependency (#107 depends on #106). Shared components in this phase are reused by every subsequent phase.

#### Phase 1a — App Shell (must be first)

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#106](https://github.com/subculture-collective/pulse-score/issues/106) | Build app shell and sidebar navigation | critical | `web/src/layouts/AppLayout.tsx` (create), `web/src/components/Sidebar.tsx` (create), `web/src/components/Header.tsx` (create), `web/src/components/BaseLayout.tsx` (modify or replace), `web/src/App.tsx` (modify) |

**Details:**

1. Create `web/src/layouts/AppLayout.tsx`:
   - Wrapper layout with sidebar on the left, header at top, and main content area
   - Accepts children via `<Outlet />` (React Router nested routes)
   - Provides layout context (sidebar collapsed state)

2. Create `web/src/components/Sidebar.tsx`:
   - Navigation links: Dashboard (`/`), Customers (`/customers`), Integrations (`/settings/integrations`), Settings (`/settings`)
   - Active nav item highlighted using `useLocation()` + path matching
   - Icons for each nav item (lucide-react or heroicons)
   - PulseScore logo/brand at top

3. Create `web/src/components/Header.tsx`:
   - Top bar within content area
   - Organization name from auth context
   - User dropdown menu: Profile, Switch Org (placeholder), Logout
   - Use `useAuth()` hook for user info and logout

4. Refactor `web/src/App.tsx`:
   - Replace inline `Dashboard` component with `<AppLayout>` using nested routes
   - Move from flat routes to nested route structure:
     ```tsx
     <Route element={<ProtectedRoute><AppLayout /></ProtectedRoute>}>
       <Route index element={<DashboardPage />} />
       <Route path="customers" element={<CustomersPage />} />
       <Route path="customers/:id" element={<CustomerDetailPage />} />
       <Route path="settings/*" element={<SettingsPage />} />
     </Route>
     ```
   - Keep auth routes outside the layout

5. Remove or deprecate `BaseLayout.tsx` (replaced by `AppLayout`)

**Tests:** Verify sidebar renders correct nav items, active state matches current route, user menu dropdown opens/closes, layout renders with outlet content.

---

#### Phase 1b — Responsive Navigation (depends on #106)

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#107](https://github.com/subculture-collective/pulse-score/issues/107) | Build responsive navigation with collapsible sidebar | high | `web/src/components/Sidebar.tsx` (modify), `web/src/layouts/AppLayout.tsx` (modify) |

**Details:**

1. Add collapse toggle button to `Sidebar.tsx`:
   - Button at bottom of sidebar or in header to collapse/expand
   - Collapsed state: show icons only, tooltip on hover
   - Persist collapse preference to `localStorage` key `sidebar-collapsed`

2. Mobile responsive behavior (< 768px):
   - Hide sidebar, show hamburger icon in `Header.tsx`
   - Hamburger opens slide-out drawer overlay
   - Close drawer on route change (`useEffect` watching `location`)
   - Close on outside click (overlay backdrop)

3. CSS transitions for collapse/expand animation (TailwindCSS `transition-all duration-300`)

**Tests:** Collapse toggle changes sidebar width, mobile menu opens/closes, localStorage persistence, drawer closes on navigation.

---

#### Phase 1c — Shared UI Components (independent, can be parallelized)

These four issues have no dependencies on each other and only require React project setup.

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#112](https://github.com/subculture-collective/pulse-score/issues/112) | Build health score badge/indicator component | high | `web/src/components/HealthScoreBadge.tsx` (create) |
| [#119](https://github.com/subculture-collective/pulse-score/issues/119) | Build toast/notification system | high | `web/src/components/Toast.tsx` (create), `web/src/contexts/ToastContext.tsx` (create), `web/src/App.tsx` (modify to wrap with ToastProvider) |
| [#120](https://github.com/subculture-collective/pulse-score/issues/120) | Build loading skeletons and empty state components | medium | `web/src/components/skeletons/TableSkeleton.tsx`, `CardSkeleton.tsx`, `ChartSkeleton.tsx`, `ProfileSkeleton.tsx` (create), `web/src/components/EmptyState.tsx` (create) |
| [#121](https://github.com/subculture-collective/pulse-score/issues/121) | Build error boundary and 404 page | medium | `web/src/components/ErrorBoundary.tsx` (create), `web/src/pages/NotFoundPage.tsx` (create), `web/src/App.tsx` (modify catch-all route) |

**#112 — Health Score Badge:**

1. Create `HealthScoreBadge.tsx` with props: `score: number`, `riskLevel: 'green' | 'yellow' | 'red'`, `size?: 'sm' | 'md' | 'lg'`
2. Visual: rounded badge/circle with score number, background color matching risk level
3. Colors: green (`#22c55e` / `bg-green-500`), yellow (`#eab308` / `bg-yellow-500`), red (`#ef4444` / `bg-red-500`)
4. Size variants: `sm` (inline, 6×6), `md` (card, 8×8), `lg` (detail page, 12×12)
5. Ensure accessible contrast: white text on colored background; verify WCAG AA
6. Support `dark:` variants

**#119 — Toast System:**

1. Create `ToastContext.tsx` with `ToastProvider` and `useToast()` hook
2. Toast types: `success`, `error`, `warning`, `info` with matching colors (green/red/yellow/blue)
3. Position: fixed top-right corner (`fixed top-4 right-4 z-50`)
4. Auto-dismiss: 5s default for success/info, errors require manual dismiss
5. Max 3 visible toasts, queue additional
6. API: `toast.success('Message')`, `toast.error('Message')`, `toast.warning('Message')`, `toast.info('Message')`
7. Animation: slide-in from right, fade out on dismiss
8. Each toast: icon, message, optional action button ("Retry", "Undo"), X close button
9. Wrap `App` content in `<ToastProvider>`

**#120 — Loading Skeletons & Empty States:**

1. Create skeleton components in `web/src/components/skeletons/`:
   - `TableSkeleton.tsx` — rows of animated pulse bars matching table columns
   - `CardSkeleton.tsx` — stat card shaped skeleton with pulse
   - `ChartSkeleton.tsx` — chart-area shaped skeleton
   - `ProfileSkeleton.tsx` — avatar + text lines skeleton
2. All skeletons use TailwindCSS `animate-pulse` with `bg-gray-200 dark:bg-gray-700` bars
3. Create `EmptyState.tsx` — reusable component with props: `icon`, `title`, `description`, `actionLabel`, `onAction`
4. Variants can be created by passing different icons/text: no customers, no integrations, no events, etc.

**#121 — Error Boundary & 404 Page:**

1. Create `ErrorBoundary.tsx` as class component (React requirement for error boundaries)
   - Catches render errors via `componentDidCatch`
   - Displays friendly error message with "Try Again" button (calls `setState` to reset)
   - Logs error details to console
2. Create `NotFoundPage.tsx` with illustration/icon, "Page not found" message, and "Back to Dashboard" link
3. Update `App.tsx`: replace `<Navigate to="/" />` catch-all with `<NotFoundPage />`
4. Wrap major page routes in `<ErrorBoundary>` (or wrap at layout level)

---

#### Phase 1d — Dark Mode (depends on #106 for header toggle placement)

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#122](https://github.com/subculture-collective/pulse-score/issues/122) | Implement dark mode / theme toggle | medium | `web/src/contexts/ThemeContext.tsx` (create), `web/src/components/Header.tsx` (modify), `web/src/index.css` (modify), `web/index.html` (modify for no-flash script) |

**Details:**

1. Configure TailwindCSS v4 dark mode with class strategy:
   - Add `@custom-variant dark (&:where(.dark, .dark *));` to `index.css` (if not already configured)

2. Create `ThemeContext.tsx`:
   - Three modes: `'light'`, `'dark'`, `'system'`
   - `ThemeProvider` component that manages state
   - Reads initial value from `localStorage.getItem('theme')` or defaults to `'system'`
   - On `'system'`: use `window.matchMedia('(prefers-color-scheme: dark)')` and listen for changes
   - Apply/remove `dark` class on `<html>` element
   - Export `useTheme()` hook returning `{ theme, setTheme }`

3. Add toggle to `Header.tsx`: sun/moon icon button that cycles light → dark → system

4. Prevent flash of wrong theme:
   - Add inline `<script>` in `index.html` `<head>` that reads localStorage and applies `dark` class before React hydrates

5. Audit all existing components for `dark:` variants — add `dark:bg-gray-900`, `dark:text-gray-100`, etc. to layout, sidebar, header

**Tests:** Theme toggle switches modes, localStorage persists preference, system preference detection works, `dark` class applied/removed on `<html>`.

---

### Phase 2 — Dashboard Overview (depends on Phase 1a for layout)

These three issues build the dashboard page content. #108 is the page container; #109 and #110 are chart components embedded within it.

#### Phase 2a — Dashboard Page + Stat Cards

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#108](https://github.com/subculture-collective/pulse-score/issues/108) | Build dashboard overview page with summary stat cards | high | `web/src/pages/DashboardPage.tsx` (create), `web/src/components/StatCard.tsx` (create) |

**Details:**

1. Create `StatCard.tsx`:
   - Props: `title`, `value` (string/number), `trend` (percentage), `trendDirection` ('up'|'down'|'neutral'), `icon?`, `color?`
   - Arrow icon for trend direction, green for positive, red for negative
   - Format: MRR as currency, counts as integers, score as number

2. Create `DashboardPage.tsx`:
   - Fetch data from `GET /api/v1/dashboard/summary` using Axios
   - Render 4 stat cards in a responsive grid (`grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-6`)
   - Cards: Total Customers, At-Risk Customers, Total MRR ($), Average Health Score
   - Use `CardSkeleton` from #120 while loading
   - Error state with retry button, use `useToast()` for fetch errors
   - Auto-refresh every 5 minutes via `setInterval` + state refresh
   - Below stat cards: placeholder slots for charts (#109, #110)

3. Register route in `App.tsx` as index route within `AppLayout`

**Tests:** Stat cards render with correct data, loading skeleton shown while fetching, error state with retry, auto-refresh interval fires.

---

#### Phase 2b — Dashboard Charts (depend on #108 for page, can be parallel to each other)

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#109](https://github.com/subculture-collective/pulse-score/issues/109) | Build health score distribution chart | high | `web/src/components/charts/ScoreDistributionChart.tsx` (create), `web/src/pages/DashboardPage.tsx` (modify to embed chart) |
| [#110](https://github.com/subculture-collective/pulse-score/issues/110) | Build MRR trend chart | high | `web/src/components/charts/MRRTrendChart.tsx` (create), `web/src/pages/DashboardPage.tsx` (modify to embed chart) |

**#109 — Score Distribution Chart:**

1. Create `ScoreDistributionChart.tsx` using Recharts `BarChart`
2. Fetch data from `GET /api/v1/dashboard/score-distribution`
3. 10-point buckets (0-10, 11-20, ..., 91-100)
4. Color bars by risk level: red (0-39), yellow (40-69), green (70-100)
5. Legend with risk level counts and percentages
6. Tooltip on hover: bucket range and count
7. Responsive via `ResponsiveContainer`
8. Loading state: `ChartSkeleton`, empty state: `EmptyState` with "No score data yet"

**#110 — MRR Trend Chart:**

1. Create `MRRTrendChart.tsx` using Recharts `AreaChart`
2. Time range selector buttons: 7d, 30d, 90d, 1y (local state, re-fetches on change)
3. Fetch from `GET /api/v1/dashboard/mrr-trend?range=30d` (or equivalent endpoint — may need to check if this exists; fallback: derive from customer data)
4. X-axis: dates formatted (e.g., "Feb 1"), Y-axis: currency ($)
5. Gradient area fill under line (`defs > linearGradient`)
6. Tooltip: exact MRR value and date on hover
7. Responsive via `ResponsiveContainer`

> **Note:** Verify that an MRR trend endpoint exists on the backend. If not, this component will need a placeholder API hook or the backend endpoint must be added as a prerequisite.

---

### Phase 3 — Customer List & Detail (depends on Phase 1 for layout + shared components)

#### Phase 3a — Customer List Page

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#111](https://github.com/subculture-collective/pulse-score/issues/111) | Build customer list page with data table | critical | `web/src/pages/CustomersPage.tsx` (create), `web/src/components/CustomerTable.tsx` (create) |

**Details:**

1. Create `CustomersPage.tsx`:
   - Page wrapper with title "Customers"
   - Contains filter controls (placeholder for #113) + table + pagination
   - Fetch from `GET /api/v1/customers` with query parameters
   - Manage URL state: page, per_page, sort, order, risk, search, source

2. Create `CustomerTable.tsx`:
   - Columns: Name, Email, Company, MRR (formatted), Health Score (use `HealthScoreBadge` from #112), Risk Level, Last Seen (relative time)
   - Sortable column headers with visual sort indicator (arrow up/down)
   - Click sort header → toggle sort direction, update URL state
   - Row click → `navigate(/customers/${id})`
   - Use `TableSkeleton` from #120 while loading
   - Empty state when no results

3. Pagination component:
   - Shows page X of Y, prev/next buttons, optionally page numbers
   - Reads total/totalPages from API response

4. Register route: `/customers` in `App.tsx` nested under `AppLayout`

**Tests:** Table renders customer data, sorting toggles, pagination controls navigate pages, row click navigates to detail, empty/loading states.

---

#### Phase 3b — Customer Filters (depends on #111)

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#113](https://github.com/subculture-collective/pulse-score/issues/113) | Build customer search and filter controls | high | `web/src/components/CustomerFilters.tsx` (create), `web/src/pages/CustomersPage.tsx` (modify to integrate filters) |

**Details:**

1. Create `CustomerFilters.tsx`:
   - Search input with debounce (300ms) → updates URL `?search=`
   - Risk level filter: pill buttons for All / Green / Yellow / Red → updates `?risk=`
   - Source filter: dropdown for Stripe / HubSpot / Intercom → updates `?source=`
   - "Clear all" button to reset all filters

2. URL sync using `useSearchParams()` from React Router:
   - All filter state stored in URL search params
   - Component reads initial state from URL on mount (supports deep linking/bookmarking)
   - Changes to filters update URL → trigger re-fetch in `CustomersPage`

3. Integrate into `CustomersPage.tsx` above the table

**Tests:** Search updates URL with debounce, filter buttons update URL, direct URL access with filters loads correct state, clear all resets URL.

---

#### Phase 3c — Customer Detail Page (depends on #112 for badge, can start after Phase 1)

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#114](https://github.com/subculture-collective/pulse-score/issues/114) | Build customer detail page layout | high | `web/src/pages/CustomerDetailPage.tsx` (create) |

**Details:**

1. Create `CustomerDetailPage.tsx`:
   - Fetch customer data from `GET /api/v1/customers/{id}`
   - Extract `id` from URL params via `useParams()`
   - Header section: customer name, company, email, large `HealthScoreBadge` (lg), MRR display, Last Seen
   - Breadcrumb: Customers > Customer Name (with link back to `/customers`)
   - Content tabs/sections: Overview, Events, Subscriptions
   - Tab state via URL hash or nested routes
   - Overview tab: score factors breakdown (mini progress bars or meters)
   - Events tab: placeholder for #115
   - Subscriptions tab: list active Stripe subscriptions from detail API response
   - Handle 404: if customer not found, show error/not-found state
   - Loading state: `ProfileSkeleton`

2. Register route: `/customers/:id` in `App.tsx`

**Tests:** Page renders customer info, tab switching, breadcrumb navigation, 404/not-found handling, loading state.

---

#### Phase 3d — Customer Detail Sub-Components (depend on #114)

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#115](https://github.com/subculture-collective/pulse-score/issues/115) | Build customer event timeline component | high | `web/src/components/EventTimeline.tsx` (create), `web/src/pages/CustomerDetailPage.tsx` (modify to embed timeline) |
| [#116](https://github.com/subculture-collective/pulse-score/issues/116) | Build customer health score history chart | medium | `web/src/components/charts/ScoreHistoryChart.tsx` (create), `web/src/pages/CustomerDetailPage.tsx` (modify to embed chart) |

**#115 — Event Timeline:**

1. Create `EventTimeline.tsx`:
   - Fetch from `GET /api/v1/customers/{id}/events` with pagination
   - Vertical timeline layout: left line with dots, content on right
   - Each event: type icon (colored by type), title, timestamp (relative), source badge
   - Event type → icon/color mapping:
     - `payment.success` → green check
     - `payment.failed` → red X
     - `subscription.created` → blue plus
     - `ticket.opened` → yellow flag
     - Default → gray circle
   - Click event row → expand to show full event data (JSON or formatted)
   - "Load more" button at bottom for pagination (offset-based)
   - Empty state: "No events recorded yet"

2. Embed in `CustomerDetailPage` Events tab

**#116 — Score History Chart:**

1. Create `ScoreHistoryChart.tsx` using Recharts `LineChart`:
   - Score line over time with data points
   - Risk level reference areas in background: green (70-100), yellow (40-69), red (0-39) using `ReferenceArea`
   - Tooltip: exact score and date on hover
   - Time range selector (optional, if data supports it)
   - Handle sparse data gracefully (few data points → larger dot markers)
   - Responsive via `ResponsiveContainer`

2. Embed in `CustomerDetailPage` Overview section (below score factors)

---

### Phase 4 — Settings & Integration Status (depends on Phase 1 for layout)

| Issue | Title | Priority | Files to Create/Modify |
|-------|-------|----------|------------------------|
| [#117](https://github.com/subculture-collective/pulse-score/issues/117) | Build integration status indicators | medium | `web/src/components/IntegrationStatusBadge.tsx` (create), `web/src/components/IntegrationCard.tsx` (create) |
| [#118](https://github.com/subculture-collective/pulse-score/issues/118) | Build settings page with tabs | high | `web/src/pages/settings/SettingsPage.tsx` (rewrite), sub-tab components (create) |

**#117 — Integration Status Indicators:**

1. Create `IntegrationStatusBadge.tsx`:
   - Props: `status: 'connected' | 'syncing' | 'error' | 'disconnected'`
   - Visual: colored dot + status text
   - Colors: connected (green), syncing (blue with pulse animation `animate-pulse`), error (red), disconnected (gray)

2. Create `IntegrationCard.tsx`:
   - Props: `provider`, `status`, `lastSyncAt`, `customerCount`
   - Provider icon/name, status badge, last sync relative time ("5 minutes ago"), customer count
   - Action buttons: "Sync Now", "Disconnect" (conditional on status)

**#118 — Settings Page with Tabs:**

1. Rewrite `SettingsPage.tsx` with tabbed layout:
   - Tabs: Organization, Profile, Integrations, Scoring, Billing, Team
   - URL-synced: `/settings/organization`, `/settings/profile`, etc. (nested routes)
   - Use React Router nested routes or search params for tab selection

2. Tab content components (create each as separate file):
   - `web/src/pages/settings/OrganizationTab.tsx` — org name, slug, plan info (fetch from `/organizations/current`)
   - `web/src/pages/settings/ProfileTab.tsx` — user name, email (fetch from `/users/me`, PATCH to update)
   - `web/src/pages/settings/IntegrationsTab.tsx` — grid of `IntegrationCard` components (fetch from `/integrations`), includes existing `StripeConnectionCard`
   - `web/src/pages/settings/ScoringTab.tsx` — weight/threshold config (linked to scoring config API)
   - `web/src/pages/settings/BillingTab.tsx` — subscription plan + usage info (placeholder for now)
   - `web/src/pages/settings/TeamTab.tsx` — member list with roles, invite button (fetch from `/members`)

3. Each tab lazily loaded (`React.lazy` or just separate components)
4. Responsive: horizontal tabs on desktop, dropdown or vertical stack on mobile

**Tests:** Tab switching works, URL reflects selected tab, direct URL navigates to correct tab, each tab loads its data.

---

## File Summary

### New Files (create)

| File | Issue | Description |
|------|-------|-------------|
| `web/src/layouts/AppLayout.tsx` | #106 | Main app shell with sidebar + header + outlet |
| `web/src/components/Sidebar.tsx` | #106, #107 | Sidebar navigation, collapsible + responsive |
| `web/src/components/Header.tsx` | #106, #122 | Top header bar with user menu + theme toggle |
| `web/src/pages/DashboardPage.tsx` | #108 | Dashboard overview page |
| `web/src/components/StatCard.tsx` | #108 | Summary stat card |
| `web/src/components/charts/ScoreDistributionChart.tsx` | #109 | Score histogram bar chart |
| `web/src/components/charts/MRRTrendChart.tsx` | #110 | MRR trend area chart |
| `web/src/pages/CustomersPage.tsx` | #111 | Customer list page |
| `web/src/components/CustomerTable.tsx` | #111 | Customer data table |
| `web/src/components/HealthScoreBadge.tsx` | #112 | Health score badge/indicator |
| `web/src/components/CustomerFilters.tsx` | #113 | Search + filter controls |
| `web/src/pages/CustomerDetailPage.tsx` | #114 | Customer detail page |
| `web/src/components/EventTimeline.tsx` | #115 | Customer event timeline |
| `web/src/components/charts/ScoreHistoryChart.tsx` | #116 | Score history line chart |
| `web/src/components/IntegrationStatusBadge.tsx` | #117 | Status dot indicator |
| `web/src/components/IntegrationCard.tsx` | #117 | Integration connection card |
| `web/src/pages/settings/OrganizationTab.tsx` | #118 | Settings org tab |
| `web/src/pages/settings/ProfileTab.tsx` | #118 | Settings profile tab |
| `web/src/pages/settings/IntegrationsTab.tsx` | #118 | Settings integrations tab |
| `web/src/pages/settings/ScoringTab.tsx` | #118 | Settings scoring tab |
| `web/src/pages/settings/BillingTab.tsx` | #118 | Settings billing tab |
| `web/src/pages/settings/TeamTab.tsx` | #118 | Settings team tab |
| `web/src/components/Toast.tsx` | #119 | Toast notification component |
| `web/src/contexts/ToastContext.tsx` | #119 | Toast context + provider |
| `web/src/components/skeletons/TableSkeleton.tsx` | #120 | Table loading skeleton |
| `web/src/components/skeletons/CardSkeleton.tsx` | #120 | Card loading skeleton |
| `web/src/components/skeletons/ChartSkeleton.tsx` | #120 | Chart loading skeleton |
| `web/src/components/skeletons/ProfileSkeleton.tsx` | #120 | Profile loading skeleton |
| `web/src/components/EmptyState.tsx` | #120 | Empty state component |
| `web/src/components/ErrorBoundary.tsx` | #121 | React error boundary |
| `web/src/pages/NotFoundPage.tsx` | #121 | 404 page |
| `web/src/contexts/ThemeContext.tsx` | #122 | Dark mode theme context |

### Modified Files

| File | Issues | Changes |
|------|--------|---------|
| `web/src/App.tsx` | #106, #119, #121, #122 | Nested route structure, ErrorBoundary wrapping, ToastProvider, ThemeProvider |
| `web/src/components/BaseLayout.tsx` | #106 | Deprecated/removed (replaced by AppLayout) |
| `web/src/pages/settings/SettingsPage.tsx` | #118 | Rewrite with tabbed layout + nested routes |
| `web/src/pages/DashboardPage.tsx` | #109, #110 | Add chart components |
| `web/src/pages/CustomerDetailPage.tsx` | #115, #116 | Embed timeline + score chart |
| `web/src/index.css` | #122 | Dark mode custom variant |
| `web/index.html` | #122 | Theme flash prevention script |
| `web/package.json` | Phase 0 | Add recharts, lucide-react |

---

## Execution Order Summary

| Order | Issue(s) | Title | Est. Complexity |
|-------|----------|-------|-----------------|
| 0 | — | Install recharts, lucide-react, configure dark mode CSS | Low |
| 1 | #106 | App shell + sidebar navigation | High |
| 2 | #112, #119, #120, #121 | Health badge, toast system, skeletons, error boundary (parallel) | Medium each |
| 3 | #107 | Responsive collapsible sidebar | Medium |
| 4 | #122 | Dark mode / theme toggle | Medium |
| 5 | #108 | Dashboard overview + stat cards | Medium |
| 6 | #109, #110 | Score distribution chart, MRR trend chart (parallel) | Medium each |
| 7 | #111 | Customer list page + data table | High |
| 8 | #113 | Customer search/filter controls (URL-synced) | Medium |
| 9 | #114 | Customer detail page layout | High |
| 10 | #115, #116 | Event timeline, score history chart (parallel) | Medium each |
| 11 | #117 | Integration status indicators | Low |
| 12 | #118 | Settings page with tabs | High |

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| MRR trend API endpoint may not exist | #110 chart has no data source | Check backend for `/dashboard/mrr-trend` or equivalent; if missing, create a stub endpoint or derive from existing data |
| Recharts bundle size | Increased JS payload | Use tree-shakeable imports (`import { BarChart, Bar } from 'recharts'`) |
| Dark mode audit across all components | Missed `dark:` variants look broken | Do a full dark mode pass after all components built; add dark variants during initial creation |
| Inconsistent loading/error patterns | Poor UX | Enforce use of shared skeletons (#120), toast (#119), and error boundary (#121) in all pages |
| Customer list performance with large datasets | Slow table rendering | Use pagination (already server-side), avoid rendering all rows, consider virtualization if > 100 rows visible |

---

## Acceptance Criteria (Epic-Level)

- [ ] Responsive layout works from 320px to 1920px
- [ ] Dashboard displays correct summary statistics and charts
- [ ] Customer list paginated, sortable, filterable with URL-synced state
- [ ] Customer detail shows full profile, score breakdown, timeline
- [ ] Settings page tabs load correct content
- [ ] Dark mode toggles and persists preference
- [ ] Loading states, empty states, and error boundaries all functional
