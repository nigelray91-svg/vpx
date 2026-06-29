-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- Users
-- ---------------------------------------------------------------------------
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL,
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    password_hash   TEXT,                       -- NULL for OAuth-only accounts
    full_name       TEXT NOT NULL DEFAULT '',
    role            TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user','admin')),
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended','deleted')),
    google_id       TEXT UNIQUE,
    balance_cents   BIGINT NOT NULL DEFAULT 0 CHECK (balance_cents >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- email is stored lowercased by the application; enforce uniqueness here.
CREATE UNIQUE INDEX users_email_key ON users (lower(email));

-- ---------------------------------------------------------------------------
-- Refresh-token sessions (opaque token, only the hash is stored)
-- ---------------------------------------------------------------------------
CREATE TABLE sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL UNIQUE,          -- sha256 of opaque refresh token
    user_agent   TEXT NOT NULL DEFAULT '',
    ip           TEXT NOT NULL DEFAULT '',
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- ---------------------------------------------------------------------------
-- Plans / products available for resale
-- ---------------------------------------------------------------------------
CREATE TABLE plans (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                  TEXT NOT NULL UNIQUE,
    name                  TEXT NOT NULL,
    proxy_type            TEXT NOT NULL CHECK (proxy_type IN ('residential','isp','datacenter','ipv6','mobile')),
    unit                  TEXT NOT NULL CHECK (unit IN ('gb','ip','port')),
    wholesale_cents_unit  BIGINT NOT NULL CHECK (wholesale_cents_unit >= 0),
    retail_cents_unit     BIGINT NOT NULL CHECK (retail_cents_unit >= 0),
    min_quantity          INTEGER NOT NULL DEFAULT 1,
    active                BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order            INTEGER NOT NULL DEFAULT 0,
    metadata              JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- Orders (a purchase that provisions proxy access upstream)
-- ---------------------------------------------------------------------------
CREATE TABLE orders (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id       UUID NOT NULL REFERENCES plans(id),
    quantity      INTEGER NOT NULL CHECK (quantity > 0),
    unit          TEXT NOT NULL,
    total_cents   BIGINT NOT NULL CHECK (total_cents >= 0),
    status        TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','provisioning','active','failed','cancelled','expired')),
    vault_ref     TEXT,                         -- upstream order/subuser reference
    expires_at    TIMESTAMPTZ,
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX orders_user_id_idx ON orders (user_id);
CREATE INDEX orders_status_idx ON orders (status);

-- ---------------------------------------------------------------------------
-- Provisioned proxy credentials / endpoints
-- ---------------------------------------------------------------------------
CREATE TABLE proxies (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id             UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    proxy_type           TEXT NOT NULL,
    protocol             TEXT NOT NULL DEFAULT 'http' CHECK (protocol IN ('http','https','socks5')),
    host                 TEXT NOT NULL,
    port                 INTEGER NOT NULL,
    username             TEXT NOT NULL,
    password             TEXT NOT NULL,
    pool                 TEXT NOT NULL DEFAULT '',
    rotation             TEXT NOT NULL DEFAULT 'rotating' CHECK (rotation IN ('rotating','sticky')),
    sticky_ttl_seconds   INTEGER NOT NULL DEFAULT 0,
    bandwidth_limit_bytes BIGINT NOT NULL DEFAULT 0,
    bandwidth_used_bytes  BIGINT NOT NULL DEFAULT 0,
    vault_ref            TEXT,
    status               TEXT NOT NULL DEFAULT 'active'
                         CHECK (status IN ('active','suspended','expired','revoked')),
    expires_at           TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX proxies_user_id_idx ON proxies (user_id);
CREATE INDEX proxies_order_id_idx ON proxies (order_id);

-- ---------------------------------------------------------------------------
-- Wallet ledger (append-only, double-entry-ish single account view)
-- ---------------------------------------------------------------------------
CREATE TABLE ledger_entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind            TEXT NOT NULL CHECK (kind IN ('topup','debit','refund','adjustment')),
    amount_cents    BIGINT NOT NULL,            -- positive credit / negative debit
    balance_after   BIGINT NOT NULL,
    reference       TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ledger_user_id_idx ON ledger_entries (user_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- Payments (Stripe / NOWPayments)
-- ---------------------------------------------------------------------------
CREATE TABLE payments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider      TEXT NOT NULL CHECK (provider IN ('stripe','nowpayments')),
    provider_ref  TEXT NOT NULL,                -- session id / payment id
    amount_cents  BIGINT NOT NULL CHECK (amount_cents > 0),
    currency      TEXT NOT NULL DEFAULT 'usd',
    status        TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','confirming','paid','failed','expired','refunded')),
    credited      BOOLEAN NOT NULL DEFAULT FALSE,
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_ref)
);
CREATE INDEX payments_user_id_idx ON payments (user_id);

-- ---------------------------------------------------------------------------
-- Webhook event idempotency log
-- ---------------------------------------------------------------------------
CREATE TABLE webhook_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider    TEXT NOT NULL,
    event_id    TEXT NOT NULL,
    event_type  TEXT NOT NULL DEFAULT '',
    processed   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, event_id)
);

-- ---------------------------------------------------------------------------
-- updated_at trigger
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at    BEFORE UPDATE ON users    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER plans_updated_at    BEFORE UPDATE ON plans    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER orders_updated_at   BEFORE UPDATE ON orders   FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER proxies_updated_at  BEFORE UPDATE ON proxies  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER payments_updated_at BEFORE UPDATE ON payments FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS webhook_events;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS ledger_entries;
DROP TABLE IF EXISTS proxies;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS plans;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS set_updated_at();
-- +goose StatementEnd
