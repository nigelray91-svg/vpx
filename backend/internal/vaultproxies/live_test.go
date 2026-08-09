package vaultproxies

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeUpstream serves the exact JSON documented at https://vaultproxies.net/docs
// so we can verify the live client maps the real API contract correctly without
// network access to the upstream.
func fakeUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/reseller/balance", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "test-key" {
			http.Error(w, `{"error":"unauthorized"}`, 401)
			return
		}
		_, _ = w.Write([]byte(`{"balance_cents": 15000}`))
	})

	mux.HandleFunc("/api/reseller/categories", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"categories":[
			{"id":1,"key":"resi_pergb","name":"Residential Per GB","pricing_type":"pergb","price_per_unit_cts":85,"unit":"GB","min_units":1},
			{"id":2,"key":"resi_unlim","name":"Residential Unlimited","pricing_type":"unlim","price_per_unit_cts":200,"unit":"HOUR","min_units":1},
			{"id":3,"key":"dc_unlim","name":"Datacenter Unlimited","pricing_type":"unlim","price_per_unit_cts":250,"unit":"DAY","min_units":1},
			{"id":4,"key":"ipv6_pergb","name":"IPv6 Per GB","pricing_type":"pergb","price_per_unit_cts":50,"unit":"GB","min_units":1}
		]}`))
	})

	mux.HandleFunc("/api/reseller/order", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method"}`, 405)
			return
		}
		_, _ = w.Write([]byte(`{
			"service":{"id":123,"user_id":1,"category_id":1,"status":"active","pricing_type":"pergb","unit":"GB","remaining_gb":5,"username":"abc123xyz","password":"secretpass123","created_at":"2026-05-05T12:00:00Z"},
			"invoice":{"id":456,"amount_cents":425,"status":"paid"}
		}`))
	})

	mux.HandleFunc("/api/reseller/services", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"services":[
			{"id":123,"username":"abc123xyz","password":"secretpass123","remaining_gb":4.5,"status":"active","expires_at":"2026-07-06T12:00:00Z"}
		]}`))
	})

	mux.HandleFunc("/api/reseller/proxy/generator/locations", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("plan_key") == "ipv6_pergb" {
			_, _ = w.Write([]byte(`{"countries":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"countries":[
			{"code":"US","name":"United States","states":[{"code":"CA","name":"California","cities":[{"name":"Los Angeles"}]}]},
			{"code":"DE","name":"Germany"}
		]}`))
	})

	mux.HandleFunc("/api/reseller/proxy/generations/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method"}`, 405)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		// Echo back the documented shape; assert the payload keys we send.
		if _, ok := body["service_id"]; !ok {
			http.Error(w, `{"error":"service_id required"}`, 400)
			return
		}
		_, _ = w.Write([]byte(`{
			"generations":[{
				"id":1751712000000000000,"user_id":1,"service_id":123,"plan_key":"resi_pergb",
				"country":"us","protocol":"HTTP","format":"user:pass@ip:port",
				"hostname":"resi-gb.vaultproxies.com","port":80,
				"username":"abc123xyz-geo-us-sess-k2j9x1pq-life-600","password":"secretpass123",
				"output_line":"abc123xyz-geo-us-sess-k2j9x1pq-life-600:secretpass123@resi-gb.vaultproxies.com:80",
				"created_at":"2026-07-05T12:00:00Z"
			}],
			"generation":{"id":1751712000000000000}
		}`))
	})

	return httptest.NewServer(mux)
}

func newClient(t *testing.T) *Live {
	srv := fakeUpstream(t)
	t.Cleanup(srv.Close)
	return NewLive(srv.URL, "test-key", Gateways{
		"residential": "resi.gw.example:8000",
		"datacenter":  "dc.gw.example:9000",
		"ipv6":        "ipv6.gw.example:9090",
	})
}

func TestLiveCatalog(t *testing.T) {
	c := newClient(t)
	products, err := c.Catalog(context.Background())
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if len(products) != 4 {
		t.Fatalf("expected 4 categories, got %d", len(products))
	}
	byCode := map[string]Product{}
	for _, p := range products {
		byCode[p.Code] = p
	}
	if p := byCode["resi_pergb"]; p.Type != "residential" || p.Unit != "gb" || p.WholesaleCents != 85 || p.PricingType != "pergb" {
		t.Fatalf("resi_pergb mapped wrong: %+v", p)
	}
	if p := byCode["dc_unlim"]; p.Type != "datacenter" || p.Unit != "day" || p.WholesaleCents != 250 {
		t.Fatalf("dc_unlim mapped wrong: %+v", p)
	}
	if p := byCode["ipv6_pergb"]; p.Type != "ipv6" || p.WholesaleCents != 50 {
		t.Fatalf("ipv6_pergb mapped wrong: %+v", p)
	}
}

// Provisioning must take the gateway from the generator response rather than
// from configuration — the upstream docs are explicit that the generate call
// returns the correct hostname/port so none are hardcoded.
func TestLiveProvisionUsesGeneratorGateway(t *testing.T) {
	c := newClient(t)
	res, err := c.Provision(context.Background(), ProvisionRequest{
		CategoryKey: "resi_pergb", Units: 5, ProxyType: "residential",
	})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	if res.Ref != "123" {
		t.Fatalf("expected ref 123, got %s", res.Ref)
	}
	if len(res.Proxies) != 1 {
		t.Fatalf("expected 1 credential")
	}
	cr := res.Proxies[0]
	if cr.Host != "resi-gb.vaultproxies.com" || cr.Port != 80 {
		t.Fatalf("expected the generator's gateway, got %s:%d", cr.Host, cr.Port)
	}
	// The generator rewrites the username with the session grammar.
	if cr.Username != "abc123xyz-geo-us-sess-k2j9x1pq-life-600" || cr.Password != "secretpass123" {
		t.Fatalf("credentials wrong: %+v", cr)
	}
	if cr.BandwidthLimitBytes != 5_000_000_000 {
		t.Fatalf("expected 5GB limit, got %d", cr.BandwidthLimitBytes)
	}
}

// When the generator is unavailable the paid order must still yield usable
// credentials, falling back to the operator-configured gateway.
func TestLiveProvisionFallsBackWhenGeneratorFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/reseller/order", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"service":{"id":123,"status":"active","remaining_gb":5,"username":"abc123xyz","password":"secretpass123"},"invoice":{"id":456,"amount_cents":425,"status":"paid"}}`))
	})
	mux.HandleFunc("/api/reseller/proxy/generations/create", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"generator down"}`, 503)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := NewLive(srv.URL, "test-key", Gateways{"residential": "resi.gw.example:8000"})
	res, err := c.Provision(context.Background(), ProvisionRequest{
		CategoryKey: "resi_pergb", Units: 5, ProxyType: "residential",
	})
	if err != nil {
		t.Fatalf("provision must not fail when only the generator is down: %v", err)
	}
	cr := res.Proxies[0]
	if cr.Host != "resi.gw.example" || cr.Port != 8000 {
		t.Fatalf("expected configured fallback gateway, got %s:%d", cr.Host, cr.Port)
	}
	if cr.Username != "abc123xyz" {
		t.Fatalf("expected the order's credentials, got %q", cr.Username)
	}
}

func TestLiveGenerate(t *testing.T) {
	c := newClient(t)
	gens, err := c.Generate(context.Background(), GenerateRequest{
		ServiceID: 123, PlanKey: "resi_pergb", Country: "US",
		Protocol: "HTTP", Format: "user:pass@ip:port",
		Mode: RotationSticky, Count: 1, SessionSeconds: 600,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	g := gens[0]
	if g.ID != 1751712000000000000 {
		t.Fatalf("id lost precision: %d", g.ID)
	}
	if g.Hostname != "resi-gb.vaultproxies.com" || g.Port != 80 {
		t.Fatalf("gateway wrong: %s:%d", g.Hostname, g.Port)
	}
	if g.OutputLine != "abc123xyz-geo-us-sess-k2j9x1pq-life-600:secretpass123@resi-gb.vaultproxies.com:80" {
		t.Fatalf("output line wrong: %s", g.OutputLine)
	}
}

func TestLiveGenerateRequiresService(t *testing.T) {
	c := newClient(t)
	if _, err := c.Generate(context.Background(), GenerateRequest{PlanKey: "resi_pergb"}); err == nil {
		t.Fatal("expected missing service_id to be rejected before the request")
	}
}

func TestLiveLocations(t *testing.T) {
	c := newClient(t)
	countries, err := c.Locations(context.Background(), "resi_pergb", "")
	if err != nil {
		t.Fatalf("locations: %v", err)
	}
	if len(countries) != 2 || countries[0].Code != "US" {
		t.Fatalf("unexpected countries: %+v", countries)
	}
	if len(countries[0].States) != 1 || countries[0].States[0].Cities[0].Name != "Los Angeles" {
		t.Fatalf("state/city tree not parsed: %+v", countries[0])
	}
	// Plans without geo targeting return an empty list, not an error.
	empty, err := c.Locations(context.Background(), "ipv6_pergb", "")
	if err != nil || len(empty) != 0 {
		t.Fatalf("expected empty locations for ipv6_pergb, got %+v (%v)", empty, err)
	}
}

func TestFormatLine(t *testing.T) {
	cases := map[string]string{
		"ip:port:user:pass":            "h.example:80:u1:p1",
		"user:pass@ip:port":            "u1:p1@h.example:80",
		"ip:port@user:pass":            "h.example:80@u1:p1",
		"user:pass:ip:port":            "u1:p1:h.example:80",
		"ip:port:pass:user":            "h.example:80:p1:u1",
		"protocol://user:pass@ip:port": "http://u1:p1@h.example:80",
		"protocol://ip:port":           "http://h.example:80",
		"ip:port":                      "h.example:80",
		"user:pass":                    "u1:p1",
		// "host:" is accepted in place of "ip:".
		"host:port:user:pass": "h.example:80:u1:p1",
	}
	for format, want := range cases {
		if got := FormatLine(format, "h.example", 80, "u1", "p1", "HTTP"); got != want {
			t.Errorf("FormatLine(%q) = %q, want %q", format, got, want)
		}
	}
}

func TestLiveUsageAndBalance(t *testing.T) {
	c := newClient(t)
	u, err := c.Usage(context.Background(), "123")
	if err != nil {
		t.Fatalf("usage: %v", err)
	}
	if !u.Active || u.RemainingGB != 4.5 {
		t.Fatalf("usage wrong: %+v", u)
	}
	if u.ExpiresAt == nil {
		t.Fatalf("expected expires_at parsed")
	}
	bal, err := c.Balance(context.Background())
	if err != nil || bal != 15000 {
		t.Fatalf("balance wrong: %d %v", bal, err)
	}
	if !c.Healthy(context.Background()) {
		t.Fatalf("expected healthy")
	}
}
