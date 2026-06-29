// Command server is the VPX reseller API.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vaultproxies/vpx/backend/internal/api"
	"github.com/vaultproxies/vpx/backend/internal/auth"
	"github.com/vaultproxies/vpx/backend/internal/cache"
	"github.com/vaultproxies/vpx/backend/internal/catalog"
	"github.com/vaultproxies/vpx/backend/internal/config"
	"github.com/vaultproxies/vpx/backend/internal/payments"
	"github.com/vaultproxies/vpx/backend/internal/store"
	"github.com/vaultproxies/vpx/backend/internal/vaultproxies"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	log := newLogger(cfg.LogLevel)
	slog.SetDefault(log)

	// Apply database migrations before serving.
	if err := store.Migrate(cfg.DatabaseURL); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("postgres", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	rc, err := cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Error("redis", "err", err)
		os.Exit(1)
	}
	defer rc.Close()

	// Optional admin bootstrap.
	if adminEmail := os.Getenv("ADMIN_EMAIL"); adminEmail != "" {
		if err := st.PromoteAdmin(ctx, adminEmail); err != nil {
			log.Warn("admin bootstrap failed", "err", err)
		} else {
			log.Info("admin bootstrap applied", "email", adminEmail)
		}
	}

	successURL := cfg.PublicBaseURL + "/dashboard/billing?status=success"
	cancelURL := cfg.PublicBaseURL + "/dashboard/billing?status=cancelled"
	ipnURL := cfg.APIBaseURL + "/api/v1/webhooks/nowpayments"

	app := &api.App{
		Cfg:    cfg,
		Log:    log,
		Store:  st,
		Cache:  rc,
		Auth:   auth.NewManager(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL, cfg.AppName),
		Turnst: auth.NewTurnstile(cfg.TurnstileSecretKey, cfg.TurnstileEnabled),
		Google: auth.NewGoogleOAuth(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL),
		Vault:  vaultproxies.New(cfg.VaultMode, cfg.VaultBaseURL, cfg.VaultAPIKey),
		Stripe: payments.NewStripe(cfg.StripeSecretKey, cfg.StripeWebhookSecret, successURL, cancelURL),
		Now:    payments.NewNowPayments(cfg.NowPaymentsAPIKey, cfg.NowPaymentsIPNSecret, cfg.NowPaymentsBaseURL, successURL, cancelURL, ipnURL),
	}

	// Best-effort: mirror the upstream reseller catalog into the plans table on
	// startup so retail pricing tracks VaultProxies. Non-fatal — the seeded
	// catalog remains if the upstream is unreachable.
	go func() {
		syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := catalog.Sync(syncCtx, st, app.Vault, cfg.ResellerMarkup, log); err != nil {
			log.Warn("startup catalog sync skipped", "err", err)
		} else {
			_ = rc.Del(syncCtx, "cache:plans:v1")
		}
	}()

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           app.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		log.Info("server listening", "port", cfg.HTTPPort, "env", cfg.AppEnv, "vault_mode", cfg.VaultMode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown", "err", err)
	}
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
