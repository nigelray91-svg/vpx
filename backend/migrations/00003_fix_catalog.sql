-- +goose Up
-- +goose StatementBegin
-- Corrected, comprehensive resale catalog with market-realistic prices.
-- Adds a datacenter "per GB" (bandwidth) option alongside "per IP", keeps
-- mobile "per GB", and normalizes pricing across every proxy type.
-- Prices are USD cents per unit (gb / ip). retail = what the customer pays;
-- wholesale = your VaultProxies cost — adjust both to your real numbers.

-- Replace the old single datacenter (per-IP only) entry.
DELETE FROM plans WHERE code = 'dc-shared';

INSERT INTO plans (code, name, proxy_type, unit, wholesale_cents_unit, retail_cents_unit, min_quantity, sort_order, active)
VALUES
    ('resi-rotating', 'Residential — Rotating',       'residential', 'gb', 175, 280, 1,  10, TRUE),
    ('resi-sticky',   'Residential — Sticky Session',  'residential', 'gb', 190, 320, 1,  20, TRUE),
    ('isp-static',    'ISP / Static Residential',      'isp',         'ip', 150, 250, 1,  30, TRUE),
    ('dc-ip',         'Datacenter — Dedicated IP',     'datacenter',  'ip',  50, 100, 3,  40, TRUE),
    ('dc-gb',         'Datacenter — Bandwidth',        'datacenter',  'gb',  30,  60, 1,  50, TRUE),
    ('ipv6-pool',     'IPv6 — Bandwidth',              'ipv6',        'gb',  25,  50, 1,  60, TRUE),
    ('mobile-4g',     'Mobile 4G/5G',                  'mobile',      'gb', 350, 650, 1,  70, TRUE)
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
DELETE FROM plans WHERE code IN ('dc-ip','dc-gb');
-- Restore the previous datacenter entry.
INSERT INTO plans (code, name, proxy_type, unit, wholesale_cents_unit, retail_cents_unit, min_quantity, sort_order)
VALUES ('dc-shared', 'Datacenter — Shared', 'datacenter', 'ip', 40, 90, 5, 40)
ON CONFLICT (code) DO NOTHING;
-- +goose StatementEnd
