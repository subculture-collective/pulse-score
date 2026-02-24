# PulseScore

Customer health scoring platform for B2B SaaS companies. Connect Stripe, HubSpot, and Intercom to monitor customer health with automated scoring.

## Project Structure

```
pulse-score/
├── cmd/api/              # Application entry point
│   └── main.go
├── internal/             # Private application code
│   ├── config/           # Configuration management
│   ├── handler/          # HTTP handlers
│   ├── middleware/        # HTTP middleware
│   ├── model/            # Domain models
│   ├── repository/       # Database access layer
│   └── service/          # Business logic
├── pkg/                  # Reusable packages
├── migrations/           # Database migrations
├── scripts/
│   └── init-db/          # PostgreSQL initialization scripts
├── web/                  # React/TypeScript frontend (Vite + TailwindCSS v4)
│   └── src/
│       └── components/
├── docker-compose.dev.yml
├── Makefile
└── .env.example
```

## Tech Stack

- **Backend:** Go (net/http)
- **Frontend:** React 19, TypeScript, Vite, TailwindCSS v4
- **Database:** PostgreSQL 16
- **Deployment:** Docker, Nginx, VPS

## Prerequisites

- Go 1.24+
- Node.js 20+
- Docker & Docker Compose

## Getting Started

### 1. Clone and configure

```bash
cp .env.example .env
```

### 2. Start the database

```bash
docker compose -f docker-compose.dev.yml up -d
```

### 3. Run the API

```bash
make run
```

The API starts on http://localhost:8080. Health check: `GET /healthz`

### 4. Run the frontend

```bash
cd web
npm install
npm run dev
```

The frontend starts on http://localhost:5173.

## Development Commands

### Backend (Makefile)

| Command             | Description                   |
| ------------------- | ----------------------------- |
| `make build`        | Build the Go binary           |
| `make run`          | Build and run the API         |
| `make test`         | Run tests with race detection |
| `make lint`         | Run `go vet`                  |
| `make dev-db`       | Start development PostgreSQL  |
| `make dev-db-down`  | Stop development PostgreSQL   |
| `make migrate-up`   | Run database migrations up    |
| `make migrate-down` | Roll back database migrations |

### Frontend (web/)

| Command           | Description              |
| ----------------- | ------------------------ |
| `npm run dev`     | Start Vite dev server    |
| `npm run build`   | Production build         |
| `npm run lint`    | ESLint check             |
| `npm run format`  | Format with Prettier     |
| `npm run preview` | Preview production build |

## Billing & Subscription (Epic 12)

PulseScore now includes a dedicated Stripe billing domain (separate from Stripe customer-data integration):

- Plan catalog: `free`, `growth`, `scale` (`internal/billing/plans.go`)
- Protected billing APIs:
	- `GET /api/v1/billing/subscription`
	- `POST /api/v1/billing/checkout` (admin)
	- `POST /api/v1/billing/portal-session` (admin)
	- `POST /api/v1/billing/cancel` (admin)
- Public billing webhook:
	- `POST /api/v1/webhooks/stripe-billing`

### Required production billing env vars

- `STRIPE_BILLING_SECRET_KEY`
- `STRIPE_BILLING_PUBLISHABLE_KEY`
- `STRIPE_BILLING_WEBHOOK_SECRET`
- `STRIPE_BILLING_PRICE_GROWTH_MONTHLY`
- `STRIPE_BILLING_PRICE_GROWTH_ANNUAL`
- `STRIPE_BILLING_PRICE_SCALE_MONTHLY`
- `STRIPE_BILLING_PRICE_SCALE_ANNUAL`

Optional:

- `STRIPE_BILLING_PORTAL_RETURN_URL` (defaults to `http://localhost:5173/settings/billing`)
