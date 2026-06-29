package vaultproxies

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// endpoints are the four real VaultProxies reseller API paths.
// Reference: https://vaultproxies.net/docs
var endpoints = struct {
	Balance    string
	Categories string
	Order      string
	Services   string
}{
	Balance:    "/api/reseller/balance",
	Categories: "/api/reseller/categories",
	Order:      "/api/reseller/order",
	Services:   "/api/reseller/services",
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
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
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

// typeFromKey derives a proxy type from the category key.
func typeFromKey(key string) string {
	switch {
	case strings.HasPrefix(key, "resi"):
		return "residential"
	case strings.HasPrefix(key, "ipv6"):
		return "ipv6"
	case strings.HasPrefix(key, "dc"):
		return "datacenter"
	default:
		return "datacenter"
	}
}

// ---- Catalog (GET /api/reseller/categories) ----

type category struct {
	ID            int    `json:"id"`
	Key           string `json:"key"`
	Name          string `json:"name"`
	PricingType   string `json:"pricing_type"`
	PricePerUnit  int64  `json:"price_per_unit_cts"`
	Unit          string `json:"unit"`
	MinUnits      int    `json:"min_units"`
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
	host, port := l.gateway(req.ProxyType)

	var limitBytes int64
	if out.Service.RemainingGB > 0 {
		limitBytes = int64(out.Service.RemainingGB*1e9 + 0.5)
	}

	cred := ProxyCredential{
		Protocol:            "http",
		Host:                host,
		Port:                port,
		Username:            out.Service.Username,
		Password:            out.Service.Password,
		Pool:                req.CategoryKey,
		Rotation:            "rotating", // sticky is controlled via the username
		BandwidthLimitBytes: limitBytes,
	}
	return &ProvisionResult{Ref: ref, Proxies: []ProxyCredential{cred}}, nil
}

// gateway returns the host/port for a proxy type from configuration.
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
