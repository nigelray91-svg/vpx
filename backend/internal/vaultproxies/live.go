package vaultproxies

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// endpoints are the six real VaultProxies reseller API paths.
// Reference: https://vaultproxies.net/docs
var endpoints = struct {
	Balance    string
	Categories string
	Order      string
	Services   string
	Locations  string
	Generate   string
}{
	Balance:    "/api/reseller/balance",
	Categories: "/api/reseller/categories",
	Order:      "/api/reseller/order",
	Services:   "/api/reseller/services",
	Locations:  "/api/reseller/proxy/generator/locations",
	Generate:   "/api/reseller/proxy/generations/create",
}

type Live struct {
	baseURL  string
	apiKey   string
	gateways Gateways
	http     *http.Client
}

func NewLive(baseURL, apiKey string, gateways Gateways) *Live {
	if gateways == nil {
		gateways = Gateways{}
	}
	return &Live{
		baseURL:  strings.TrimRight(baseURL, "/"),
		apiKey:   apiKey,
		gateways: gateways,
		http:     &http.Client{Timeout: 20 * time.Second},
	}
}

func (l *Live) logf(format string, args ...any) {
	slog.Warn("vaultproxies: " + fmt.Sprintf(format, args...))
}

func (l *Live) do(ctx context.Context, method, path string, body any, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, l.baseURL+path, rdr)
	if err != nil {
		return 0, err
	}
	// VaultProxies reseller API: single X-API-Key header.
	req.Header.Set("X-API-Key", l.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := l.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	// Generous cap: a 10,000-line generation is several MB, and truncating it
	// would surface as an opaque JSON decode error.
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if resp.StatusCode >= 400 {
		// Errors are { "error": "..." }.
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &e)
		msg := e.Error
		if msg == "" {
			msg = string(data)
		}
		return resp.StatusCode, fmt.Errorf("vaultproxies %s %s: %d %s", method, path, resp.StatusCode, msg)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

// typeFromKey derives a proxy type from the category key. The documented plan
// keys are resi_pergb, resi_unlim, resi_unlim_budget, dc_unlim, dc_pergb,
// mobile_pergb, backup_pergb, ipv6_pergb, shared_isp and eu_isp.
func typeFromKey(key string) string {
	switch {
	case strings.HasPrefix(key, "resi"):
		return "residential"
	case strings.HasPrefix(key, "ipv6"):
		return "ipv6"
	case strings.HasPrefix(key, "dc"):
		return "datacenter"
	case strings.HasPrefix(key, "mobile"):
		return "mobile"
	case strings.HasSuffix(key, "isp"):
		return "isp"
	case strings.HasPrefix(key, "backup"):
		return "residential"
	default:
		return "datacenter"
	}
}

// ---- Catalog (GET /api/reseller/categories) ----

type category struct {
	ID           int    `json:"id"`
	Key          string `json:"key"`
	Name         string `json:"name"`
	PricingType  string `json:"pricing_type"`
	PricePerUnit int64  `json:"price_per_unit_cts"`
	Unit         string `json:"unit"`
	MinUnits     int    `json:"min_units"`
}

type categoriesResponse struct {
	Categories []category `json:"categories"`
}

func (l *Live) Catalog(ctx context.Context) ([]Product, error) {
	var out categoriesResponse
	if _, err := l.do(ctx, http.MethodGet, endpoints.Categories, nil, &out); err != nil {
		return nil, err
	}
	products := make([]Product, 0, len(out.Categories))
	for _, c := range out.Categories {
		if c.Key == "" {
			continue
		}
		min := c.MinUnits
		if min < 1 {
			min = 1
		}
		products = append(products, Product{
			Code:           c.Key,
			Name:           c.Name,
			Type:           typeFromKey(c.Key),
			Unit:           strings.ToLower(c.Unit),
			PricingType:    c.PricingType,
			WholesaleCents: c.PricePerUnit,
			MinQuantity:    min,
		})
	}
	return products, nil
}

// ---- Order (POST /api/reseller/order) ----

type orderPayload struct {
	CategoryKey string `json:"category_key"`
	Units       int    `json:"units"`
	TimeUnit    string `json:"time_unit,omitempty"`
}

type orderService struct {
	ID          int     `json:"id"`
	Status      string  `json:"status"`
	PricingType string  `json:"pricing_type"`
	Unit        string  `json:"unit"`
	RemainingGB float64 `json:"remaining_gb"`
	Username    string  `json:"username"`
	Password    string  `json:"password"`
	CreatedAt   string  `json:"created_at"`
}

type orderResponse struct {
	Service orderService `json:"service"`
	Invoice struct {
		ID          int    `json:"id"`
		AmountCents int64  `json:"amount_cents"`
		Status      string `json:"status"`
	} `json:"invoice"`
}

func (l *Live) Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
	payload := orderPayload{
		CategoryKey: req.CategoryKey,
		Units:       req.Units,
		TimeUnit:    req.TimeUnit,
	}
	var out orderResponse
	if _, err := l.do(ctx, http.MethodPost, endpoints.Order, payload, &out); err != nil {
		return nil, err
	}
	if out.Service.Username == "" || out.Service.Password == "" {
		return nil, fmt.Errorf("vaultproxies: order returned no credentials")
	}
	ref := strconv.Itoa(out.Service.ID)

	var limitBytes int64
	if out.Service.RemainingGB > 0 {
		limitBytes = int64(out.Service.RemainingGB*1e9 + 0.5)
	}

	cred := ProxyCredential{
		Protocol:            "http",
		Username:            out.Service.Username,
		Password:            out.Service.Password,
		Pool:                req.CategoryKey,
		Rotation:            RotationRotating,
		BandwidthLimitBytes: limitBytes,
	}

	// The order response carries credentials but no gateway. Ask the generator
	// for a ready-to-use line: it returns the authoritative hostname/port for
	// this plan, which is why no gateway is hardcoded here.
	gens, gerr := l.Generate(ctx, GenerateRequest{
		ServiceID: out.Service.ID,
		PlanKey:   req.CategoryKey,
		Protocol:  "HTTP",
		Mode:      RotationRotating,
		Count:     1,
	})
	if gerr == nil && len(gens) > 0 {
		g := gens[0]
		cred.Host, cred.Port = g.Hostname, g.Port
		// The generator rewrites the username to encode geo/session grammar.
		if g.Username != "" {
			cred.Username = g.Username
		}
		if g.Password != "" {
			cred.Password = g.Password
		}
	} else {
		// Fall back to an operator-configured gateway so a generator outage
		// still yields usable credentials rather than failing the paid order.
		if gerr != nil {
			l.logf("generate after order failed, falling back to configured gateway: %v", gerr)
		}
		cred.Host, cred.Port = l.gateway(req.ProxyType)
	}

	return &ProvisionResult{Ref: ref, Proxies: []ProxyCredential{cred}}, nil
}

// gateway returns the host/port for a proxy type from optional configuration.
// It is only a fallback: the generator endpoint is authoritative.
func (l *Live) gateway(proxyType string) (string, int) {
	hp := l.gateways[proxyType]
	if hp == "" {
		hp = l.gateways["default"]
	}
	host, portStr, ok := strings.Cut(hp, ":")
	if !ok {
		// Unconfigured gateway — surface a clearly-invalid placeholder so it is
		// obvious the operator must set VAULT_GATEWAY_* from the dashboard.
		return "configure-gateway." + proxyType + ".invalid", 0
	}
	port, _ := strconv.Atoi(portStr)
	return host, port
}

// ---- Generator (POST /api/reseller/proxy/generations/create) ----

type generateExtra struct {
	Mode           string `json:"mode,omitempty"`
	Count          int    `json:"count,omitempty"`
	SessionSeconds int    `json:"session_seconds,omitempty"`
	SessionMinutes int    `json:"session_minutes,omitempty"`
	SessionLength  string `json:"session_length,omitempty"`
}

type generatePayload struct {
	ServiceID int           `json:"service_id"`
	PlanKey   string        `json:"plan_key"`
	Protocol  string        `json:"protocol,omitempty"`
	Format    string        `json:"format,omitempty"`
	Country   string        `json:"country,omitempty"`
	Continent string        `json:"continent,omitempty"`
	State     string        `json:"state,omitempty"`
	City      string        `json:"city,omitempty"`
	IPs       []string      `json:"ips,omitempty"`
	Extra     generateExtra `json:"extra,omitempty"`
}

type generateResponse struct {
	Generations []Generation `json:"generations"`
}

// Generate turns an active service into ready-to-use proxy lines. The response
// carries the correct hostname and port for the plan, so callers never hardcode
// a gateway.
func (l *Live) Generate(ctx context.Context, req GenerateRequest) ([]Generation, error) {
	if req.ServiceID == 0 || req.PlanKey == "" {
		return nil, fmt.Errorf("vaultproxies: service_id and plan_key are required")
	}
	payload := generatePayload{
		ServiceID: req.ServiceID,
		PlanKey:   req.PlanKey,
		Protocol:  req.Protocol,
		Format:    req.Format,
		Country:   req.Country,
		Continent: req.Continent,
		State:     req.State,
		City:      req.City,
		IPs:       req.IPs,
		Extra: generateExtra{
			Mode:           req.Mode,
			Count:          req.Count,
			SessionSeconds: req.SessionSeconds,
			SessionLength:  req.SessionLength,
		},
	}
	var out generateResponse
	if _, err := l.do(ctx, http.MethodPost, endpoints.Generate, payload, &out); err != nil {
		return nil, err
	}
	if len(out.Generations) == 0 {
		return nil, fmt.Errorf("vaultproxies: generator returned no proxies")
	}
	return out.Generations, nil
}

// ---- Generator locations (GET /api/reseller/proxy/generator/locations) ----

type locationsResponse struct {
	Countries []Country `json:"countries"`
}

// Locations lists the geo targets available for a plan. Plans without geo
// targeting return an empty list.
func (l *Live) Locations(ctx context.Context, planKey, country string) ([]Country, error) {
	q := url.Values{}
	if planKey != "" {
		q.Set("plan_key", planKey)
	}
	if country != "" {
		q.Set("country", country)
	}
	path := endpoints.Locations
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out locationsResponse
	if _, err := l.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Countries, nil
}

// ---- Usage (GET /api/reseller/services) ----

type servicesResponse struct {
	Services []struct {
		ID          int     `json:"id"`
		Username    string  `json:"username"`
		Password    string  `json:"password"`
		RemainingGB float64 `json:"remaining_gb"`
		Status      string  `json:"status"`
		ExpiresAt   *string `json:"expires_at"`
	} `json:"services"`
}

func (l *Live) Usage(ctx context.Context, ref string) (*Usage, error) {
	var out servicesResponse
	if _, err := l.do(ctx, http.MethodGet, endpoints.Services, nil, &out); err != nil {
		return nil, err
	}
	for _, s := range out.Services {
		if strconv.Itoa(s.ID) == ref {
			u := &Usage{Ref: ref, RemainingGB: s.RemainingGB, Active: s.Status == "active"}
			if s.ExpiresAt != nil {
				if t, err := time.Parse(time.RFC3339, *s.ExpiresAt); err == nil {
					u.ExpiresAt = &t
				}
			}
			return u, nil
		}
	}
	return &Usage{Ref: ref, Active: false}, nil
}

// Revoke is not supported by the reseller API (services expire on their own).
func (l *Live) Revoke(ctx context.Context, ref string) error { return nil }

// Healthy probes the balance endpoint.
func (l *Live) Healthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	code, err := l.do(ctx, http.MethodGet, endpoints.Balance, nil, nil)
	return err == nil && code < 400
}

// Balance returns the reseller wallet balance in cents (GET /api/reseller/balance).
func (l *Live) Balance(ctx context.Context) (int64, error) {
	var out struct {
		BalanceCents int64 `json:"balance_cents"`
	}
	if _, err := l.do(ctx, http.MethodGet, endpoints.Balance, nil, &out); err != nil {
		return 0, err
	}
	return out.BalanceCents, nil
}
