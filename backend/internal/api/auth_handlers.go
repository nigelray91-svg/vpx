package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/vaultproxies/vpx/backend/internal/auth"
	"github.com/vaultproxies/vpx/backend/internal/models"
	"github.com/vaultproxies/vpx/backend/internal/store"
)

type registerReq struct {
	Email          string `json:"email" validate:"required,email,max=255"`
	Password       string `json:"password" validate:"required,min=8,max=72"`
	FullName       string `json:"full_name" validate:"max=120"`
	TurnstileToken string `json:"turnstile_token"`
}

type loginReq struct {
	Email          string `json:"email" validate:"required,email,max=255"`
	Password       string `json:"password" validate:"required,max=72"`
	TurnstileToken string `json:"turnstile_token"`
}

type authResponse struct {
	User        *models.User `json:"user"`
	AccessToken string       `json:"access_token"`
	ExpiresIn   int          `json:"expires_in"`
}

func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	return host
}

func (a *App) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if !decode(w, r, &req) {
		return
	}
	if err := a.Turnst.Verify(r.Context(), req.TurnstileToken, clientIP(r)); err != nil {
		writeErrorCode(w, http.StatusForbidden, "captcha verification failed", "captcha")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid password")
		return
	}
	u, err := a.Store.CreateUser(r.Context(), email, hash, strings.TrimSpace(req.FullName), nil, false)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "an account with that email already exists")
			return
		}
		a.Log.Error("create user", "err", err)
		writeError(w, http.StatusInternalServerError, "could not create account")
		return
	}
	a.issueSession(w, r, u)
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !decode(w, r, &req) {
		return
	}
	if err := a.Turnst.Verify(r.Context(), req.TurnstileToken, clientIP(r)); err != nil {
		writeErrorCode(w, http.StatusForbidden, "captcha verification failed", "captcha")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	u, err := a.Store.GetUserByEmail(r.Context(), email)
	// Constant-ish behaviour: always run a hash compare to avoid user enumeration.
	if errors.Is(err, store.ErrNotFound) {
		auth.CheckPassword("$2a$12$0000000000000000000000000000000000000000000000000000", req.Password)
		writeErrorCode(w, http.StatusUnauthorized, "invalid email or password", "invalid_credentials")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	if u.Status != "active" {
		writeErrorCode(w, http.StatusForbidden, "account is not active", "account_inactive")
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		writeErrorCode(w, http.StatusUnauthorized, "invalid email or password", "invalid_credentials")
		return
	}
	a.issueSession(w, r, u)
}

// startSession mints an access token plus a rotating refresh session and sets
// the auth cookies. It performs no response writing beyond the cookies so both
// the JSON and the OAuth-redirect flows can share it.
func (a *App) startSession(w http.ResponseWriter, r *http.Request, u *models.User) (string, error) {
	access, err := a.Auth.IssueAccessToken(u.ID, u.Role)
	if err != nil {
		return "", err
	}
	refresh, hash, err := auth.NewRefreshToken()
	if err != nil {
		return "", err
	}
	csrfTok, err := auth.RandomString(24)
	if err != nil {
		return "", err
	}
	expires := time.Now().Add(a.Auth.RefreshTTL())
	if err := a.Store.CreateSession(r.Context(), u.ID, hash, r.UserAgent(), clientIP(r), expires); err != nil {
		return "", err
	}
	a.setAuthCookies(w, access, refresh, csrfTok)
	return access, nil
}

// issueSession creates a refresh session + access token and sets cookies.
func (a *App) issueSession(w http.ResponseWriter, r *http.Request, u *models.User) {
	access, err := a.startSession(w, r, u)
	if err != nil {
		a.Log.Error("start session", "err", err, "user", u.ID)
		writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{
		User:        u,
		AccessToken: access,
		ExpiresIn:   int(a.Auth.AccessTTL().Seconds()),
	})
}

func (a *App) handleRefresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(cookieRefresh)
	if err != nil || c.Value == "" {
		writeErrorCode(w, http.StatusUnauthorized, "no refresh token", "unauthenticated")
		return
	}
	hash := auth.HashToken(c.Value)
	sess, err := a.Store.GetSessionByHash(r.Context(), hash)
	if err != nil {
		a.clearAuthCookies(w)
		writeErrorCode(w, http.StatusUnauthorized, "session expired", "unauthenticated")
		return
	}
	// Reuse of an already-rotated refresh token means the token leaked (the
	// legitimate client holds the newer one). Kill the whole family so the
	// thief and the victim are both logged out rather than racing each other.
	if sess.RevokedAt != nil {
		a.Log.Warn("refresh token reuse detected, revoking all sessions", "user", sess.UserID, "ip", clientIP(r))
		if err := a.Store.RevokeAllUserSessions(r.Context(), sess.UserID); err != nil {
			a.Log.Error("revoke all sessions", "err", err, "user", sess.UserID)
		}
		a.clearAuthCookies(w)
		writeErrorCode(w, http.StatusUnauthorized, "session expired", "unauthenticated")
		return
	}
	if time.Now().After(sess.ExpiresAt) {
		a.clearAuthCookies(w)
		writeErrorCode(w, http.StatusUnauthorized, "session expired", "unauthenticated")
		return
	}
	u, err := a.Store.GetUserByID(r.Context(), sess.UserID)
	if err != nil || u.Status != "active" {
		a.clearAuthCookies(w)
		writeErrorCode(w, http.StatusUnauthorized, "session invalid", "unauthenticated")
		return
	}
	// Rotate the refresh token (defends against token theft / replay).
	_ = a.Store.RevokeSession(r.Context(), hash)
	a.issueSession(w, r, u)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieRefresh); err == nil && c.Value != "" {
		_ = a.Store.RevokeSession(r.Context(), auth.HashToken(c.Value))
	}
	a.clearAuthCookies(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	uid, _ := userID(r)
	u, err := a.Store.GetUserByID(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// ---- Google OAuth ----

func (a *App) handleGoogleStart(w http.ResponseWriter, r *http.Request) {
	if !a.Google.Enabled() {
		writeError(w, http.StatusNotImplemented, "google login not configured")
		return
	}
	state, err := auth.RandomString(24)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "oauth error")
		return
	}
	// Store state in Redis for 10 minutes to validate the callback (CSRF for OAuth).
	if err := a.Cache.Set(r.Context(), "oauth:state:"+state, "1", 10*time.Minute); err != nil {
		writeError(w, http.StatusInternalServerError, "oauth error")
		return
	}
	http.Redirect(w, r, a.Google.AuthCodeURL(state), http.StatusFound)
}

func (a *App) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if !a.Google.Enabled() {
		writeError(w, http.StatusNotImplemented, "google login not configured")
		return
	}
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if state == "" || code == "" {
		a.redirectAuthError(w, r, "invalid_oauth_response")
		return
	}
	if _, ok, _ := a.Cache.GetDel(r.Context(), "oauth:state:"+state); !ok {
		a.redirectAuthError(w, r, "invalid_oauth_state")
		return
	}
	gu, err := a.Google.Exchange(r.Context(), code)
	if err != nil {
		a.Log.Warn("google exchange", "err", err)
		a.redirectAuthError(w, r, "oauth_exchange_failed")
		return
	}
	u, err := a.upsertGoogleUser(r.Context(), gu)
	if errors.Is(err, errUnverifiedGoogleEmail) {
		a.redirectAuthError(w, r, "oauth_email_unverified")
		return
	}
	if err != nil {
		a.Log.Error("google upsert", "err", err)
		a.redirectAuthError(w, r, "oauth_account_error")
		return
	}
	if u.Status != "active" {
		a.redirectAuthError(w, r, "account_inactive")
		return
	}
	// Issue cookies, then redirect to the frontend dashboard.
	if _, err := a.startSession(w, r, u); err != nil {
		a.Log.Error("google start session", "err", err, "user", u.ID)
		a.redirectAuthError(w, r, "oauth_session_error")
		return
	}
	http.Redirect(w, r, a.Cfg.PublicBaseURL+"/dashboard", http.StatusFound)
}

// errUnverifiedGoogleEmail is returned when Google reports the profile's email
// as unverified. Trusting it would let anyone who controls a Google Workspace
// domain claim an arbitrary address and take over the matching local account.
var errUnverifiedGoogleEmail = errors.New("google email not verified")

func (a *App) upsertGoogleUser(ctx context.Context, gu *auth.GoogleUser) (*models.User, error) {
	// An existing link is keyed on the immutable Google subject, so it stays
	// valid regardless of the email's current verification state.
	if u, err := a.Store.GetUserByGoogleID(ctx, gu.Sub); err == nil {
		return u, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	// Everything below binds an *email* to an identity, so the email must be
	// proven. Google only sets email_verified for addresses it has validated.
	if !gu.EmailVerified {
		return nil, errUnverifiedGoogleEmail
	}
	email := strings.ToLower(strings.TrimSpace(gu.Email))
	// Link to an existing email account if present.
	if u, err := a.Store.GetUserByEmail(ctx, email); err == nil {
		if err := a.Store.LinkGoogleID(ctx, u.ID, gu.Sub); err != nil {
			return nil, err
		}
		return u, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	sub := gu.Sub
	return a.Store.CreateUser(ctx, email, "", gu.Name, &sub, true)
}

func (a *App) redirectAuthError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, a.Cfg.PublicBaseURL+"/login?error="+code, http.StatusFound)
}
