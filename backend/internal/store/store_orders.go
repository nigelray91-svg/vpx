package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vaultproxies/vpx/backend/internal/models"
)

// ---------------------------------------------------------------------------
// Orders
// ---------------------------------------------------------------------------

const orderCols = `id, user_id, plan_id, quantity, unit, total_cents, status, vault_ref, expires_at, created_at`

func scanOrder(row pgx.Row) (*models.Order, error) {
	var o models.Order
	err := row.Scan(&o.ID, &o.UserID, &o.PlanID, &o.Quantity, &o.Unit, &o.TotalCents, &o.Status, &o.VaultRef, &o.ExpiresAt, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *Store) CreateOrder(ctx context.Context, userID, planID uuid.UUID, qty int, unit string, total int64) (*models.Order, error) {
	return scanOrder(s.pool.QueryRow(ctx,
		`INSERT INTO orders (user_id, plan_id, quantity, unit, total_cents, status)
		 VALUES ($1,$2,$3,$4,$5,'pending')
		 RETURNING `+orderCols, userID, planID, qty, unit, total))
}

func (s *Store) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string, vaultRef *string, expires *time.Time) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE orders SET status=$1, vault_ref=COALESCE($2, vault_ref), expires_at=COALESCE($3, expires_at) WHERE id=$4`,
		status, vaultRef, expires, id)
	return err
}

func (s *Store) GetOrder(ctx context.Context, id, userID uuid.UUID) (*models.Order, error) {
	return scanOrder(s.pool.QueryRow(ctx, `SELECT `+orderCols+` FROM orders WHERE id=$1 AND user_id=$2`, id, userID))
}

func (s *Store) ListOrders(ctx context.Context, userID uuid.UUID) ([]models.Order, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT o.id, o.user_id, o.plan_id, o.quantity, o.unit, o.total_cents, o.status,
		        o.vault_ref, o.expires_at, o.created_at,
		        COALESCE(p.name,''), COALESCE(p.code,''), COALESCE(p.proxy_type,'')
		 FROM orders o LEFT JOIN plans p ON p.id = o.plan_id
		 WHERE o.user_id=$1 ORDER BY o.created_at DESC LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.PlanID, &o.Quantity, &o.Unit, &o.TotalCents,
			&o.Status, &o.VaultRef, &o.ExpiresAt, &o.CreatedAt,
			&o.PlanName, &o.PlanCode, &o.ProxyType); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Proxies
// ---------------------------------------------------------------------------

const proxyCols = `id, user_id, order_id, proxy_type, protocol, host, port, username, password, pool, rotation, sticky_ttl_seconds, bandwidth_limit_bytes, bandwidth_used_bytes, status, expires_at, created_at`

func scanProxy(row pgx.Row) (*models.Proxy, error) {
	var p models.Proxy
	err := row.Scan(&p.ID, &p.UserID, &p.OrderID, &p.ProxyType, &p.Protocol, &p.Host, &p.Port,
		&p.Username, &p.Password, &p.Pool, &p.Rotation, &p.StickyTTLSeconds,
		&p.BandwidthLimitBytes, &p.BandwidthUsedBytes, &p.Status, &p.ExpiresAt, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) CreateProxy(ctx context.Context, p *models.Proxy, vaultRef string) (*models.Proxy, error) {
	return scanProxy(s.pool.QueryRow(ctx,
		`INSERT INTO proxies (user_id, order_id, proxy_type, protocol, host, port, username, password, pool, rotation, sticky_ttl_seconds, bandwidth_limit_bytes, vault_ref, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		 RETURNING `+proxyCols,
		p.UserID, p.OrderID, p.ProxyType, p.Protocol, p.Host, p.Port, p.Username, p.Password,
		p.Pool, p.Rotation, p.StickyTTLSeconds, p.BandwidthLimitBytes, vaultRef, p.ExpiresAt))
}

func (s *Store) ListProxies(ctx context.Context, userID uuid.UUID) ([]models.Proxy, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+proxyCols+` FROM proxies WHERE user_id=$1 ORDER BY created_at DESC LIMIT 500`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (s *Store) GetProxy(ctx context.Context, id, userID uuid.UUID) (*models.Proxy, error) {
	return scanProxy(s.pool.QueryRow(ctx, `SELECT `+proxyCols+` FROM proxies WHERE id=$1 AND user_id=$2`, id, userID))
}

// ---------------------------------------------------------------------------
// Payments
// ---------------------------------------------------------------------------

func (s *Store) CreatePayment(ctx context.Context, userID uuid.UUID, provider, ref string, amount int64, currency string, meta map[string]any) (*models.Payment, error) {
	raw, _ := json.Marshal(meta)
	var p models.Payment
	err := s.pool.QueryRow(ctx,
		`INSERT INTO payments (user_id, provider, provider_ref, amount_cents, currency, metadata)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (provider, provider_ref) DO UPDATE SET updated_at=now()
		 RETURNING id, user_id, provider, provider_ref, amount_cents, currency, status, credited, created_at`,
		userID, provider, ref, amount, currency, raw).
		Scan(&p.ID, &p.UserID, &p.Provider, &p.ProviderRef, &p.AmountCents, &p.Currency, &p.Status, &p.Credited, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// MarkPaymentPaidAndCredit transitions a payment to paid and credits the wallet
// exactly once, all in a single transaction (idempotent on the credited flag).
func (s *Store) MarkPaymentPaidAndCredit(ctx context.Context, provider, ref string) (credited bool, err error) {
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var (
			userID          uuid.UUID
			amount          int64
			alreadyCredited bool
		)
		row := tx.QueryRow(ctx,
			`SELECT user_id, amount_cents, credited FROM payments
			 WHERE provider=$1 AND provider_ref=$2 FOR UPDATE`, provider, ref)
		if e := row.Scan(&userID, &amount, &alreadyCredited); e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return e
		}
		if alreadyCredited {
			credited = false // no-op, already done
			return nil
		}
		var balance int64
		if e := tx.QueryRow(ctx,
			`UPDATE users SET balance_cents = balance_cents + $1 WHERE id=$2 RETURNING balance_cents`,
			amount, userID).Scan(&balance); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx,
			`INSERT INTO ledger_entries (user_id, kind, amount_cents, balance_after, reference, description)
			 VALUES ($1,'topup',$2,$3,$4,$5)`, userID, amount, balance, ref, provider+" top-up"); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx,
			`UPDATE payments SET status='paid', credited=TRUE WHERE provider=$1 AND provider_ref=$2`,
			provider, ref); e != nil {
			return e
		}
		credited = true
		return nil
	})
	return credited, err
}

func (s *Store) UpdatePaymentStatus(ctx context.Context, provider, ref, status string) error {
	_, err := s.pool.Exec(ctx, `UPDATE payments SET status=$1 WHERE provider=$2 AND provider_ref=$3`, status, provider, ref)
	return err
}

func (s *Store) ListPayments(ctx context.Context, userID uuid.UUID) ([]models.Payment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, provider, provider_ref, amount_cents, currency, status, credited, created_at
		 FROM payments WHERE user_id=$1 ORDER BY created_at DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Payment
	for rows.Next() {
		var p models.Payment
		if err := rows.Scan(&p.ID, &p.UserID, &p.Provider, &p.ProviderRef, &p.AmountCents, &p.Currency, &p.Status, &p.Credited, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Webhook idempotency
// ---------------------------------------------------------------------------

// MarkWebhookSeen returns true if this is the first time the event is seen.
func (s *Store) MarkWebhookSeen(ctx context.Context, provider, eventID, eventType string) (firstSeen bool, err error) {
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO webhook_events (provider, event_id, event_type, processed)
		 VALUES ($1,$2,$3,TRUE) ON CONFLICT (provider, event_id) DO NOTHING`,
		provider, eventID, eventType)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
