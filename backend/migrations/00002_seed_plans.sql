-- +goose Up
-- +goose StatementBegin
-- Default resale catalog. Wholesale prices are placeholders aligned to the
-- VaultProxies wholesale model; retail = wholesale * default markup. Adjust to
-- match your real upstream pricing and margin via the admin API or SQL.
INSERT INTO plans (code, name, proxy_type, unit, wholesale_cents_unit, retail_cents_unit, min_quantity, sort_order)
VALUES
    ('resi-rotating', 'Residential — Rotating',      'residential', 'gb',  150,  300, 1, 10),
    ('resi-sticky',   'Residential — Sticky Session','residential', 'gb',  175,  350, 1, 20),
    ('isp-static',    'ISP / Static Residential',    'isp',         'ip',  120,  250, 1, 30),
    ('dc-shared',     'Datacenter — Shared',         'datacenter',  'ip',   40,   90, 5, 40),
    ('ipv6-pool',     'IPv6 Pool',                   'ipv6',        'gb',   25,   60, 1, 50),
    ('mobile-4g',     'Mobile 4G/5G',                'mobile',      'gb',  400,  800, 1, 60)
ON CONFLICT (code) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM plans WHERE code IN
    ('resi-rotating','resi-sticky','isp-static','dc-shared','ipv6-pool','mobile-4g');
-- +goose StatementEnd
