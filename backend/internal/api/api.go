package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"github.com/vaultproxies/vpx/backend/internal/auth"
	"github.com/vaultproxies/vpx/backend/internal/cache"
	"github.com/vaultproxies/vpx/backend/internal/config"
	"github.com/vaultproxies/vpx/backend/internal/payments"
	"github.com/vaultproxies/vpx/backend/internal/store"
	"github.com/vaultproxies/vpx/backend/internal/vaultproxies"
)

// App bundles all dependencies shared by the HTTP handlers.
type App struct {
	Cfg    *config.Config
	Log    *slog.Logger
	Store  *store.Store
	Cache  *cache.Cache
	Auth   *auth.Manager
	Turnst *auth.Turnstile
	Google *auth.GoogleOAuth
	Vault  vaultproxies.Client
	Stripe *payments.Stripe
	Now    *payments.NowPayments
}

type ctxKey string

const (
	ctxUserID ctxKey = "uid"
	ctxRole   ctxKey = "role"
)

func userID(r *http.Request) (uuid.UUID, bool) {
	v, ok := r.Context().Value(ctxUserID).(uuid.UUID)
	return v, ok
}

func role(r *http.Request) string {
	v, _ := r.Context().Value(ctxRole).(string)
	return v
}

// Router builds the full HTTP routing tree.
func (a *App) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	// X-Forwarded-For / X-Real-IP are attacker-controlled unless a reverse proxy
	// overwrites them. Honouring them unconditionally would let anyone rotate
	// their apparent IP and bypass the per-IP auth rate limits, so this is opt-in.
	if a.Cfg.TrustProxy {
		r.Use(middleware.RealIP)
	}
	r.Use(requestLogger(a.Log))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(securityHeaders)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{a.Cfg.PublicBaseURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", a.handleHealth)
	r.Get("/readyz", a.handleReady)

	r.Route("/api/v1", func(r chi.Router) {
		// Public config for the frontend (cheap, cached, but still bounded).
		r.With(a.rateLimit("public", 60, 5)).Get("/config", a.handlePublicConfig)
		r.With(a.rateLimit("public", 60, 5)).Get("/plans", a.handleListPlans)

		// Auth (with stricter rate limits + CSRF for cookie mutations).
		r.Route("/auth", func(r chi.Router) {
			r.With(a.rateLimit("auth", 10, 1)).Post("/register", a.handleRegister)
			r.With(a.rateLimit("auth", 10, 1)).Post("/login", a.handleLogin)
			r.With(a.rateLimit("auth", 30, 1), a.csrf).Post("/refresh", a.handleRefresh)
			r.With(a.csrf).Post("/logout", a.handleLogout)
			r.With(a.rateLimit("oauth", 20, 1)).Get("/google", a.handleGoogleStart)
			r.With(a.rateLimit("oauth", 20, 1)).Get("/google/callback", a.handleGoogleCallback)
		})

		// Payment webhooks — no auth, verified by provider signature.
		r.Post("/webhooks/stripe", a.handleStripeWebhook)
		r.Post("/webhooks/nowpayments", a.handleNowPaymentsWebhook)

		// Authenticated routes.
		r.Group(func(r chi.Router) {
			r.Use(a.requireAuth)
			r.Use(a.rateLimit("api", 120, 2))

			r.Get("/me", a.handleMe)

			r.Get("/wallet", a.handleWallet)
			r.Get("/wallet/ledger", a.handleLedger)
			r.With(a.csrf).Post("/wallet/topup", a.handleTopup)
			r.Get("/payments", a.handleListPayments)

			r.Get("/orders", a.handleListOrders)
			r.With(a.csrf).Post("/orders", a.handleCreateOrder)
			r.Get("/proxies", a.handleListProxies)
			r.Get("/proxies/{id}/usage", a.handleProxyUsage)
			// Generation is free (it mints credentials, it does not sell
			// bandwidth) but each call hits the upstream, so it gets its own
			// tighter bucket.
			r.With(a.rateLimit("generate", 20, 1), a.csrf).Post("/proxies/{id}/generate", a.handleGenerateProxies)
			r.With(a.rateLimit("locations", 30, 2)).Get("/locations", a.handleLocations)

			// Admin-only.
			r.Group(func(r chi.Router) {
				r.Use(a.requireAdmin)
				r.Get("/admin/stats", a.handleAdminStats)
				r.With(a.csrf).Post("/admin/catalog/sync", a.handleSyncCatalog)
			})
		})
	})

	return r
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := a.Store.Pool().Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ready",
		"upstream": a.Vault.Healthy(ctx),
		"time":     time.Now().UTC(),
	})
}

func (a *App) handlePublicConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"site_name":          a.Cfg.AppName,
		"turnstile_site_key": a.Cfg.TurnstileSiteKey,
		"turnstile_enabled":  a.Cfg.TurnstileEnabled,
		"google_enabled":     a.Google.Enabled(),
		"stripe_enabled":     a.Stripe.Enabled(),
		"crypto_enabled":     a.Now.Enabled(),
		"min_topup_cents":    a.Cfg.MinTopupCents,
	})
}
