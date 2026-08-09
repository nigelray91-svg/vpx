# VPX Backend API Contract (v1)

Base URL: `${API_BASE_URL}` (e.g. `https://api.example.com`). All paths are
prefixed with `/api/v1`. Requests/responses are JSON.

## Authentication model

- On `register` / `login` the server sets three cookies:
  - `vpx_access` — httpOnly access JWT (~15 min).
  - `vpx_refresh` — httpOnly refresh token, path `/api/v1/auth`.
  - `vpx_csrf` — **readable by JS**, double-submit CSRF token.
- The browser client must send `credentials: 'include'` on every request.
- For every **state-changing** request (`POST/PUT/PATCH/DELETE`) the client must
  send header `X-CSRF-Token: <value of vpx_csrf cookie>`.
- The JSON response of `login`/`register` also returns `access_token` for
  non-browser API clients (sent as `Authorization: Bearer <token>`); Bearer
  clients are exempt from CSRF.
- When an authenticated request returns `401`, call `POST /auth/refresh`
  (sends the refresh cookie) then retry once.

## Public

| Method | Path | Body | Notes |
|---|---|---|---|
| GET | `/healthz` | — | liveness |
| GET | `/readyz` | — | readiness (db + upstream) |
| GET | `/api/v1/config` | — | site name, turnstile site key, enabled providers, min topup |
| GET | `/api/v1/plans` | — | resale catalog (retail prices) |

`GET /api/v1/config` →
```json
{ "site_name":"...", "turnstile_site_key":"...", "turnstile_enabled":true,
  "google_enabled":true, "stripe_enabled":true, "crypto_enabled":true,
  "min_topup_cents":500 }
```

`GET /api/v1/plans` →
```json
{ "plans":[ { "id":"uuid","code":"resi-rotating","name":"Residential — Rotating",
  "proxy_type":"residential","unit":"gb","price_cents":300,"min_quantity":1 } ] }
```

## Auth

| Method | Path | Body |
|---|---|---|
| POST | `/api/v1/auth/register` | `{ email, password, full_name?, turnstile_token }` |
| POST | `/api/v1/auth/login` | `{ email, password, turnstile_token }` |
| POST | `/api/v1/auth/refresh` | — (CSRF) |
| POST | `/api/v1/auth/logout` | — (CSRF) |
| GET | `/api/v1/auth/google` | redirects to Google |
| GET | `/api/v1/auth/google/callback` | redirects back to `${PUBLIC_BASE_URL}/dashboard` |

`login`/`register` success →
```json
{ "user": { "id","email","full_name","role","balance_cents", ... },
  "access_token":"jwt", "expires_in":900 }
```

## Account / wallet (auth required)

| Method | Path | Body | Returns |
|---|---|---|---|
| GET | `/api/v1/me` | — | user object |
| GET | `/api/v1/wallet` | — | `{ balance_cents, currency }` |
| GET | `/api/v1/wallet/ledger` | — | `{ entries:[...] }` |
| POST | `/api/v1/wallet/topup` | `{ amount_cents, provider:"stripe"\|"nowpayments" }` | `{ checkout_url }` |
| GET | `/api/v1/payments` | — | `{ payments:[...] }` |

## Orders / proxies (auth required)

| Method | Path | Body | Returns |
|---|---|---|---|
| GET | `/api/v1/orders` | — | `{ orders:[...] }` |
| POST | `/api/v1/orders` | `{ plan_id, quantity, rotation?, sticky_ttl_seconds?, region? }` | `{ order, proxies:[...] }` |
| GET | `/api/v1/proxies` | — | `{ proxies:[...] }` |
| GET | `/api/v1/proxies/{id}/usage` | — | `{ remaining_gb, active, expires_at? }` |
| POST | `/api/v1/proxies/{id}/generate` (CSRF) | see below | `{ generations:[...], lines:[...], mode, note?, session_seconds? }` |
| GET | `/api/v1/locations?plan_key=&country=` | — | `{ countries:[...] }` |

Creating an order debits the wallet atomically; `402 insufficient_funds` if the
balance is too low. Provisioning failures auto-refund the wallet.

### Generating proxy lines

`POST /api/v1/proxies/{id}/generate` turns one of the caller's own services into
ready-to-use proxy lines via the upstream generator. Both the proxy and its
order are looked up scoped to the authenticated user, so one customer can never
generate against another's service.

**Generating does not consume bandwidth** — it mints credentials rather than
selling capacity, so `remaining_gb` is unchanged and customers may regenerate
freely. Bandwidth is only consumed by traffic through the proxy.

```json
{ "mode":"sticky", "count":5, "protocol":"HTTP",
  "format":"user:pass@ip:port", "country":"US", "city":"Los Angeles",
  "session_seconds":600 }
```

| Field | Notes |
|---|---|
| `mode` | `rotating` (default) or `sticky` |
| `count` | 1–10000, default 1 |
| `protocol` | `HTTP` (default) or `SOCKS5` |
| `format` | one of the nine documented layouts (`host:` accepted for `ip:`) |
| `country` / `state` / `city` / `continent` | honoured per plan; ignored where unsupported |
| `ips` | `shared_isp` only — pin lines to static IPs |
| `session_seconds` | sticky only; **clamped server-side** to the plan's documented cap (e.g. `resi_pergb` 6 h, `mobile_pergb` 2 h) so a too-large value returns working credentials instead of an upstream 400 |

The response's `hostname`/`port` come from the upstream generator, which is
authoritative per plan — gateways are never hardcoded.

In **rotating** mode every line is byte-identical: there is no session token to
differentiate them and a fresh exit IP is issued per request. That is by design
(and useful for tools that want a proxy-list file), so the response carries an
explanatory `note` when `count > 1`. Use `sticky` for distinct concurrent
session IPs.

`GET /api/v1/locations` lists the geo targets a plan supports, cached in Redis
for 6 h. `plan_key` must match a plan you actually sell. Plans without geo
targeting (`ipv6_pergb`, `resi_unlim_budget`) return an empty list.

A `proxy` object:
```json
{ "id","proxy_type","protocol":"http","host","port","username","password",
  "pool","rotation":"rotating","sticky_ttl_seconds":0,
  "bandwidth_limit_bytes":0,"bandwidth_used_bytes":0,"status":"active",
  "expires_at":null,"created_at":"..." }
```

## Admin (role=admin)

| Method | Path | Returns |
|---|---|---|
| GET | `/api/v1/admin/stats` | `{ users, active_proxies, orders, total_balance_cents, revenue_paid_cents }` |
| POST | `/api/v1/admin/catalog/sync` (CSRF) | `{ synced, deactivated, codes }` |

`POST /admin/catalog/sync` pulls the live VaultProxies reseller catalog and
upserts it into the `plans` table with `retail = round(wholesale * RESELLER_MARKUP)`,
deactivating any plan no longer offered upstream. It also runs automatically on
backend startup (best-effort).

## Webhooks (no auth; signature-verified)

- `POST /api/v1/webhooks/stripe` — verified via `Stripe-Signature`.
- `POST /api/v1/webhooks/nowpayments` — verified via `x-nowpayments-sig` HMAC-SHA512.

## Error shape

```json
{ "error":"human message", "code":"machine_code", "fields":{"Email":"required"} }
```
Common codes: `unauthenticated`, `forbidden`, `csrf`, `rate_limited`,
`invalid_credentials`, `captcha`, `insufficient_funds`, `provision_failed`,
`generate_failed`, `locations_failed`, `account_inactive`.

OAuth failures redirect to `${PUBLIC_BASE_URL}/login?error=<code>` with codes
`invalid_oauth_state`, `invalid_oauth_response`, `oauth_exchange_failed`,
`oauth_account_error`, `oauth_email_unverified`, `oauth_session_error`,
`account_inactive`.
