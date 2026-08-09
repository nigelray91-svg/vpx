// Package vaultproxies is the client for the VaultProxies Reseller API
// (https://vaultproxies.net/docs): a single X-API-Key header and six endpoints
// — GET /api/reseller/balance, GET /api/reseller/categories,
// POST /api/reseller/order, GET /api/reseller/services,
// GET /api/reseller/proxy/generator/locations and
// POST /api/reseller/proxy/generations/create.
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
	Ref         string
	RemainingGB float64
	Active      bool
	ExpiresAt   *time.Time
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

// GenerateRequest asks the upstream generator to turn an active service into
// ready-to-use proxy lines (POST /api/reseller/proxy/generations/create).
// Which fields are honoured depends on the plan — see the per-plan capability
// table in the upstream docs.
type GenerateRequest struct {
	ServiceID int      // active upstream service id
	PlanKey   string   // must match the service's plan
	Protocol  string   // HTTP (default) | SOCKS5
	Format    string   // output line format, e.g. user:pass@ip:port
	Country   string   // ISO-2 code or full name
	Continent string   // resi_pergb only: EU|NA|SA|AS|AF|OC
	State     string   // resi_unlim, backup_pergb
	City      string   // resi_pergb, resi_unlim, backup_pergb
	IPs       []string // shared_isp only: pin lines to static IPs
	Mode      string   // rotating (default) | sticky

	Count int // 1..10000, default 1
	// SessionSeconds is the sticky session lifetime. Caps vary per plan
	// (resi_pergb 21600, shared_isp / resi_unlim_budget 86400, mobile_pergb 7200).
	SessionSeconds int
	// SessionLength is eu_isp's qualitative sticky length: long | short.
	SessionLength string
}

// Generation is a single generated proxy line. Hostname and Port are the
// authoritative gateway for the plan — never hardcode them.
type Generation struct {
	ID         int64  `json:"id"`
	ServiceID  int    `json:"service_id"`
	PlanKey    string `json:"plan_key"`
	Country    string `json:"country"`
	Protocol   string `json:"protocol"`
	Format     string `json:"format"`
	Hostname   string `json:"hostname"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	OutputLine string `json:"output_line"`
	CreatedAt  string `json:"created_at"`
}

// City, State and Country describe the generator's geo-targeting tree
// (GET /api/reseller/proxy/generator/locations).
type City struct {
	Name string `json:"name"`
}

type State struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Cities []City `json:"cities,omitempty"`
}

type Country struct {
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	States []State `json:"states,omitempty"`
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
	// Generate turns an active service into ready-to-use proxy lines,
	// returning the authoritative gateway hostname/port for the plan.
	Generate(ctx context.Context, req GenerateRequest) ([]Generation, error)
	// Locations lists the geo targets a plan supports (empty when unsupported).
	Locations(ctx context.Context, planKey, country string) ([]Country, error)
	// Revoke disables a previously provisioned ref.
	Revoke(ctx context.Context, ref string) error
	// Healthy reports whether the upstream is reachable.
	Healthy(ctx context.Context) bool
}

// Gateways maps a proxy type (residential|datacenter|ipv6|mobile|isp) to the
// "host:port" of the upstream proxy gateway. This is only a fallback for when
// the generator endpoint is unavailable — the generator returns the correct
// hostname and port per plan, so leaving these unset is normal.
type Gateways map[string]string

// New returns a live or mock client depending on mode.
func New(mode, baseURL, apiKey string, gateways Gateways) Client {
	if mode == "mock" || apiKey == "" {
		return NewMock()
	}
	return NewLive(baseURL, apiKey, gateways)
}
