// Package models holds the core domain types shared across the application.
package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	PasswordHash  string    `json:"-"`
	FullName      string    `json:"full_name"`
	Role          string    `json:"role"`
	Status        string    `json:"status"`
	GoogleID      *string   `json:"-"`
	BalanceCents  int64     `json:"balance_cents"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	UserAgent string
	IP        string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type Plan struct {
	ID                 uuid.UUID `json:"id"`
	Code               string    `json:"code"`
	Name               string    `json:"name"`
	ProxyType          string    `json:"proxy_type"`
	Unit               string    `json:"unit"`
	WholesaleCentsUnit int64     `json:"-"`
	RetailCentsUnit    int64     `json:"retail_cents_unit"`
	MinQuantity        int       `json:"min_quantity"`
	Active             bool      `json:"active"`
	SortOrder          int       `json:"sort_order"`
	CreatedAt          time.Time `json:"created_at"`
}

type Order struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	PlanID     uuid.UUID  `json:"plan_id"`
	Quantity   int        `json:"quantity"`
	Unit       string     `json:"unit"`
	TotalCents int64      `json:"total_cents"`
	Status     string     `json:"status"`
	VaultRef   *string    `json:"vault_ref,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Proxy struct {
	ID                  uuid.UUID  `json:"id"`
	UserID              uuid.UUID  `json:"user_id"`
	OrderID             uuid.UUID  `json:"order_id"`
	ProxyType           string     `json:"proxy_type"`
	Protocol            string     `json:"protocol"`
	Host                string     `json:"host"`
	Port                int        `json:"port"`
	Username            string     `json:"username"`
	Password            string     `json:"password"`
	Pool                string     `json:"pool"`
	Rotation            string     `json:"rotation"`
	StickyTTLSeconds    int        `json:"sticky_ttl_seconds"`
	BandwidthLimitBytes int64      `json:"bandwidth_limit_bytes"`
	BandwidthUsedBytes  int64      `json:"bandwidth_used_bytes"`
	Status              string     `json:"status"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

type LedgerEntry struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Kind         string    `json:"kind"`
	AmountCents  int64     `json:"amount_cents"`
	BalanceAfter int64     `json:"balance_after"`
	Reference    string    `json:"reference"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

type Payment struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Provider    string    `json:"provider"`
	ProviderRef string    `json:"provider_ref"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	Credited    bool      `json:"credited"`
	CreatedAt   time.Time `json:"created_at"`
}
