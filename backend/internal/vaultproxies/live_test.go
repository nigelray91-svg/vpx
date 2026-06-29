package vaultproxies

import (
	"context"
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

func TestLiveProvisionAndGateway(t *testing.T) {
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
	if cr.Username != "abc123xyz" || cr.Password != "secretpass123" {
		t.Fatalf("credentials wrong: %+v", cr)
	}
	if cr.Host != "resi.gw.example" || cr.Port != 8000 {
		t.Fatalf("gateway not applied: %s:%d", cr.Host, cr.Port)
	}
	if cr.BandwidthLimitBytes != 5_000_000_000 {
		t.Fatalf("expected 5GB limit, got %d", cr.BandwidthLimitBytes)
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
