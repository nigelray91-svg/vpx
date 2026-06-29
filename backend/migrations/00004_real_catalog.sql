-- +goose Up
-- +goose StatementBegin
-- Align the catalog with the real VaultProxies reseller API
-- (GET /api/reseller/categories): four categories keyed resi_pergb, resi_unlim,
-- dc_unlim, ipv6_pergb, with pergb (unit GB) and unlim (unit HOUR/DAY) pricing.

-- Allow time-window units used by "unlim" plans.
ALTER TABLE plans DROP CONSTRAINT IF EXISTS plans_unit_check;
ALTER TABLE plans ADD CONSTRAINT plans_unit_check
    CHECK (unit IN ('gb','ip','port','hour','day','week','month'));

-- Retire the earlier placeholder plans (the live/mock catalog sync also does
-- this automatically, but do it here so a fresh offline DB is correct too).
UPDATE plans SET active = FALSE
WHERE code IN ('resi-rotating','resi-sticky','isp-static','dc-ip','dc-gb','ipv6-pool','mobile-4g');

-- Fallback catalog (used until the first successful sync). Wholesale = example
-- reseller rate from the docs; retail = wholesale * 2 (default markup).
INSERT INTO plans (code, name, proxy_type, unit, wholesale_cents_unit, retail_cents_unit, min_quantity, sort_order, active)
VALUES
    ('resi_pergb', 'Residential Per GB',     'residential', 'gb',    50, 100, 1, 10, TRUE),
    ('resi_unlim', 'Residential Unlimited',  'residential', 'hour', 200, 400, 1, 20, TRUE),
    ('dc_unlim',   'Datacenter Unlimited',   'datacenter',  'day',  250, 500, 1, 40, TRUE),
    ('ipv6_pergb', 'IPv6 Per GB',            'ipv6',        'gb',    18,  36, 1, 60, TRUE)
ON CONFLICT (code) DO UPDATE SET
    name                 = EXCLUDED.name,
    proxy_type           = EXCLUDED.proxy_type,
    unit                 = EXCLUDED.unit,
    wholesale_cents_unit = EXCLUDED.wholesale_cents_unit,
    retail_cents_unit    = EXCLUDED.retail_cents_unit,
    min_quantity         = EXCLUDED.min_quantity,
    sort_order           = EXCLUDED.sort_order,
    active               = TRUE,
    updated_at           = now();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM plans WHERE code IN ('resi_pergb','resi_unlim','dc_unlim','ipv6_pergb');
ALTER TABLE plans DROP CONSTRAINT IF EXISTS plans_unit_check;
ALTER TABLE plans ADD CONSTRAINT plans_unit_check CHECK (unit IN ('gb','ip','port'));
UPDATE plans SET active = TRUE
WHERE code IN ('resi-rotating','resi-sticky','isp-static','dc-ip','dc-gb','ipv6-pool','mobile-4g');
-- +goose StatementEnd
