package vaultproxies

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
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
	// Mirror the live flow: the gateway comes from the generator's per-plan
	// table, not from a hardcoded per-type guess.
	gw, ok := mockHosts[req.CategoryKey]
	if !ok {
		gw = struct {
			Host string
			Port int
		}{"gw." + req.ProxyType + ".mock-vaultproxies.local", 8000}
	}
	host, port := gw.Host, gw.Port
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

// mockHosts mirrors the per-plan gateway table from the upstream docs so the
// mock produces realistically-shaped lines.
var mockHosts = map[string]struct {
	Host string
	Port int
}{
	"resi_pergb":   {"resi-gb.vaultproxies.com", 80},
	"resi_unlim":   {"resi.vaultproxies.com", 8080},
	"dc_unlim":     {"eu-dc.vaultproxies.com", 10808},
	"dc_pergb":     {"dc-gb.vaultproxies.com", 777},
	"mobile_pergb": {"mobile.vaultproxies.com", 8080},
	"backup_pergb": {"na.vaultproxies.com", 80},
	"shared_isp":   {"isp.vaultproxies.com", 30},
	"eu_isp":       {"eu-isp.vaultproxies.com", 30},
	"ipv6_pergb":   {"ipv6.vaultproxies.com", 30},
}

func (m *Mock) Generate(ctx context.Context, req GenerateRequest) ([]Generation, error) {
	gw, ok := mockHosts[req.PlanKey]
	if !ok {
		gw = struct {
			Host string
			Port int
		}{"resi-gb.mock-vaultproxies.local", 80}
	}
	count := req.Count
	if count < 1 {
		count = 1
	}
	if count > 10000 {
		count = 10000
	}
	format := req.Format
	if format == "" {
		format = "ip:port:user:pass"
	}
	base := "u" + randHex(4)
	pass := randHex(8)

	out := make([]Generation, 0, count)
	for i := 0; i < count; i++ {
		user := base
		if req.Country != "" {
			user += "-geo-" + strings.ToLower(req.Country)
		}
		if req.Mode == RotationSticky {
			ttl := req.SessionSeconds
			if ttl <= 0 {
				ttl = 600
			}
			user += "-sess-" + randHex(4) + "-life-" + strconv.Itoa(ttl)
		}
		out = append(out, Generation{
			ID:         time.Now().UnixNano() + int64(i),
			ServiceID:  req.ServiceID,
			PlanKey:    req.PlanKey,
			Country:    strings.ToLower(req.Country),
			Protocol:   firstNonEmpty(req.Protocol, "HTTP"),
			Format:     format,
			Hostname:   gw.Host,
			Port:       gw.Port,
			Username:   user,
			Password:   pass,
			OutputLine: FormatLine(format, gw.Host, gw.Port, user, pass, firstNonEmpty(req.Protocol, "HTTP")),
			CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

func (m *Mock) Locations(ctx context.Context, planKey, country string) ([]Country, error) {
	if planKey == "ipv6_pergb" || planKey == "resi_unlim_budget" {
		return []Country{}, nil
	}
	return []Country{
		{Code: "US", Name: "United States", States: []State{
			{Code: "CA", Name: "California", Cities: []City{{Name: "Los Angeles"}, {Name: "San Francisco"}}},
			{Code: "NY", Name: "New York", Cities: []City{{Name: "New York"}}},
		}},
		{Code: "DE", Name: "Germany"},
		{Code: "GB", Name: "United Kingdom"},
	}, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
