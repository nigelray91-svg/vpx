package api

import (
	"net/http"
	"time"
)

// plansCacheKey is the Redis key for the cached public plan list.
const plansCacheKey = "cache:plans:v1"

// handleListPlans returns the public resale catalog (retail prices only).
func (a *App) handleListPlans(w http.ResponseWriter, r *http.Request) {
	const cacheKey = plansCacheKey
	if v, ok, _ := a.Cache.Get(r.Context(), cacheKey); ok {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Cache", "HIT")
		_, _ = w.Write([]byte(v))
		return
	}
	plans, err := a.Store.ListActivePlans(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load plans")
		return
	}
	type planView struct {
		ID            string `json:"id"`
		Code          string `json:"code"`
		Name          string `json:"name"`
		ProxyType     string `json:"proxy_type"`
		Unit          string `json:"unit"`
		PriceCents    int64  `json:"price_cents"`
		MinQuantity   int    `json:"min_quantity"`
	}
	views := make([]planView, 0, len(plans))
	for _, p := range plans {
		views = append(views, planView{
			ID: p.ID.String(), Code: p.Code, Name: p.Name, ProxyType: p.ProxyType,
			Unit: p.Unit, PriceCents: p.RetailCentsUnit, MinQuantity: p.MinQuantity,
		})
	}
	// Cache the serialized payload for a short window.
	body := mustJSON(map[string]any{"plans": views})
	_ = a.Cache.Set(r.Context(), cacheKey, body, 60*time.Second)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Cache", "MISS")
	_, _ = w.Write([]byte(body))
}
