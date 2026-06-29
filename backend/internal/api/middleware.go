package api

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

const (
	cookieAccess  = "vpx_access"
	cookieRefresh = "vpx_refresh"
	cookieCSRF    = "vpx_csrf"
	refreshPath   = "/api/v1/auth"
)

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Info("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"dur_ms", time.Since(start).Milliseconds(),
				"req_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}

// securityHeaders applies hardening headers to every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// API serves JSON only; lock down with a tight CSP.
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// requireAuth validates the access token from the cookie or Bearer header.
func (a *App) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerOrCookie(r)
		if token == "" {
			writeErrorCode(w, http.StatusUnauthorized, "authentication required", "unauthenticated")
			return
		}
		claims, err := a.Auth.ParseAccessToken(token)
		if err != nil {
			writeErrorCode(w, http.StatusUnauthorized, "invalid or expired token", "unauthenticated")
			return
		}
		uid, err := uuid.Parse(claims.Subject)
		if err != nil {
			writeErrorCode(w, http.StatusUnauthorized, "invalid token subject", "unauthenticated")
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, uid)
		ctx = context.WithValue(ctx, ctxRole, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if role(r) != "admin" {
			writeErrorCode(w, http.StatusForbidden, "admin access required", "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerOrCookie(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	if c, err := r.Cookie(cookieAccess); err == nil {
		return c.Value
	}
	return ""
}

// csrf enforces double-submit CSRF protection for cookie-authenticated,
// state-changing requests. Requests authenticated purely via a Bearer header
// (non-browser clients) are exempt because browsers never attach custom
// Authorization headers cross-site automatically.
func (a *App) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bearer-only API clients are not subject to CSRF.
		if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(cookieCSRF)
		header := r.Header.Get("X-CSRF-Token")
		if err != nil || cookie.Value == "" || header == "" ||
			subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
			writeErrorCode(w, http.StatusForbidden, "invalid CSRF token", "csrf")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// rateLimit applies a per-identity token-bucket limit backed by Redis.
// capacity = burst, refill = sustained requests/second.
func (a *App) rateLimit(scope string, capacity, refillPerSec int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := clientIP(r)
			if uid, ok := userID(r); ok {
				id = uid.String()
			}
			key := "rl:" + scope + ":" + id
			res, err := a.Cache.Allow(r.Context(), key, capacity, refillPerSec)
			if err != nil {
				// Fail open on cache errors but log — availability over strictness.
				a.Log.Warn("ratelimit error", "err", err)
				next.ServeHTTP(w, r)
				return
			}
			if !res.Allowed {
				w.Header().Set("Retry-After", itoa(res.RetryAfter))
				writeErrorCode(w, http.StatusTooManyRequests, "rate limit exceeded", "rate_limited")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func itoa(n int) string {
	if n <= 0 {
		return "1"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

// ---- Cookie helpers ----

func (a *App) cookie(name, value, path string, ttl time.Duration, httpOnly bool) *http.Cookie {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   a.Cfg.CookieDomain,
		HttpOnly: httpOnly,
		Secure:   a.Cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
	if ttl > 0 {
		c.Expires = time.Now().Add(ttl)
		c.MaxAge = int(ttl.Seconds())
	} else {
		c.MaxAge = -1
	}
	return c
}

func (a *App) setAuthCookies(w http.ResponseWriter, access, refresh, csrfTok string) {
	http.SetCookie(w, a.cookie(cookieAccess, access, "/", a.Auth.AccessTTL(), true))
	http.SetCookie(w, a.cookie(cookieRefresh, refresh, refreshPath, a.Auth.RefreshTTL(), true))
	// CSRF cookie is readable by JS (double-submit); not httpOnly.
	http.SetCookie(w, a.cookie(cookieCSRF, csrfTok, "/", a.Auth.RefreshTTL(), false))
}

func (a *App) clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, a.cookie(cookieAccess, "", "/", 0, true))
	http.SetCookie(w, a.cookie(cookieRefresh, "", refreshPath, 0, true))
	http.SetCookie(w, a.cookie(cookieCSRF, "", "/", 0, false))
}
