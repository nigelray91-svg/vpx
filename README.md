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

Implemented against the real [VaultProxies Reseller API](https://vaultproxies.net/docs):
single `X-API-Key` header, base `https://vaultproxies.net`, four endpoints —

| Our use | Upstream |
|---|---|
| Catalog sync | `GET /api/reseller/categories` |
| Provision order | `POST /api/reseller/order` (`category_key`, `units`, `time_unit`) |
| Live usage | `GET /api/reseller/services` |
| Health / wallet | `GET /api/reseller/balance` |

Categories are `resi_pergb`, `resi_unlim`, `dc_unlim`, `ipv6_pergb` (pricing_type
`pergb`/`unlim`; units GB/HOUR/DAY). The order endpoint returns the proxy
**username/password** but not the gateway host:port, so set `VAULT_GATEWAY_*`
from your dashboard. Sticky sessions are controlled via the username
(`-session-XXXX-time-YYYY`, up to 12h), not at order time.

All wire structs live in `backend/internal/vaultproxies/live.go` and are covered
by `live_test.go`, which asserts the mapping against the exact JSON from the docs.
Run `VAULTPROXIES_MODE=mock` to exercise everything without the upstream; set
`live` + `VAULTPROXIES_API_KEY` on a network that can reach vaultproxies.net.

## Pricing / margin — synced from the upstream

The catalog is **not hardcoded**. On startup (and via `POST /api/v1/admin/catalog/sync`),
the backend pulls the VaultProxies reseller product catalog and upserts it into the
`plans` table, computing **`retail = round(wholesale × RESELLER_MARKUP)`**. Set
`RESELLER_MARKUP` (e.g. `1.40` = +40%) and your margin is applied to every plan
automatically; products retired upstream are deactivated. The seed migration only
provides a fallback catalog for first boot / when the upstream is unreachable
(e.g. `VAULTPROXIES_MODE=mock`).

> Because this build environment blocks `vaultproxies.net`, the sync runs against
> the **mock** catalog here. Set `VAULTPROXIES_MODE=live` + a valid
> `VAULTPROXIES_API_KEY` on a network that can reach the upstream, confirm the
> catalog endpoint/fields in `backend/internal/vaultproxies/live.go` against
> `/docs`, and the real products + prices populate automatically.

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
