package vaultproxies

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// endpoints centralizes every upstream path. Adjust these (and the request/
// response field tags below) to match https://vaultproxies.net/docs exactly.
var endpoints = struct {
	Provision string
	Usage     string // %s = ref
	Revoke    string // %s = ref
	Health    string
}{
	Provision: "/api/v1/reseller/orders",
	Usage:     "/api/v1/reseller/orders/%s/usage",
	Revoke:    "/api/v1/reseller/orders/%s",
	Health:    "/api/v1/reseller/ping",
}

type Live struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewLive(baseURL, apiKey string) *Live {
	return &Live{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 20 * time.Second},
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
	// Authentication: VaultProxies reseller API key. Confirm header name with /docs.
	req.Header.Set("Authorization", "Bearer "+l.apiKey)
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
		return resp.StatusCode, fmt.Errorf("vaultproxies %s %s: %d %s", method, path, resp.StatusCode, string(data))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

// wire structs — map these to the real upstream schema.
type provisionPayload struct {
	Type      string `json:"type"`
	Unit      string `json:"unit"`
	Quantity  int    `json:"quantity"`
	Rotation  string `json:"rotation"`
	StickyTTL int    `json:"sticky_ttl_seconds,omitempty"`
	Pool      string `json:"pool,omitempty"`
	Region    string `json:"region,omitempty"`
	Label     string `json:"label,omitempty"`
}

type provisionResponse struct {
	ID        string            `json:"id"`
	Reference string            `json:"reference"`
	ExpiresAt *time.Time        `json:"expires_at"`
	Proxies   []ProxyCredential `json:"proxies"`
	// Some APIs return a single endpoint object instead of a list:
	Endpoint *ProxyCredential `json:"endpoint"`
}

func (l *Live) Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
	payload := provisionPayload{
		Type:      req.ProxyType,
		Unit:      req.Unit,
		Quantity:  req.Quantity,
		Rotation:  req.Rotation,
		StickyTTL: req.StickyTTLSec,
		Pool:      req.Pool,
		Region:    req.Region,
		Label:     req.Label,
	}
	var out provisionResponse
	if _, err := l.do(ctx, http.MethodPost, endpoints.Provision, payload, &out); err != nil {
		return nil, err
	}
	ref := out.Reference
	if ref == "" {
		ref = out.ID
	}
	proxies := out.Proxies
	if len(proxies) == 0 && out.Endpoint != nil {
		proxies = []ProxyCredential{*out.Endpoint}
	}
	if ref == "" || len(proxies) == 0 {
		return nil, fmt.Errorf("vaultproxies: provision returned no usable credentials")
	}
	return &ProvisionResult{Ref: ref, Proxies: proxies, ExpiresAt: out.ExpiresAt}, nil
}

type usageResponse struct {
	BandwidthUsedBytes int64 `json:"bandwidth_used_bytes"`
	BandwidthCapBytes  int64 `json:"bandwidth_cap_bytes"`
	Active             bool  `json:"active"`
}

func (l *Live) Usage(ctx context.Context, ref string) (*Usage, error) {
	var out usageResponse
	if _, err := l.do(ctx, http.MethodGet, fmt.Sprintf(endpoints.Usage, ref), nil, &out); err != nil {
		return nil, err
	}
	return &Usage{
		Ref:                ref,
		BandwidthUsedBytes: out.BandwidthUsedBytes,
		BandwidthCapBytes:  out.BandwidthCapBytes,
		Active:             out.Active,
	}, nil
}

func (l *Live) Revoke(ctx context.Context, ref string) error {
	_, err := l.do(ctx, http.MethodDelete, fmt.Sprintf(endpoints.Revoke, ref), nil, nil)
	return err
}

func (l *Live) Healthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	code, err := l.do(ctx, http.MethodGet, endpoints.Health, nil, nil)
	return err == nil && code < 400
}
