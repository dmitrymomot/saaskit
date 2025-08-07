package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/dmitrymomot/saaskit/pkg/logger"
	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

const (
	ProviderGithub = "github"
)

// GitHubOAuthConfig holds the configuration for GitHub OAuth
type GitHubOAuthConfig struct {
	ClientID     string        `env:"GITHUB_OAUTH_CLIENT_ID,required"`
	ClientSecret string        `env:"GITHUB_OAUTH_CLIENT_SECRET,required"`
	RedirectURL  string        `env:"GITHUB_OAUTH_REDIRECT_URL,required"`
	Scopes       []string      `env:"GITHUB_OAUTH_SCOPES" envSeparator:"," envDefault:"user:email"`
	StateTTL     time.Duration `env:"GITHUB_OAUTH_STATE_TTL" envDefault:"10m"`
	VerifiedOnly bool          `env:"GITHUB_OAUTH_VERIFIED_ONLY" envDefault:"true"`
}

// gitHubOAuthService handles GitHub OAuth authentication
type gitHubOAuthService struct {
	storage      OAuthStorage
	config       GitHubOAuthConfig
	oauth2Config *oauth2.Config
	tokenSecret  string
	logger       *slog.Logger

	// Hooks for extending OAuth behavior
	afterAuth  func(ctx context.Context, user *User) error
	beforeLink func(ctx context.Context, userID uuid.UUID) error
	afterLink  func(ctx context.Context, user *User) error
}

type GitHubOAuthOption func(*gitHubOAuthService)

// WithGitHubLogger sets a custom logger for the service
func WithGitHubLogger(logger *slog.Logger) GitHubOAuthOption {
	return func(s *gitHubOAuthService) {
		s.logger = logger
	}
}

// WithGitHubAfterAuth sets a hook that runs after successful OAuth authentication
func WithGitHubAfterAuth(fn func(context.Context, *User) error) GitHubOAuthOption {
	return func(s *gitHubOAuthService) {
		s.afterAuth = fn
	}
}

// WithGitHubBeforeLink sets a hook that runs before linking OAuth account
func WithGitHubBeforeLink(fn func(context.Context, uuid.UUID) error) GitHubOAuthOption {
	return func(s *gitHubOAuthService) {
		s.beforeLink = fn
	}
}

// WithGitHubAfterLink sets a hook that runs after successful OAuth account linking
func WithGitHubAfterLink(fn func(context.Context, *User) error) GitHubOAuthOption {
	return func(s *gitHubOAuthService) {
		s.afterLink = fn
	}
}

// NewGitHubOAuthService creates a new GitHub OAuth service
func NewGitHubOAuthService(storage OAuthStorage, config GitHubOAuthConfig, tokenSecret string, opts ...GitHubOAuthOption) OAuthAuthenticator {
	s := &gitHubOAuthService{
		storage: storage,
		config:  config,
		oauth2Config: &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes:       config.Scopes,
			Endpoint:     github.Endpoint,
		},
		tokenSecret: tokenSecret,
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// GetAuthURL generates an OAuth authorization URL with CSRF protection via state parameter
func (s *gitHubOAuthService) GetAuthURL(ctx context.Context) (string, error) {
	state, err := s.generateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	expiresAt := time.Now().Add(s.config.StateTTL)
	if err := s.storage.StoreState(ctx, state, expiresAt); err != nil {
		return "", fmt.Errorf("failed to store state: %w", err)
	}

	url := s.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return url, nil
}

// Auth handles OAuth callback - authenticates user or links to existing user.
// State validation prevents CSRF attacks by ensuring request originated from our auth flow.
func (s *gitHubOAuthService) Auth(ctx context.Context, code, state string, linkToUserID *uuid.UUID) (*User, error) {
	// Consume state token (one-time use prevents replay attacks)
	if err := s.storage.ConsumeState(ctx, state); err != nil {
		if errors.Is(err, ErrStateNotFound) {
			return nil, ErrInvalidState
		}
		return nil, fmt.Errorf("failed to validate state: %w", err)
	}

	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, ErrInvalidCode
	}

	githubUser, err := s.fetchGitHubUser(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch github user: %w", err)
	}

	// GitHub may not return email in user endpoint, fetch from emails endpoint
	if githubUser.Email == "" {
		emails, err := s.fetchGitHubEmails(ctx, token.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch github emails: %w", err)
		}

		// Find primary verified email
		for _, email := range emails {
			if email.Primary && email.Verified {
				githubUser.Email = email.Email
				githubUser.VerifiedEmail = true
				break
			}
		}

		// Fallback to any verified email if no primary found
		if githubUser.Email == "" {
			for _, email := range emails {
				if email.Verified {
					githubUser.Email = email.Email
					githubUser.VerifiedEmail = true
					break
				}
			}
		}

		if githubUser.Email == "" {
			return nil, ErrNoPrimaryEmail
		}
	}

	githubUser.Email = sanitizer.NormalizeEmail(githubUser.Email)

	// Security: reject unverified emails to prevent account takeover
	if s.config.VerifiedOnly && !githubUser.VerifiedEmail {
		return nil, ErrUnverifiedEmail
	}

	if linkToUserID != nil {
		return s.handleLinking(ctx, *linkToUserID, githubUser)
	}

	return s.handleAuth(ctx, githubUser)
}

// Unlink removes GitHub OAuth link from a user
func (s *gitHubOAuthService) Unlink(ctx context.Context, userID uuid.UUID) error {
	if err := s.storage.RemoveOAuthLink(ctx, userID, ProviderGithub); err != nil {
		if errors.Is(err, ErrNoProviderLink) {
			return ErrNoProviderLink
		}
		return fmt.Errorf("failed to unlink github account: %w", err)
	}
	return nil
}

func (s *gitHubOAuthService) handleLinking(ctx context.Context, userID uuid.UUID, githubUser *githubUserInfo) (*User, error) {
	// Convert GitHub numeric ID to string for storage
	providerUserID := strconv.FormatInt(githubUser.ID, 10)

	existingUser, err := s.storage.GetUserByOAuth(ctx, ProviderGithub, providerUserID)
	if err == nil {
		if existingUser.ID != userID {
			return nil, ErrProviderLinked
		}
		// Already linked to this user, nothing to do
		return existingUser, nil
	}

	// Execute before link hook if set (only if actually linking)
	if s.beforeLink != nil {
		if err := s.beforeLink(ctx, userID); err != nil {
			return nil, fmt.Errorf("link blocked: %w", err)
		}
	}

	user, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := s.storage.StoreOAuthLink(ctx, userID, ProviderGithub, providerUserID); err != nil {
		return nil, fmt.Errorf("failed to link github account: %w", err)
	}

	// Execute after link hook if set (only if actually linked)
	if s.afterLink != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("afterLink hook panicked",
						logger.UserID(user.ID.String()),
						slog.Any("panic", r),
						logger.Component("github_oauth"),
					)
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := s.afterLink(ctx, user); err != nil {
				s.logger.Error("afterLink hook failed",
					logger.UserID(user.ID.String()),
					logger.Error(err),
					logger.Component("github_oauth"),
				)
			}
		}()
	}

	return user, nil
}

func (s *gitHubOAuthService) handleAuth(ctx context.Context, githubUser *githubUserInfo) (*User, error) {
	// Convert GitHub numeric ID to string for storage
	providerUserID := strconv.FormatInt(githubUser.ID, 10)

	user, err := s.storage.GetUserByOAuth(ctx, ProviderGithub, providerUserID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check oauth link: %w", err)
	}

	_, err = s.storage.GetUserByEmail(ctx, githubUser.Email)
	if err == nil {
		return nil, ErrProviderEmailInUse // Prevent account takeover via OAuth
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	user = &User{
		ID:         uuid.New(),
		Email:      githubUser.Email,
		AuthMethod: MethodOAuthGithub,
		IsVerified: githubUser.VerifiedEmail,
		CreatedAt:  time.Now(),
	}

	if err := s.storage.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := s.storage.StoreOAuthLink(ctx, user.ID, ProviderGithub, providerUserID); err != nil {
		// Clean up user record to maintain consistency if OAuth link fails
		if deleteErr := s.storage.DeleteUser(ctx, user.ID); deleteErr != nil {
			s.logger.Error("failed to cleanup user after oauth link save failure",
				logger.UserID(user.ID.String()),
				slog.String("email", user.Email),
				slog.String("provider", ProviderGithub),
				logger.Error(deleteErr),
				logger.Component("github_oauth"),
			)
		}
		return nil, fmt.Errorf("failed to store oauth link: %w", err)
	}

	// Execute after auth hook if set
	if s.afterAuth != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("afterAuth hook panicked",
						logger.UserID(user.ID.String()),
						slog.Any("panic", r),
						logger.Component("github_oauth"),
					)
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := s.afterAuth(ctx, user); err != nil {
				s.logger.Error("afterAuth hook failed",
					logger.UserID(user.ID.String()),
					logger.Error(err),
					logger.Component("github_oauth"),
				)
			}
		}()
	}

	return user, nil
}

func (s *gitHubOAuthService) generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

type githubUserInfo struct {
	ID            int64  `json:"id"`
	Login         string `json:"login"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	VerifiedEmail bool   // Set based on email verification status
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (s *gitHubOAuthService) fetchGitHubUser(ctx context.Context, accessToken string) (*githubUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var user githubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	// If email is present in user endpoint, assume it's verified
	// (GitHub only shows email in this endpoint if it's public and verified)
	if user.Email != "" {
		user.VerifiedEmail = true
	}

	return &user, nil
}

func (s *gitHubOAuthService) fetchGitHubEmails(ctx context.Context, accessToken string) ([]githubEmail, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var emails []githubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	return emails, nil
}
