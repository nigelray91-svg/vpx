// Package vaultproxies is the client for the VaultProxies wholesale reseller
// API (https://vaultproxies.net/docs).
//
// IMPORTANT: The exact upstream endpoint paths and JSON field names could not
// be fetched from this build environment (egress policy blocks vaultproxies.net),
// so the live client below targets the documented reseller model — provision a
// sub-user / order for a proxy type, receive endpoint credentials, query usage.
// Every path and payload field is centralized in this file: align the
// `endpoints` map and the request/response structs with the real /docs and the
// rest of the application needs no changes. Run with VAULTPROXIES_MODE=mock to
// exercise the whole stack without contacting the upstream.
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

// ProvisionRequest asks the upstream to create proxy access.
type ProvisionRequest struct {
	ProxyType   string // residential | isp | datacenter | ipv6 | mobile
	Unit        string // gb | ip | port
	Quantity    int
	Rotation    string // rotating | sticky
	StickyTTLSec int
	Pool        string // optional sub-pool / product code
	Region      string // optional geo targeting
	Label       string // our order id, for upstream traceability
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

// Usage reports consumption for a provisioned ref.
type Usage struct {
	Ref                string
	BandwidthUsedBytes int64
	BandwidthCapBytes  int64
	Active             bool
}

// Client is the upstream abstraction used by the rest of the app.
type Client interface {
	// Provision creates proxy access and returns usable credentials.
	Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
	// Usage returns consumption for a previously provisioned ref.
	Usage(ctx context.Context, ref string) (*Usage, error)
	// Revoke disables a previously provisioned ref.
	Revoke(ctx context.Context, ref string) error
	// Healthy reports whether the upstream is reachable.
	Healthy(ctx context.Context) bool
}

// New returns a live or mock client depending on mode.
func New(mode, baseURL, apiKey string) Client {
	if mode == "mock" || apiKey == "" {
		return NewMock()
	}
	return NewLive(baseURL, apiKey)
}
