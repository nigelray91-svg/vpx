package api

import (
	"net/http"

	"github.com/vaultproxies/vpx/backend/internal/catalog"
)

func (a *App) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := a.Store.AdminStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// handleSyncCatalog pulls the upstream reseller catalog into the plans table
// (retail = wholesale * markup) and invalidates the public plans cache.
func (a *App) handleSyncCatalog(w http.ResponseWriter, r *http.Request) {
	res, err := catalog.Sync(r.Context(), a.Store, a.Vault, a.Cfg.ResellerMarkup, a.Log)
	if err != nil {
		writeError(w, http.StatusBadGateway, "catalog sync failed: "+err.Error())
		return
	}
	_ = a.Cache.Del(r.Context(), plansCacheKey)
	writeJSON(w, http.StatusOK, res)
}
