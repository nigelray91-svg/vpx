package api

import "net/http"

func (a *App) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := a.Store.AdminStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
