package vaultproxies

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"
)

// Mock implements Client with deterministic-looking fake data mirroring the
// real reseller API, so the full stack runs locally without the upstream.
type Mock struct{}

func NewMock() *Mock { return &Mock{} }

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Catalog mirrors the four real reseller categories with example reseller rates
// (cents per unit). Replace with live data via VAULTPROXIES_MODE=live.
func (m *Mock) Catalog(ctx context.Context) ([]Product, error) {
	return []Product{
		{Code: "resi_pergb", Name: "Residential Per GB", Type: "residential", Unit: "gb", PricingType: "pergb", WholesaleCents: 50, MinQuantity: 1},
		{Code: "resi_unlim", Name: "Residential Unlimited", Type: "residential", Unit: "hour", PricingType: "unlim", WholesaleCents: 200, MinQuantity: 1},
		{Code: "dc_unlim", Name: "Datacenter Unlimited", Type: "datacenter", Unit: "day", PricingType: "unlim", WholesaleCents: 250, MinQuantity: 1},
		{Code: "ipv6_pergb", Name: "IPv6 Per GB", Type: "ipv6", Unit: "gb", PricingType: "pergb", WholesaleCents: 18, MinQuantity: 1},
	}, nil
}

func (m *Mock) Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
	host := "gw." + req.ProxyType + ".mock-vaultproxies.local"
	port := 8000
	switch req.ProxyType {
	case "residential":
		port = 8000
	case "datacenter":
		port = 9000
	case "ipv6":
		port = 9090
	}
	var limit int64
	if req.Units > 0 {
		limit = int64(req.Units) * 1e9
	}
	cred := ProxyCredential{
		Protocol:            "http",
		Host:                host,
		Port:                port,
		Username:            "u" + randHex(4),
		Password:            randHex(8),
		Pool:                req.CategoryKey,
		Rotation:            "rotating",
		BandwidthLimitBytes: limit,
	}
	ref := strconv.Itoa(100000 + int(randHex(2)[0]))
	return &ProvisionResult{Ref: ref, Proxies: []ProxyCredential{cred}}, nil
}

func (m *Mock) Usage(ctx context.Context, ref string) (*Usage, error) {
	exp := time.Now().Add(30 * 24 * time.Hour)
	return &Usage{Ref: ref, RemainingGB: 4.5, Active: true, ExpiresAt: &exp}, nil
}

func (m *Mock) Revoke(ctx context.Context, ref string) error { return nil }

func (m *Mock) Healthy(ctx context.Context) bool { return true }
