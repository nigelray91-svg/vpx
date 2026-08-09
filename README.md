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
| Frontend | Next.js 15 (App Router), TypeScript, Tailwind |
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
single `X-API-Key` header, base `https://vaultproxies.net`, all six endpoints —

| Our use | Upstream |
|---|---|
| Catalog sync | `GET /api/reseller/categories` |
| Provision order | `POST /api/reseller/order` (`category_key`, `units`, `time_unit`) |
| Live usage | `GET /api/reseller/services` |
| Health / wallet | `GET /api/reseller/balance` |
| Geo targets | `GET /api/reseller/proxy/generator/locations` |
| Proxy lines | `POST /api/reseller/proxy/generations/create` |

Orderable categories are `resi_pergb`, `resi_unlim`, `dc_unlim`, `ipv6_pergb`
(pricing_type `pergb`/`unlim`; units GB/HOUR/DAY).

**Gateways are not hardcoded.** The order endpoint returns the proxy
username/password but no host:port; the generator returns the authoritative
`hostname`/`port` for the plan, so provisioning calls it straight after ordering.
`VAULT_GATEWAY_*` remains only as a fallback if the generator is unavailable, so
a paid order still yields usable credentials. Sticky sessions and geo targeting
are likewise handled by the generator — usernames are never assembled by hand.

### Whitelabel proxy DNS

The generator returns the upstream's own gateway hostname (e.g.
`resi-gb.vaultproxies.com`), and that hostname ends up in every proxy line your
customers use — so shipping it unmodified advertises your supplier.

Set `PROXY_BRAND_DOMAIN=proxies.yourbrand.com` and the backend rewrites each
gateway to your domain, preserving the label so the endpoints stay distinct:

```
resi-gb.vaultproxies.com  ->  resi-gb.proxies.yourbrand.com
eu-isp.vaultproxies.com   ->  eu-isp.proxies.yourbrand.com
```

Create one **CNAME per gateway label** pointing at the upstream host (the full
list is in `.env.example`). These are plaintext HTTP/SOCKS5 gateways with no TLS
to the proxy itself, so a CNAME resolves to the same endpoint and authentication
is unaffected. Keep the records **DNS-only** — proxying them through a CDN would
break the proxy protocol. `PROXY_HOSTNAME_MAP` overrides individual hosts, and
`PROXY_BRAND_STRICT=true` makes the backend refuse to emit an unbranded hostname
rather than leak one. Rewriting happens at the single point every generated line
passes through, and `TestGenerateNeverLeaksUpstreamHostname` asserts nothing
customer-facing contains the upstream domain.

Two upstream behaviours worth knowing, both by design:

- **Generating is free.** It mints credentials rather than selling capacity, so
  `remaining_gb` does not move and customers can regenerate as often as they
  like. Bandwidth is consumed by traffic only.
- **Rotating mode returns identical lines.** With no session token there is
  nothing to differentiate them, and a fresh exit IP is issued per request.
  `count` is only meaningful for `sticky`, where each line carries its own
  session. The API returns an explanatory `note` rather than letting this look
  like a bug.

All wire structs live in `backend/internal/vaultproxies/live.go` and are covered
by `live_test.go`, which asserts the mapping against the exact JSON from the docs
(including generator gateway selection and the fallback path).
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

## Deploying

See [`docs/DEPLOY.md`](docs/DEPLOY.md) for the step-by-step server runbook.

## DNS

See [`docs/DNS.md`](docs/DNS.md) for the exact records to create — A records
pointing the panel/API at your server, and CNAMEs pointing the branded proxy
gateways at the upstream. Proxy traffic never transits your server.

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
