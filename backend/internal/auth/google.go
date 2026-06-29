package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleOAuth wraps the Google OAuth2 / OIDC userinfo flow.
type GoogleOAuth struct {
	cfg     *oauth2.Config
	enabled bool
	client  *http.Client
}

func NewGoogleOAuth(clientID, clientSecret, redirectURL string) *GoogleOAuth {
	return &GoogleOAuth{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
		enabled: clientID != "" && clientSecret != "",
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (g *GoogleOAuth) Enabled() bool { return g.enabled }

// AuthCodeURL builds the consent URL with a CSRF state value.
func (g *GoogleOAuth) AuthCodeURL(state string) string {
	return g.cfg.AuthCodeURL(state, oauth2.AccessTypeOnline, oauth2.SetAuthURLParam("prompt", "select_account"))
}

type GoogleUser struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// Exchange swaps an authorization code for the user's verified profile.
func (g *GoogleOAuth) Exchange(ctx context.Context, code string) (*GoogleUser, error) {
	tok, err := g.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oauth exchange: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo status %d", resp.StatusCode)
	}
	var u GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	if u.Sub == "" || u.Email == "" {
		return nil, fmt.Errorf("incomplete google profile")
	}
	return &u, nil
}
