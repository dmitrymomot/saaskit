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

// GoogleOAuthConfig holds the configuration for the Google OAuth adapter.
// Keep only provider-specific fields here; policies like VerifiedOnly and StateTTL
// are configured in the core OAuth service options.
type GoogleOAuthConfig struct {
	ClientID     string        `env:"GOOGLE_OAUTH_CLIENT_ID,required"`
	ClientSecret string        `env:"GOOGLE_OAUTH_CLIENT_SECRET,required"`
	RedirectURL  string        `env:"GOOGLE_OAUTH_REDIRECT_URL,required"`
	Scopes       []string      `env:"GOOGLE_OAUTH_SCOPES" envSeparator:"," envDefault:"https://www.googleapis.com/auth/userinfo.email"`
	StateTTL     time.Duration `env:"GOOGLE_OAUTH_STATE_TTL" envDefault:"10m"`
	VerifiedOnly bool          `env:"GOOGLE_OAUTH_VERIFIED_ONLY" envDefault:"true"`
}

// googleAdapter implements ProviderAdapter for Google.
// It encapsulates oauth2.Config and Google userinfo API calls.
type googleAdapter struct {
	conf       *oauth2.Config
	httpClient *http.Client
}

// NewGoogleAdapter constructs a Google ProviderAdapter using the provided config.
// The adapter hides oauth2 details and exposes only the ProviderAdapter surface.
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

func (a *googleAdapter) ProviderID() string {
	return OAuthProviderGoogle
}

func (a *googleAdapter) AuthURL(state string) (string, error) {
	// Provider-specific options can be added here without leaking details to the core service.
	return a.conf.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

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
		// Name, AvatarURL, Raw can be set in the future if needed
	}, nil
}

func (a *googleAdapter) fetchGoogleUser(ctx context.Context, accessToken string) (*gUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := a.httpClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	resp, err := client.Do(req)
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
}
