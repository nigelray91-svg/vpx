// Package vaultproxies is the client for the VaultProxies Reseller API
// (https://vaultproxies.net/docs): a single X-API-Key header and four endpoints
// — GET /api/reseller/balance, GET /api/reseller/categories,
// POST /api/reseller/order, GET /api/reseller/services.
//
// The live implementation (live.go) maps the documented JSON exactly and is
// verified by live_test.go against a fake server returning the docs' sample
// payloads. Run with VAULTPROXIES_MODE=mock to exercise the whole stack without
// contacting the upstream.
package vaultproxies

import (
	"context"
	"time"
)

// Rotation modes.
const (
	RotationRotating = "rotating"
	RotationSticky   = "sticky"
)

// ProvisionRequest asks the upstream to create a proxy service
// (POST /api/reseller/order).
type ProvisionRequest struct {
	CategoryKey string // e.g. resi_pergb, resi_unlim, dc_unlim, ipv6_pergb
	Units       int    // GB for pergb plans; hours/days for unlim plans
	TimeUnit    string // unlim plans only: hour | day | week | month
	ProxyType   string // residential | datacenter | ipv6 (for gateway selection)
	Label       string // our order id, for traceability
}

// ProxyCredential is a single usable endpoint returned by the upstream.
type ProxyCredential struct {
	Protocol            string `json:"protocol"`
	Host                string `json:"host"`
	Port                int    `json:"port"`
	Username            string `json:"username"`
	Password            string `json:"password"`
	Pool                string `json:"pool"`
	Rotation            string `json:"rotation"`
	StickyTTLSeconds    int    `json:"sticky_ttl_seconds"`
	BandwidthLimitBytes int64  `json:"bandwidth_limit_bytes"`
}

// ProvisionResult is the upstream's response to a provision request.
type ProvisionResult struct {
	Ref       string            // upstream order / sub-user reference
	Proxies   []ProxyCredential // one or more endpoints
	ExpiresAt *time.Time
}

// Usage reports the live state of a provisioned service
// (derived from GET /api/reseller/services).
type Usage struct {
	Ref             string
	RemainingGB     float64
	Active          bool
	ExpiresAt       *time.Time
}

// Product is one purchasable category in the upstream reseller catalog
// (GET /api/reseller/categories). The reseller platform mirrors these into its
// own `plans` table, applying a markup to derive the retail price.
type Product struct {
	Code           string // category key, e.g. resi_pergb
	Name           string
	Type           string // residential | datacenter | ipv6 (derived from key)
	Unit           string // gb | hour | day
	PricingType    string // pergb | unlim
	WholesaleCents int64  // reseller rate per unit, in USD cents
	MinQuantity    int
}

// Client is the upstream abstraction used by the rest of the app.
type Client interface {
	// Catalog returns the upstream reseller product catalog (with wholesale
	// prices) so the platform can sync it into its own plans table.
	Catalog(ctx context.Context) ([]Product, error)
	// Provision creates proxy access and returns usable credentials.
	Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
	// Usage returns consumption for a previously provisioned ref.
	Usage(ctx context.Context, ref string) (*Usage, error)
	// Revoke disables a previously provisioned ref.
	Revoke(ctx context.Context, ref string) error
	// Healthy reports whether the upstream is reachable.
	Healthy(ctx context.Context) bool
}

// Gateways maps a proxy type (residential|datacenter|ipv6) to the "host:port"
// of the upstream proxy gateway. The reseller API returns username/password but
// not the gateway endpoint, which you obtain from the reseller dashboard.
type Gateways map[string]string

// New returns a live or mock client depending on mode.
func New(mode, baseURL, apiKey string, gateways Gateways) Client {
	if mode == "mock" || apiKey == "" {
		return NewMock()
	}
	return NewLive(baseURL, apiKey, gateways)
}
