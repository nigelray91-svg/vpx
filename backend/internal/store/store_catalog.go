package store

import "context"

// UpsertPlan inserts or updates a plan keyed by its upstream code. Used by the
// catalog sync to mirror the VaultProxies reseller catalog into our DB.
func (s *Store) UpsertPlan(ctx context.Context, code, name, proxyType, unit string, wholesaleCents, retailCents int64, minQty, sortOrder int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO plans (code, name, proxy_type, unit, wholesale_cents_unit, retail_cents_unit, min_quantity, sort_order, active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,TRUE)
		ON CONFLICT (code) DO UPDATE SET
			name                 = EXCLUDED.name,
			proxy_type           = EXCLUDED.proxy_type,
			unit                 = EXCLUDED.unit,
			wholesale_cents_unit = EXCLUDED.wholesale_cents_unit,
			retail_cents_unit    = EXCLUDED.retail_cents_unit,
			min_quantity         = EXCLUDED.min_quantity,
			sort_order           = EXCLUDED.sort_order,
			active               = TRUE,
			updated_at           = now()`,
		code, name, proxyType, unit, wholesaleCents, retailCents, minQty, sortOrder)
	return err
}

// DeactivatePlansNotIn marks any plan whose code is not in the supplied set as
// inactive, so retired upstream products disappear from the catalog without
// breaking existing orders that reference them.
func (s *Store) DeactivatePlansNotIn(ctx context.Context, codes []string) (int64, error) {
	if len(codes) == 0 {
		return 0, nil
	}
	tag, err := s.pool.Exec(ctx,
		`UPDATE plans SET active = FALSE, updated_at = now() WHERE code <> ALL($1) AND active = TRUE`,
		codes)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
