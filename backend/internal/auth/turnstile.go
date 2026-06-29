package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Turnstile verifies Cloudflare Turnstile tokens server-side.
type Turnstile struct {
	secret  string
	enabled bool
	client  *http.Client
}

func NewTurnstile(secret string, enabled bool) *Turnstile {
	return &Turnstile{
		secret:  secret,
		enabled: enabled && secret != "",
		client:  &http.Client{Timeout: 8 * time.Second},
	}
}

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type turnstileResp struct {
	Success     bool     `json:"success"`
	ErrorCodes  []string `json:"error-codes"`
	Hostname    string   `json:"hostname"`
	ChallengeTS string   `json:"challenge_ts"`
}

// Verify returns nil when the token is valid (or when Turnstile is disabled).
func (t *Turnstile) Verify(ctx context.Context, token, remoteIP string) error {
	if !t.enabled {
		return nil
	}
	if strings.TrimSpace(token) == "" {
		return ErrInvalidToken
	}
	form := url.Values{}
	form.Set("secret", t.secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out turnstileResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if !out.Success {
		return ErrInvalidToken
	}
	return nil
}
