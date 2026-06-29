package vaultproxies

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// Mock implements Client with deterministic-looking fake endpoints so the full
// stack runs locally without contacting the upstream. NEVER returns real proxies.
type Mock struct{}

func NewMock() *Mock { return &Mock{} }

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Catalog returns a representative wholesale catalog so the sync path runs
// end-to-end without the upstream. Replace with live data via VAULTPROXIES_MODE=live.
func (m *Mock) Catalog(ctx context.Context) ([]Product, error) {
	return []Product{
		{Code: "resi-rotating", Name: "Residential — Rotating", Type: "residential", Unit: "gb", WholesaleCents: 175, MinQuantity: 1},
		{Code: "resi-sticky", Name: "Residential — Sticky Session", Type: "residential", Unit: "gb", WholesaleCents: 190, MinQuantity: 1},
		{Code: "isp-static", Name: "ISP / Static Residential", Type: "isp", Unit: "ip", WholesaleCents: 150, MinQuantity: 1},
		{Code: "dc-ip", Name: "Datacenter — Dedicated IP", Type: "datacenter", Unit: "ip", WholesaleCents: 50, MinQuantity: 3},
		{Code: "dc-gb", Name: "Datacenter — Bandwidth", Type: "datacenter", Unit: "gb", WholesaleCents: 30, MinQuantity: 1},
		{Code: "ipv6-pool", Name: "IPv6 — Bandwidth", Type: "ipv6", Unit: "gb", WholesaleCents: 25, MinQuantity: 1},
		{Code: "mobile-4g", Name: "Mobile 4G/5G", Type: "mobile", Unit: "gb", WholesaleCents: 350, MinQuantity: 1},
	}, nil
}

func (m *Mock) Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
	ref := "mock_" + randHex(8)
	exp := time.Now().Add(30 * 24 * time.Hour)
	port := 10000
	switch req.ProxyType {
	case "residential", "mobile":
		port = 7777
	case "isp":
		port = 8080
	case "datacenter":
		port = 3128
	case "ipv6":
		port = 9090
	}
	cred := ProxyCredential{
		Protocol:            "http",
		Host:                fmt.Sprintf("gw.%s.mock-vaultproxies.local", req.ProxyType),
		Port:                port,
		Username:            "u-" + randHex(4),
		Password:            randHex(8),
		Pool:                req.Pool,
		Rotation:            req.Rotation,
		StickyTTLSeconds:    req.StickyTTLSec,
		BandwidthLimitBytes: int64(req.Quantity) * 1024 * 1024 * 1024,
	}
	return &ProvisionResult{Ref: ref, Proxies: []ProxyCredential{cred}, ExpiresAt: &exp}, nil
}

func (m *Mock) Usage(ctx context.Context, ref string) (*Usage, error) {
	return &Usage{Ref: ref, BandwidthUsedBytes: 0, BandwidthCapBytes: 1024 * 1024 * 1024, Active: true}, nil
}

func (m *Mock) Revoke(ctx context.Context, ref string) error { return nil }

func (m *Mock) Healthy(ctx context.Context) bool { return true }
