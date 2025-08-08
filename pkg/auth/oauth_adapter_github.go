package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// GitHubOAuthConfig holds configuration for GitHub OAuth provider.
type GitHubOAuthConfig struct {
	ClientID     string        `env:"GITHUB_OAUTH_CLIENT_ID,required"`
	ClientSecret string        `env:"GITHUB_OAUTH_CLIENT_SECRET,required"`
	RedirectURL  string        `env:"GITHUB_OAUTH_REDIRECT_URL,required"`
	Scopes       []string      `env:"GITHUB_OAUTH_SCOPES" envSeparator:"," envDefault:"user:email"`
	StateTTL     time.Duration `env:"GITHUB_OAUTH_STATE_TTL" envDefault:"10m"`
	VerifiedOnly bool          `env:"GITHUB_OAUTH_VERIFIED_ONLY" envDefault:"true"`
}

type githubAdapter struct {
	conf       *oauth2.Config
	httpClient *http.Client
}

// NewGitHubAdapter creates a new GitHub OAuth provider adapter.
func NewGitHubAdapter(cfg GitHubOAuthConfig) ProviderAdapter {
	return &githubAdapter{
		conf: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint:     github.Endpoint,
		},
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ProviderID returns the GitHub provider identifier.
func (a *githubAdapter) ProviderID() string {
	return OAuthProviderGithub
}

// AuthURL builds the GitHub authorization URL with the given state token.
func (a *githubAdapter) AuthURL(state string) (string, error) {
	// We intentionally keep this minimal. Provider-specific options (like offline access)
	// can be added here without leaking details to the core service.
	return a.conf.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// ResolveProfile exchanges the authorization code for user profile information from GitHub.
// Handles both user endpoint and emails endpoint to find verified email addresses.
func (a *githubAdapter) ResolveProfile(ctx context.Context, code string) (ProviderProfile, error) {
	tok, err := a.conf.Exchange(ctx, code)
	if err != nil {
		// Treat exchange failures as invalid code for the core flow.
		return ProviderProfile{}, ErrInvalidCode
	}

	u, err := a.fetchGitHubUser(ctx, tok.AccessToken)
	if err != nil {
		return ProviderProfile{}, fmt.Errorf("fetch github user: %w", err)
	}

	// Always fetch from /user/emails to get proper verification status
	emails, err := a.fetchGitHubEmails(ctx, tok.AccessToken)
	if err != nil {
		return ProviderProfile{}, fmt.Errorf("fetch github emails: %w", err)
	}

	var email string
	var verified bool

	// Prefer primary verified
	for _, e := range emails {
		if e.Primary && e.Verified {
			email = e.Email
			verified = true
			break
		}
	}
	// Fallback to any verified
	if email == "" {
		for _, e := range emails {
			if e.Verified {
				email = e.Email
				verified = true
				break
			}
		}
	}

	if email == "" {
		return ProviderProfile{}, ErrNoPrimaryEmail
	}

	return ProviderProfile{
		ProviderUserID: strconv.FormatInt(u.ID, 10),
		Email:          email,
		EmailVerified:  verified,
	}, nil
}

func (a *githubAdapter) fetchGitHubUser(ctx context.Context, accessToken string) (*ghUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var user ghUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (a *githubAdapter) fetchGitHubEmails(ctx context.Context, accessToken string) ([]ghEmail, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var emails []ghEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	return emails, nil
}

type ghUser struct {
	ID int64 `json:"id"`
}

type ghEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// Compile-time interface assertion
var _ ProviderAdapter = (*githubAdapter)(nil)
