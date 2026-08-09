// Package store is the PostgreSQL persistence layer (pgx).
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/vaultproxies/vpx/backend/internal/models"
	"github.com/vaultproxies/vpx/backend/migrations"
)

var ErrNotFound = errors.New("not found")
var ErrInsufficientFunds = errors.New("insufficient funds")

type Store struct {
	pool *pgxpool.Pool
}

// New opens a pooled connection to Postgres.
func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MaxConnLifetime = time.Hour
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// Migrate applies all embedded migrations using a temporary database/sql handle.
func Migrate(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open for migrate: %w", err)
	}
	defer db.Close()
	_ = stdlib.GetDefaultDriver // ensure pgx stdlib driver is linked

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, fullName string, googleID *string, emailVerified bool) (*models.User, error) {
	const q = `
		INSERT INTO users (email, password_hash, full_name, google_id, email_verified)
		VALUES ($1, NULLIF($2,''), $3, $4, $5)
		RETURNING id, email, email_verified, COALESCE(password_hash,''), full_name, role, status, google_id, balance_cents, created_at, updated_at`
	var u models.User
	err := s.pool.QueryRow(ctx, q, email, passwordHash, fullName, googleID, emailVerified).Scan(
		&u.ID, &u.Email, &u.EmailVerified, &u.PasswordHash, &u.FullName, &u.Role, &u.Status, &u.GoogleID, &u.BalanceCents, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Email, &u.EmailVerified, &u.PasswordHash, &u.FullName,
		&u.Role, &u.Status, &u.GoogleID, &u.BalanceCents, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

const userCols = `id, email, email_verified, COALESCE(password_hash,''), full_name, role, status, google_id, balance_cents, created_at, updated_at`

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE lower(email)=lower($1)`, email))
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1`, id))
}

func (s *Store) GetUserByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE google_id=$1`, googleID))
}

func (s *Store) LinkGoogleID(ctx context.Context, userID uuid.UUID, googleID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET google_id=$1, email_verified=TRUE WHERE id=$2`, googleID, userID)
	return err
}

func (s *Store) SetPassword(ctx context.Context, userID uuid.UUID, hash string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET password_hash=$1 WHERE id=$2`, hash, userID)
	return err
}

func (s *Store) PromoteAdmin(ctx context.Context, email string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET role='admin' WHERE lower(email)=lower($1)`, email)
	return err
}

// ---------------------------------------------------------------------------
// Sessions (refresh tokens)
// ---------------------------------------------------------------------------

func (s *Store) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash, ua, ip string, expires time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (user_id, token_hash, user_agent, ip, expires_at) VALUES ($1,$2,$3,$4,$5)`,
		userID, tokenHash, ua, ip, expires)
	return err
}

func (s *Store) GetSessionByHash(ctx context.Context, tokenHash string) (*models.Session, error) {
	var se models.Session
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, user_agent, ip, expires_at, revoked_at, created_at
		 FROM sessions WHERE token_hash=$1`, tokenHash).
		Scan(&se.ID, &se.UserID, &se.TokenHash, &se.UserAgent, &se.IP, &se.ExpiresAt, &se.RevokedAt, &se.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &se, nil
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`, tokenHash)
	return err
}

func (s *Store) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return err
}

// ---------------------------------------------------------------------------
// Plans
// ---------------------------------------------------------------------------

const planCols = `id, code, name, proxy_type, unit, wholesale_cents_unit, retail_cents_unit, min_quantity, active, sort_order, created_at`

func scanPlan(row pgx.Row) (*models.Plan, error) {
	var p models.Plan
	err := row.Scan(&p.ID, &p.Code, &p.Name, &p.ProxyType, &p.Unit, &p.WholesaleCentsUnit,
		&p.RetailCentsUnit, &p.MinQuantity, &p.Active, &p.SortOrder, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) ListActivePlans(ctx context.Context) ([]models.Plan, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+planCols+` FROM plans WHERE active=TRUE ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Plan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (s *Store) GetPlanByID(ctx context.Context, id uuid.UUID) (*models.Plan, error) {
	return scanPlan(s.pool.QueryRow(ctx, `SELECT `+planCols+` FROM plans WHERE id=$1`, id))
}

// ---------------------------------------------------------------------------
// Wallet — atomic debit/credit via the ledger
// ---------------------------------------------------------------------------

// Credit adds funds and writes a ledger entry atomically. Idempotency must be
// enforced by the caller (e.g. via payments.credited flag in the same tx).
func (s *Store) Credit(ctx context.Context, userID uuid.UUID, kind string, amount int64, reference, desc string) (int64, error) {
	if amount <= 0 {
		return 0, errors.New("credit amount must be positive")
	}
	var balance int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`UPDATE users SET balance_cents = balance_cents + $1 WHERE id=$2 RETURNING balance_cents`,
			amount, userID).Scan(&balance); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO ledger_entries (user_id, kind, amount_cents, balance_after, reference, description)
			 VALUES ($1,$2,$3,$4,$5,$6)`, userID, kind, amount, balance, reference, desc)
		return err
	})
	return balance, err
}

// Debit subtracts funds, failing if the balance would go negative.
func (s *Store) Debit(ctx context.Context, userID uuid.UUID, amount int64, reference, desc string) (int64, error) {
	if amount <= 0 {
		return 0, errors.New("debit amount must be positive")
	}
	var balance int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE users SET balance_cents = balance_cents - $1 WHERE id=$2 AND balance_cents >= $1`,
			amount, userID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrInsufficientFunds
		}
		if err := tx.QueryRow(ctx, `SELECT balance_cents FROM users WHERE id=$1`, userID).Scan(&balance); err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO ledger_entries (user_id, kind, amount_cents, balance_after, reference, description)
			 VALUES ($1,'debit',$2,$3,$4,$5)`, userID, -amount, balance, reference, desc)
		return err
	})
	return balance, err
}

func (s *Store) ListLedger(ctx context.Context, userID uuid.UUID, limit int) ([]models.LedgerEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, kind, amount_cents, balance_after, reference, description, created_at
		 FROM ledger_entries WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.LedgerEntry
	for rows.Next() {
		var e models.LedgerEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Kind, &e.AmountCents, &e.BalanceAfter, &e.Reference, &e.Description, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
