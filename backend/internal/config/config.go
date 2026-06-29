// Package config loads and validates runtime configuration from the
// environment. All secrets come from the environment — nothing is hardcoded.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv        string
	AppName       string
	PublicBaseURL string
	APIBaseURL    string
	HTTPPort      string
	LogLevel      string

	DatabaseURL string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSecret     []byte
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
	CookieDomain  string
	CookieSecure  bool

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	TurnstileSiteKey   string
	TurnstileSecretKey string
	TurnstileEnabled   bool

	StripeSecretKey      string
	StripeWebhookSecret  string
	StripePublishableKey string

	NowPaymentsAPIKey    string
	NowPaymentsIPNSecret string
	NowPaymentsBaseURL   string

	VaultBaseURL string
	VaultAPIKey  string
	VaultMode    string // live | mock

	ResellerMarkup float64
	MinTopupCents  int64
}

// Load reads configuration from the environment and validates required fields.
func Load() (*Config, error) {
	c := &Config{
		AppEnv:        getEnv("APP_ENV", "development"),
		AppName:       getEnv("APP_NAME", "Proxia"),
		PublicBaseURL: getEnv("PUBLIC_BASE_URL", "http://localhost:3000"),
		APIBaseURL:    getEnv("API_BASE_URL", "http://localhost:8080"),
		HTTPPort:      getEnv("HTTP_PORT", "8080"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		CookieDomain: getEnv("COOKIE_DOMAIN", ""),
		CookieSecure: getEnvBool("COOKIE_SECURE", true),

		GoogleClientID:     getEnv("GOOGLE_OAUTH_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_OAUTH_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_OAUTH_REDIRECT_URL", ""),

		TurnstileSiteKey:   getEnv("TURNSTILE_SITE_KEY", ""),
		TurnstileSecretKey: getEnv("TURNSTILE_SECRET_KEY", ""),
		TurnstileEnabled:   getEnvBool("TURNSTILE_ENABLED", true),

		StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),

		NowPaymentsAPIKey:    getEnv("NOWPAYMENTS_API_KEY", ""),
		NowPaymentsIPNSecret: getEnv("NOWPAYMENTS_IPN_SECRET", ""),
		NowPaymentsBaseURL:   getEnv("NOWPAYMENTS_BASE_URL", "https://api.nowpayments.io"),

		VaultBaseURL: getEnv("VAULTPROXIES_BASE_URL", "https://vaultproxies.net"),
		VaultAPIKey:  getEnv("VAULTPROXIES_API_KEY", ""),
		VaultMode:    getEnv("VAULTPROXIES_MODE", "live"),

		ResellerMarkup: getEnvFloat("RESELLER_MARKUP", 1.40),
		MinTopupCents:  int64(getEnvInt("MIN_TOPUP_CENTS", 500)),
	}

	secret := getEnv("JWT_SECRET", "")
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be set and at least 32 characters")
	}
	c.JWTSecret = []byte(secret)

	var err error
	if c.JWTAccessTTL, err = time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m")); err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}
	if c.JWTRefreshTTL, err = time.ParseDuration(getEnv("JWT_REFRESH_TTL", "720h")); err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}

	c.DatabaseURL = getEnv("DATABASE_URL", "")
	if c.DatabaseURL == "" {
		c.DatabaseURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			getEnv("POSTGRES_USER", "vpx"),
			getEnv("POSTGRES_PASSWORD", "vpx"),
			getEnv("POSTGRES_HOST", "localhost"),
			getEnv("POSTGRES_PORT", "5432"),
			getEnv("POSTGRES_DB", "vpx"),
			getEnv("POSTGRES_SSLMODE", "disable"),
		)
	}

	if c.ResellerMarkup < 1.0 {
		return nil, fmt.Errorf("RESELLER_MARKUP must be >= 1.0")
	}

	return c, nil
}

func (c *Config) IsProduction() bool { return strings.EqualFold(c.AppEnv, "production") }

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
