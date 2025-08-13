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

// GoogleOAuthConfig holds configuration for Google OAuth provider.
type GoogleOAuthConfig struct {
	ClientID     string        `env:"GOOGLE_OAUTH_CLIENT_ID,required"`
	ClientSecret string        `env:"GOOGLE_OAUTH_CLIENT_SECRET,required"`
	RedirectURL  string        `env:"GOOGLE_OAUTH_REDIRECT_URL,required"`
	Scopes       []string      `env:"GOOGLE_OAUTH_SCOPES" envSeparator:"," envDefault:"https://www.googleapis.com/auth/userinfo.email"`
	StateTTL     time.Duration `env:"GOOGLE_OAUTH_STATE_TTL" envDefault:"10m"`
	VerifiedOnly bool          `env:"GOOGLE_OAUTH_VERIFIED_ONLY" envDefault:"true"`
}

type googleAdapter struct {
	conf       *oauth2.Config
	httpClient *http.Client
}

// NewGoogleAdapter creates a new Google OAuth provider adapter.
func NewGoogleAdapter(cfg GoogleOAuthConfig) ProviderAdapter {
	return &googleAdapter{
		conf: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint:     google.Endpoint,
		},
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ProviderID returns the Google provider identifier.
func (a *googleAdapter) ProviderID() string {
	return OAuthProviderGoogle
}

// AuthURL builds the Google authorization URL with the given state token.
func (a *googleAdapter) AuthURL(state string) (string, error) {
	// Provider-specific options can be added here without leaking details to the core service.
	return a.conf.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// ResolveProfile exchanges the authorization code for user profile information from Google.
func (a *googleAdapter) ResolveProfile(ctx context.Context, code string) (ProviderProfile, error) {
	tok, err := a.conf.Exchange(ctx, code)
	if err != nil {
		// Treat exchange failures as invalid code for the core flow.
		return ProviderProfile{}, ErrInvalidCode
	}

	u, err := a.fetchGoogleUser(ctx, tok.AccessToken)
	if err != nil {
		return ProviderProfile{}, fmt.Errorf("fetch google user: %w", err)
	}
	if u.Email == "" {
		return ProviderProfile{}, ErrNoPrimaryEmail
	}

	return ProviderProfile{
		ProviderUserID: u.ID,
		Email:          u.Email,
		EmailVerified:  u.VerifiedEmail,
		Name:           u.Name,
		AvatarURL:      u.Picture,
	}, nil
}

func (a *googleAdapter) fetchGoogleUser(ctx context.Context, accessToken string) (*gUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google api returned status %d", resp.StatusCode)
	}

	var user gUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

type gUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// Compile-time interface assertion
var _ ProviderAdapter = (*googleAdapter)(nil)
