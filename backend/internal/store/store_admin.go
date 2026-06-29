package store

import "context"

type AdminStatsResult struct {
	Users          int64 `json:"users"`
	ActiveProxies  int64 `json:"active_proxies"`
	Orders         int64 `json:"orders"`
	TotalBalance   int64 `json:"total_balance_cents"`
	RevenuePaid    int64 `json:"revenue_paid_cents"`
}

func (s *Store) AdminStats(ctx context.Context) (*AdminStatsResult, error) {
	var res AdminStatsResult
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM users WHERE status <> 'deleted'),
			(SELECT count(*) FROM proxies WHERE status = 'active'),
			(SELECT count(*) FROM orders),
			(SELECT COALESCE(sum(balance_cents),0) FROM users),
			(SELECT COALESCE(sum(amount_cents),0) FROM payments WHERE status = 'paid')
	`).Scan(&res.Users, &res.ActiveProxies, &res.Orders, &res.TotalBalance, &res.RevenuePaid)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
