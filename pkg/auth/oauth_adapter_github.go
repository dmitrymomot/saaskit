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

// GitHubOAuthConfig holds the configuration for the GitHub OAuth adapter.
// Fields include provider credentials, scopes, and optional provider policy fields.
// Policy fields (StateTTL, VerifiedOnly) are intended to be consumed by the core
// OAuth service options (e.g., WithStateTTL, WithVerifiedOnly) during wiring.
type GitHubOAuthConfig struct {
	ClientID     string        `env:"GITHUB_OAUTH_CLIENT_ID,required"`
	ClientSecret string        `env:"GITHUB_OAUTH_CLIENT_SECRET,required"`
	RedirectURL  string        `env:"GITHUB_OAUTH_REDIRECT_URL,required"`
	Scopes       []string      `env:"GITHUB_OAUTH_SCOPES" envSeparator:"," envDefault:"user:email"`
	StateTTL     time.Duration `env:"GITHUB_OAUTH_STATE_TTL" envDefault:"10m"`
	VerifiedOnly bool          `env:"GITHUB_OAUTH_VERIFIED_ONLY" envDefault:"true"`
}

// githubAdapter implements ProviderAdapter for GitHub.
// It encapsulates oauth2.Config and all GitHub-specific API calls.
type githubAdapter struct {
	conf       *oauth2.Config
	httpClient *http.Client
}

// NewGitHubAdapter constructs a GitHub ProviderAdapter using the provided config.
// The adapter hides oauth2 details and exposes only the ProviderAdapter surface.
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

func (a *githubAdapter) ProviderID() string {
	return OAuthProviderGithub
}

func (a *githubAdapter) AuthURL(state string) (string, error) {
	// We intentionally keep this minimal. Provider-specific options (like offline access)
	// can be added here without leaking details to the core service.
	return a.conf.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

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

	// GitHub may not return email in the /user endpoint; fetch from /user/emails
	if u.Email == "" {
		emails, err := a.fetchGitHubEmails(ctx, tok.AccessToken)
		if err != nil {
			return ProviderProfile{}, fmt.Errorf("fetch github emails: %w", err)
		}

		// Prefer primary verified
		for _, e := range emails {
			if e.Primary && e.Verified {
				u.Email = e.Email
				u.VerifiedEmail = true
				break
			}
		}
		// Fallback to any verified
		if u.Email == "" {
			for _, e := range emails {
				if e.Verified {
					u.Email = e.Email
					u.VerifiedEmail = true
					break
				}
			}
		}

		if u.Email == "" {
			return ProviderProfile{}, ErrNoPrimaryEmail
		}
	}

	profile := ProviderProfile{
		ProviderUserID: strconv.FormatInt(u.ID, 10),
		Email:          u.Email,
		EmailVerified:  u.VerifiedEmail,
		Name:           firstNonEmpty(u.Name, u.Login),
		// AvatarURL, Raw can be set in the future if needed
	}

	return profile, nil
}

func (a *githubAdapter) fetchGitHubUser(ctx context.Context, accessToken string) (*ghUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

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
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var user ghUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	// If email is present in the user endpoint, assume it's verified.
	if user.Email != "" {
		user.VerifiedEmail = true
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
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var emails []ghEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	return emails, nil
}

type ghUser struct {
	ID            int64  `json:"id"`
	Login         string `json:"login"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	VerifiedEmail bool   // derived from presence/verified flags
}

type ghEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
