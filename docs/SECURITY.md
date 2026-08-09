# Security Model

This document summarizes the security controls built into VPX and the
operational steps required for a hardened production deployment.

## Authentication & sessions
- **Passwords**: bcrypt (cost 12). Min 8 / max 72 bytes enforced. Login uses a
  dummy hash compare on unknown emails to avoid user enumeration / timing leaks.
- **Access tokens**: short-lived HS256 JWTs (default 15 min) with explicit
  `alg` allow-list (`HS256` only) and issuer validation — blocks `alg=none` and
  algorithm-confusion attacks.
- **Refresh tokens**: high-entropy opaque tokens; only their SHA-256 hash is
  stored. Tokens **rotate** on every refresh and the old one is revoked
  (replay/theft mitigation). Stored in an httpOnly cookie scoped to `/api/v1/auth`.
- **Refresh reuse detection**: presenting an already-rotated refresh token means
  the token leaked (the legitimate client holds the newer one), so *all* of that
  user's sessions are revoked rather than just rejecting the request.
- **Cookies**: `HttpOnly`, `Secure` (configurable for local dev), `SameSite=Lax`,
  domain-scoped so panel↔api subdomains stay same-site.
- **CSRF**: double-submit token. State-changing cookie-authenticated requests must
  echo the readable `vpx_csrf` cookie in the `X-CSRF-Token` header (constant-time
  compare). Bearer-token API clients are exempt (no ambient credentials).
- **OAuth (Google)**: `state` nonce stored in Redis (10 min, single-use) to
  prevent OAuth CSRF; profile fetched from the OIDC userinfo endpoint. Linking a
  Google identity to an existing local account **requires `email_verified`** —
  otherwise anyone controlling a Google Workspace domain could assert an
  arbitrary address and take over the matching password account. An existing
  link is keyed on the immutable Google `sub`, so it survives regardless.
- **Bot protection**: Cloudflare Turnstile (managed) verified server-side on
  register and login.

## Authorization
- Role-based access (`user` / `admin`); admin routes gated by middleware.
- Every per-user query is scoped by `user_id` (no IDOR — e.g. proxy/order
  lookups require both the resource id and the owning user id).

## Payments integrity
- **Stripe** webhooks verified with `webhook.ConstructEvent` (signature + replay
  window). **NOWPayments** IPN verified via HMAC-SHA512 over sorted JSON with a
  constant-time comparison.
- **Idempotency**: a `webhook_events` table dedupes provider events; crediting a
  payment flips a `credited` flag inside the same DB transaction so a wallet is
  credited at most once even under duplicate/retried webhooks.
- **Wallet**: atomic debit/credit with a `balance_cents >= 0` constraint and a
  guarded conditional `UPDATE` — balances can never go negative; every movement
  writes an append-only ledger row. Provisioning failures auto-refund.

## Input / transport
- All SQL uses parameterized queries (pgx) — no string concatenation.
- JSON bodies are size-limited (1 MB) with `DisallowUnknownFields` + struct
  validation (`go-playground/validator`).
- Strict CORS: only the configured `PUBLIC_BASE_URL` origin, credentialed.
- Security headers on every response: `X-Content-Type-Options`, `X-Frame-Options:
  DENY`, `Referrer-Policy`, `COOP`, `Permissions-Policy`, and a locked-down CSP
  for the JSON API. Caddy adds HSTS at the TLS edge.

## Rate limiting & abuse
- Redis Lua **token-bucket** limiter. Tighter buckets on `/auth` (brute-force
  resistance), broader buckets on the authenticated API, keyed by user id or IP.
  Public (`/config`, `/plans`), OAuth, generator and locations routes each get
  their own bucket.
- **Client IP spoofing**: `X-Forwarded-For` / `X-Real-IP` are attacker-controlled
  unless a reverse proxy overwrites them, and honouring them blindly would let
  anyone rotate their apparent IP to bypass the per-IP auth limits. They are
  therefore only trusted when `TRUST_PROXY=true`. Note Caddy *appends* to
  `X-Forwarded-For` by default, so `deploy/Caddyfile` explicitly overwrites it
  with `{remote_host}` — enabling `TRUST_PROXY` without that (or an equivalent)
  reintroduces the bypass.

## Upstream / provisioning
- Generator calls are scoped to the caller: the proxy **and** its order are
  loaded by `(id, user_id)`, so no customer can generate credentials against
  another's service. `plan_key` for locations must match a plan actually sold.
- Sticky session lengths are clamped to the documented per-plan caps server-side.
- A failed provision refunds on a context detached from the request, so a client
  disconnect cannot strand a debited wallet; an unrecoverable refund failure is
  logged at `ERROR` with the order, user and amount for manual reconciliation.

## Operational hardening
- Backend ships as a **non-root distroless** static binary (no shell, minimal
  surface). Postgres/Redis are not published to the host (internal network only).
- Secrets come exclusively from the environment; `.env` is gitignored and the
  repo only contains `.env.example`.
- Graceful shutdown; sane HTTP server timeouts (read/write/idle/header).

## Your production checklist
- [ ] Strong, unique `JWT_SECRET` (≥48 random bytes) and `POSTGRES_PASSWORD`.
- [ ] `APP_ENV=production`, `COOKIE_SECURE=true`, correct `COOKIE_DOMAIN`.
- [ ] TLS everywhere (Caddy profile or your own LB); HSTS enabled.
- [ ] `TRUST_PROXY=true` **only** behind a proxy that overwrites
      `X-Forwarded-For`, and make sure the backend port is not also reachable
      directly (bypassing the proxy).
- [ ] Rotate `VAULTPROXIES_API_KEY` if it has ever been pasted into a chat,
      ticket, or CI log — it authorises spending your reseller balance.
- [ ] Postgres `sslmode=require`/`verify-full` if managed/remote; set a Redis password.
- [ ] Register and verify Stripe + NOWPayments webhook secrets.
- [ ] Restrict Google OAuth redirect URIs; verify Turnstile keys.
- [ ] Confirm the VaultProxies endpoint mapping against the live `/docs`.
- [ ] Configure backups for the `pgdata` volume and a log/alerting pipeline.
- [ ] Run a dependency scan (`govulncheck`, `npm audit`) in CI before deploy.
