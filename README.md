# VPX — Whitelabel Proxy Reseller

A self-hostable, production-oriented platform for **reselling VaultProxies**
(residential / ISP / datacenter / IPv6 / mobile) under your own brand. Customers
register, top up a wallet with **cards (Stripe)** or **crypto (NOWPayments)**, and
buy proxy plans that are provisioned through the VaultProxies wholesale API. You
keep the margin.

```
┌────────────┐      ┌──────────────┐      ┌────────────────────┐
│  Next.js   │──────│   Go API     │──────│  VaultProxies API  │
│  frontend  │ HTTPS│ (chi/pgx)    │  REST│  (wholesale)       │
└────────────┘      └──────┬───────┘      └────────────────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
         ┌────▼───┐   ┌────▼───┐   ┌────▼─────────────┐
         │Postgres│   │ Redis  │   │ Stripe /         │
         │(users, │   │(cache, │   │ NOWPayments      │
         │ wallet)│   │ rate)  │   │ (top-ups)        │
         └────────┘   └────────┘   └──────────────────┘
```

## Stack

| Layer | Tech |
|---|---|
| Frontend | Next.js 14 (App Router), TypeScript, Tailwind |
| Backend | Go 1.24, chi, pgx v5, go-redis v9 |
| Database | PostgreSQL 16 |
| Cache / rate limiting | Redis 7 (Lua token bucket) |
| Auth | Email+password (bcrypt), JWT access + rotating refresh, Google OAuth, Cloudflare Turnstile |
| Payments | Stripe Checkout (cards), NOWPayments invoices (crypto) |
| Upstream | VaultProxies wholesale reseller API |
| Deploy | Docker Compose (+ optional Caddy TLS edge) |

## Quick start (local, mock upstream)

```bash
cp .env.example .env
# Edit .env: set a strong JWT_SECRET and POSTGRES_PASSWORD.
# Set VAULTPROXIES_MODE=mock to run without contacting the upstream.
# For local HTTP set COOKIE_SECURE=false and COOKIE_DOMAIN= (empty).

docker compose up -d --build
# Frontend  -> http://localhost:3000
# API       -> http://localhost:8080/healthz
```

The Go backend runs database migrations automatically on startup and seeds a
default plan catalog (`backend/migrations`).

### Make an admin

Set `ADMIN_EMAIL=you@example.com` in `.env` (the account is promoted to `admin`
on the next backend start), or run `make seed-admin ADMIN_EMAIL=you@example.com`.

## Production

1. Point `panel.example.com` and `api.example.com` at the host (same registrable
   domain so auth cookies work — see `deploy/Caddyfile`).
2. Fill in real secrets in `.env`:
   - `JWT_SECRET` (`openssl rand -base64 48`), `POSTGRES_PASSWORD`
   - `COOKIE_DOMAIN=.example.com`, `COOKIE_SECURE=true`, `APP_ENV=production`
   - `PUBLIC_BASE_URL=https://panel.example.com`, `API_BASE_URL=https://api.example.com`
   - Stripe keys + `STRIPE_WEBHOOK_SECRET`; NOWPayments `API_KEY` + `IPN_SECRET`
   - Google OAuth client id/secret + redirect `https://api.example.com/api/v1/auth/google/callback`
   - Turnstile site + secret keys
   - `VAULTPROXIES_API_KEY` and `VAULTPROXIES_MODE=live`
3. Configure provider webhooks to point at:
   - Stripe: `https://api.example.com/api/v1/webhooks/stripe` (event `checkout.session.completed`)
   - NOWPayments IPN: `https://api.example.com/api/v1/webhooks/nowpayments`
4. Start with the TLS edge:
   ```bash
   docker compose --profile edge up -d --build
   ```

## VaultProxies upstream integration

> **Note:** The exact `/docs` endpoint paths and field names could not be
> retrieved from the build environment (egress policy blocks `vaultproxies.net`).
> The client targets the documented reseller model (provision an order for a
> proxy type → receive endpoint credentials → query usage). **All paths and JSON
> field names are centralized in `backend/internal/vaultproxies/live.go`** (the
> `endpoints` struct and the request/response structs). Align them with the real
> `/docs` and nothing else in the app needs to change. Run with
> `VAULTPROXIES_MODE=mock` to exercise the full stack without the upstream.

## Pricing / margin

Plans live in the `plans` table (seeded in `backend/migrations/00002_seed_plans.sql`).
Each plan has a wholesale and a retail price per unit (GB / IP / port). Edit the
seed or the table to set your catalog and margin. `RESELLER_MARKUP` documents the
default multiplier used when designing the catalog.

## Security

See [`docs/SECURITY.md`](docs/SECURITY.md). Highlights: bcrypt password hashing,
HS256 JWT access tokens with rotating, hashed refresh tokens, httpOnly+Secure
SameSite cookies, double-submit CSRF, Redis token-bucket rate limiting, strict
CORS, signed & idempotent payment webhooks, atomic wallet ledger (no negative
balances), parameterized SQL throughout, hardened security headers, non-root
distroless container.

## API

See [`docs/API.md`](docs/API.md) for the full endpoint contract.

## Repo layout

```
backend/    Go API (cmd/server, internal/{api,auth,store,cache,payments,vaultproxies,...})
frontend/   Next.js app
deploy/      Caddyfile (TLS edge)
docs/        API.md, SECURITY.md
docker-compose.yml
```

## Disclaimer

Validate the VaultProxies endpoint mapping against the live `/docs`, run your own
security review, and complete provider (Stripe/NOWPayments/Google/Cloudflare)
production onboarding before taking real payments.
