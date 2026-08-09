package vaultproxies

import (
	"context"
	"strings"
	"testing"
)

func TestBranderHost(t *testing.T) {
	cases := []struct {
		name     string
		domain   string
		mapSpec  string
		upstream string
		want     string
	}{
		{
			name:     "domain rewrite keeps the gateway label",
			domain:   "proxies.mybrand.com",
			upstream: "resi-gb.vaultproxies.com",
			want:     "resi-gb.proxies.mybrand.com",
		},
		{
			name:     "distinct upstreams stay distinct",
			domain:   "proxies.mybrand.com",
			upstream: "eu-isp.vaultproxies.com",
			want:     "eu-isp.proxies.mybrand.com",
		},
		{
			name:     "explicit map wins over the domain rule",
			domain:   "proxies.mybrand.com",
			mapSpec:  "resi-gb.vaultproxies.com=fast.mybrand.com",
			upstream: "resi-gb.vaultproxies.com",
			want:     "fast.mybrand.com",
		},
		{
			name:     "explicit map alone",
			mapSpec:  "resi.vaultproxies.com=gw.mybrand.com",
			upstream: "resi.vaultproxies.com",
			want:     "gw.mybrand.com",
		},
		{
			name:     "matching is case-insensitive",
			mapSpec:  "resi.vaultproxies.com=gw.mybrand.com",
			upstream: "RESI.VaultProxies.com",
			want:     "gw.mybrand.com",
		},
		{
			name:     "trailing dots and spacing are tolerated",
			domain:   "  .proxies.mybrand.com. ",
			upstream: "mobile.vaultproxies.com",
			want:     "mobile.proxies.mybrand.com",
		},
		{
			name:     "unconfigured passes through",
			upstream: "resi-gb.vaultproxies.com",
			want:     "resi-gb.vaultproxies.com",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBrander(tc.domain, tc.mapSpec, false)
			got, err := b.Host(tc.upstream)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Host(%q) = %q, want %q", tc.upstream, got, tc.want)
			}
		})
	}
}

func TestBranderStrictRefusesToLeak(t *testing.T) {
	b := NewBrander("", "", true)
	if _, err := b.Host("resi-gb.vaultproxies.com"); err == nil {
		t.Fatal("strict mode must refuse to emit an unbranded hostname")
	}
}

// The whole point: nothing that reaches a customer may contain the upstream
// domain — including the pre-rendered output line.
func TestGenerateNeverLeaksUpstreamHostname(t *testing.T) {
	srv := fakeUpstream(t)
	c := NewLiveBranded(srv.URL, "test-key", nil,
		NewBrander("proxies.mybrand.com", "", false))

	gens, err := c.Generate(context.Background(), GenerateRequest{
		ServiceID: 123, PlanKey: "resi_pergb", Count: 1,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	g := gens[0]
	if g.Hostname != "resi-gb.proxies.mybrand.com" {
		t.Fatalf("hostname not branded: %q", g.Hostname)
	}
	if !strings.Contains(g.OutputLine, "resi-gb.proxies.mybrand.com") {
		t.Fatalf("output line missing branded host: %q", g.OutputLine)
	}
	// The credentials must survive the host swap untouched.
	if !strings.Contains(g.OutputLine, "abc123xyz-geo-us-sess-k2j9x1pq-life-600") ||
		!strings.Contains(g.OutputLine, "secretpass123") {
		t.Fatalf("credentials mangled by branding: %q", g.OutputLine)
	}
	for _, field := range []string{g.Hostname, g.OutputLine, g.Username, g.Password} {
		if strings.Contains(strings.ToLower(field), "vaultproxies") {
			t.Fatalf("upstream provider leaked to the customer in %q", field)
		}
	}
}

func TestProvisionBrandsTheGateway(t *testing.T) {
	srv := fakeUpstream(t)
	c := NewLiveBranded(srv.URL, "test-key", nil,
		NewBrander("proxies.mybrand.com", "", false))

	res, err := c.Provision(context.Background(), ProvisionRequest{
		CategoryKey: "resi_pergb", Units: 5, ProxyType: "residential",
	})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	if host := res.Proxies[0].Host; host != "resi-gb.proxies.mybrand.com" {
		t.Fatalf("provisioned host not branded: %q", host)
	}
}

func TestMockBrandsTheGateway(t *testing.T) {
	m := NewMockBranded(NewBrander("proxies.mybrand.com", "", false))
	gens, err := m.Generate(context.Background(), GenerateRequest{
		ServiceID: 1, PlanKey: "resi_pergb", Count: 2, Mode: RotationSticky,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, g := range gens {
		if strings.Contains(g.OutputLine, "vaultproxies") {
			t.Fatalf("mock leaked upstream host: %q", g.OutputLine)
		}
		if !strings.Contains(g.OutputLine, "proxies.mybrand.com") {
			t.Fatalf("mock did not brand the host: %q", g.OutputLine)
		}
	}
}

// A per-service gateway (e.g. resi_unlim_budget) has no pre-created CNAME, so
// the operator must be told rather than the customer getting a dead endpoint.
func TestUnknownGatewayLabelStillBrands(t *testing.T) {
	b := NewBrander("proxies.nullvault.shop", "", false)
	got, err := b.Host("svc-8821.vaultproxies.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "svc-8821.proxies.nullvault.shop" {
		t.Fatalf("got %q", got)
	}
	// Known labels must not warn.
	if _, seen := warnedLabels.Load("resi-gb"); seen {
		t.Fatal("known label should not be flagged")
	}
}
