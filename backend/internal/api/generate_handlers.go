package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/vaultproxies/vpx/backend/internal/vaultproxies"
)

// sessionCaps are the documented per-plan sticky-session limits, in seconds.
// Requests are clamped to these rather than forwarded blindly, so a customer
// gets working credentials instead of an upstream 400.
var sessionCaps = map[string]struct{ Min, Max int }{
	"resi_pergb":        {0, 21600},  // 6 h
	"shared_isp":        {0, 86400},  // 24 h
	"resi_unlim_budget": {0, 86400},  // 24 h
	"mobile_pergb":      {60, 7200},  // whole minutes, max 2 h
	"resi_unlim":        {60, 86400}, // converted to whole minutes upstream
}

const defaultSessionSeconds = 600

func clampSession(planKey string, seconds int) int {
	if seconds <= 0 {
		seconds = defaultSessionSeconds
	}
	cap, ok := sessionCaps[planKey]
	if !ok {
		return seconds
	}
	if cap.Min > 0 && seconds < cap.Min {
		seconds = cap.Min
	}
	if cap.Max > 0 && seconds > cap.Max {
		seconds = cap.Max
	}
	return seconds
}

type generateReq struct {
	Protocol       string   `json:"protocol" validate:"omitempty,oneof=HTTP SOCKS5"`
	Format         string   `json:"format" validate:"max=40"`
	Country        string   `json:"country" validate:"max=60"`
	Continent      string   `json:"continent" validate:"omitempty,oneof=EU NA SA AS AF OC"`
	State          string   `json:"state" validate:"max=60"`
	City           string   `json:"city" validate:"max=60"`
	IPs            []string `json:"ips" validate:"max=1000,dive,ip"`
	Mode           string   `json:"mode" validate:"omitempty,oneof=rotating sticky"`
	Count          int      `json:"count" validate:"min=0,max=10000"`
	SessionSeconds int      `json:"session_seconds" validate:"min=0,max=86400"`
	SessionLength  string   `json:"session_length" validate:"omitempty,oneof=long short"`
}

// handleGenerateProxies turns one of the caller's own services into ready-to-use
// proxy lines. Generation is free — it mints credentials rather than selling
// bandwidth — so it may be repeated as often as the customer likes.
func (a *App) handleGenerateProxies(w http.ResponseWriter, r *http.Request) {
	uid, _ := userID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid proxy id")
		return
	}
	var req generateReq
	if !decode(w, r, &req) {
		return
	}
	if req.Format != "" && !vaultproxies.ValidFormat(req.Format) {
		writeError(w, http.StatusBadRequest, "unsupported output format")
		return
	}

	// Ownership: both lookups are scoped to the caller's user id.
	proxy, err := a.Store.GetProxy(r.Context(), id, uid)
	if err != nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return
	}
	order, err := a.Store.GetOrder(r.Context(), proxy.OrderID, uid)
	if err != nil || order.VaultRef == nil || *order.VaultRef == "" {
		writeError(w, http.StatusConflict, "this order has no upstream service yet")
		return
	}
	serviceID, err := strconv.Atoi(*order.VaultRef)
	if err != nil {
		writeError(w, http.StatusConflict, "this order has no upstream service yet")
		return
	}
	plan, err := a.Store.GetPlanByID(r.Context(), order.PlanID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load plan")
		return
	}

	count := req.Count
	if count < 1 {
		count = 1
	}
	mode := req.Mode
	if mode == "" {
		mode = vaultproxies.RotationRotating
	}
	gr := vaultproxies.GenerateRequest{
		ServiceID: serviceID,
		PlanKey:   plan.Code,
		Protocol:  req.Protocol,
		Format:    req.Format,
		Country:   req.Country,
		Continent: req.Continent,
		State:     req.State,
		City:      req.City,
		IPs:       req.IPs,
		Mode:      mode,
		Count:     count,
	}
	if mode == vaultproxies.RotationSticky {
		gr.SessionSeconds = clampSession(plan.Code, req.SessionSeconds)
		gr.SessionLength = req.SessionLength
	}

	gens, err := a.Vault.Generate(r.Context(), gr)
	if err != nil {
		a.Log.Error("generate proxies", "err", err, "user", uid, "service", serviceID)
		writeErrorCode(w, http.StatusBadGateway, "could not generate proxies", "generate_failed")
		return
	}

	lines := make([]string, 0, len(gens))
	for _, g := range gens {
		lines = append(lines, g.OutputLine)
	}

	resp := map[string]any{
		"generations": gens,
		"lines":       lines,
		"mode":        mode,
	}
	// Rotating credentials carry no session token, so every line is byte-identical
	// and a fresh exit IP is issued per request. Say so rather than letting it
	// look like a bug.
	if mode == vaultproxies.RotationRotating && count > 1 {
		resp["note"] = "Rotating credentials rotate per request, so all lines are identical by design. " +
			"Use sticky mode if you need distinct concurrent session IPs."
	}
	if mode == vaultproxies.RotationSticky {
		resp["session_seconds"] = gr.SessionSeconds
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleLocations lists the geo targets a plan supports. The upstream catalog
// moves slowly, so responses are cached in Redis.
func (a *App) handleLocations(w http.ResponseWriter, r *http.Request) {
	planKey := strings.TrimSpace(r.URL.Query().Get("plan_key"))
	country := strings.TrimSpace(r.URL.Query().Get("country"))
	if planKey == "" {
		planKey = "resi_unlim"
	}
	// Only advertise geo targets for plans we actually sell, so this cannot be
	// used to probe arbitrary upstream keys.
	if !a.isKnownPlanCode(r, planKey) {
		writeError(w, http.StatusBadRequest, "unknown plan")
		return
	}
	if len(country) > 60 {
		writeError(w, http.StatusBadRequest, "invalid country")
		return
	}

	cacheKey := "cache:locations:v1:" + planKey + ":" + strings.ToLower(country)
	if cached, ok, err := a.Cache.Get(r.Context(), cacheKey); err == nil && ok {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(cached))
		return
	}

	countries, err := a.Vault.Locations(r.Context(), planKey, country)
	if err != nil {
		a.Log.Warn("locations", "err", err, "plan", planKey)
		writeErrorCode(w, http.StatusBadGateway, "could not load locations", "locations_failed")
		return
	}
	if countries == nil {
		countries = []vaultproxies.Country{}
	}
	body, err := json.Marshal(map[string]any{"countries": countries})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not encode locations")
		return
	}
	if err := a.Cache.Set(r.Context(), cacheKey, string(body), 6*time.Hour); err != nil {
		a.Log.Warn("cache locations", "err", err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(body)
}

// isKnownPlanCode reports whether planKey matches an active plan we sell.
func (a *App) isKnownPlanCode(r *http.Request, planKey string) bool {
	plans, err := a.Store.ListActivePlans(r.Context())
	if err != nil {
		return false
	}
	for _, p := range plans {
		if p.Code == planKey {
			return true
		}
	}
	return false
}
