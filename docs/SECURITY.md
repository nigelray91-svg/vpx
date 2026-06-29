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
- **Cookies**: `HttpOnly`, `Secure` (configurable for local dev), `SameSite=Lax`,
  domain-scoped so panel↔api subdomains stay same-site.
- **CSRF**: double-submit token. State-changing cookie-authenticated requests must
  echo the readable `vpx_csrf` cookie in the `X-CSRF-Token` header (constant-time
  compare). Bearer-token API clients are exempt (no ambient credentials).
- **OAuth (Google)**: `state` nonce stored in Redis (10 min, single-use) to
  prevent OAuth CSRF; profile fetched from the OIDC userinfo endpoint; accounts
  are linked by verified email.
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
- [ ] Postgres `sslmode=require`/`verify-full` if managed/remote; set a Redis password.
- [ ] Register and verify Stripe + NOWPayments webhook secrets.
- [ ] Restrict Google OAuth redirect URIs; verify Turnstile keys.
- [ ] Confirm the VaultProxies endpoint mapping against the live `/docs`.
- [ ] Configure backups for the `pgdata` volume and a log/alerting pipeline.
- [ ] Run a dependency scan (`govulncheck`, `npm audit`) in CI before deploy.
