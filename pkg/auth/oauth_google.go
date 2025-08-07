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
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/dmitrymomot/saaskit/pkg/logger"
	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

const (
	ProviderGoogle = "google"
)

// GoogleOAuthConfig holds the configuration for Google OAuth
type GoogleOAuthConfig struct {
	ClientID     string        `env:"GOOGLE_OAUTH_CLIENT_ID,required"`
	ClientSecret string        `env:"GOOGLE_OAUTH_CLIENT_SECRET,required"`
	RedirectURL  string        `env:"GOOGLE_OAUTH_REDIRECT_URL,required"`
	Scopes       []string      `env:"GOOGLE_OAUTH_SCOPES" envSeparator:"," envDefault:"https://www.googleapis.com/auth/userinfo.email"`
	StateTTL     time.Duration `env:"GOOGLE_OAUTH_STATE_TTL" envDefault:"10m"`
	VerifiedOnly bool          `env:"GOOGLE_OAUTH_VERIFIED_ONLY" envDefault:"true"`
}

// googleOAuthService handles Google OAuth authentication
type googleOAuthService struct {
	storage      OAuthStorage
	config       GoogleOAuthConfig
	oauth2Config *oauth2.Config
	tokenSecret  string
	logger       *slog.Logger

	// Hooks for extending OAuth behavior
	afterAuth  func(ctx context.Context, user *User) error
	beforeLink func(ctx context.Context, userID uuid.UUID) error
	afterLink  func(ctx context.Context, user *User) error
}

type GoogleOAuthOption func(*googleOAuthService)

// WithGoogleLogger sets a custom logger for the service
func WithGoogleLogger(logger *slog.Logger) GoogleOAuthOption {
	return func(s *googleOAuthService) {
		s.logger = logger
	}
}

// WithGoogleAfterAuth sets a hook that runs after successful OAuth authentication
func WithGoogleAfterAuth(fn func(context.Context, *User) error) GoogleOAuthOption {
	return func(s *googleOAuthService) {
		s.afterAuth = fn
	}
}

// WithGoogleBeforeLink sets a hook that runs before linking OAuth account
func WithGoogleBeforeLink(fn func(context.Context, uuid.UUID) error) GoogleOAuthOption {
	return func(s *googleOAuthService) {
		s.beforeLink = fn
	}
}

// WithGoogleAfterLink sets a hook that runs after successful OAuth account linking
func WithGoogleAfterLink(fn func(context.Context, *User) error) GoogleOAuthOption {
	return func(s *googleOAuthService) {
		s.afterLink = fn
	}
}

// NewGoogleOAuthService creates a new Google OAuth service
func NewGoogleOAuthService(storage OAuthStorage, config GoogleOAuthConfig, tokenSecret string, opts ...GoogleOAuthOption) OAuthAuthenticator {
	s := &googleOAuthService{
		storage: storage,
		config:  config,
		oauth2Config: &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes:       config.Scopes,
			Endpoint:     google.Endpoint,
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
func (s *googleOAuthService) GetAuthURL(ctx context.Context) (string, error) {
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
func (s *googleOAuthService) Auth(ctx context.Context, code, state string, linkToUserID *uuid.UUID) (*User, error) {
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

	googleUser, err := s.fetchGoogleUser(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch google user: %w", err)
	}

	googleUser.Email = sanitizer.NormalizeEmail(googleUser.Email)

	// Security: reject unverified emails to prevent account takeover via unverified Gmail accounts
	if s.config.VerifiedOnly && !googleUser.VerifiedEmail {
		return nil, ErrUnverifiedEmail
	}

	if linkToUserID != nil {
		return s.handleLinking(ctx, *linkToUserID, googleUser)
	}

	return s.handleAuth(ctx, googleUser)
}

// Unlink removes Google OAuth link from a user
func (s *googleOAuthService) Unlink(ctx context.Context, userID uuid.UUID) error {
	if err := s.storage.RemoveOAuthLink(ctx, userID, ProviderGoogle); err != nil {
		if errors.Is(err, ErrNoProviderLink) {
			return ErrNoProviderLink
		}
		return fmt.Errorf("failed to unlink google account: %w", err)
	}
	return nil
}

func (s *googleOAuthService) handleLinking(ctx context.Context, userID uuid.UUID, googleUser *googleUserInfo) (*User, error) {
	existingUser, err := s.storage.GetUserByOAuth(ctx, ProviderGoogle, googleUser.ID)
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

	if err := s.storage.StoreOAuthLink(ctx, userID, ProviderGoogle, googleUser.ID); err != nil {
		return nil, fmt.Errorf("failed to link google account: %w", err)
	}

	// Execute after link hook if set (only if actually linked)
	if s.afterLink != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("afterLink hook panicked",
						logger.UserID(user.ID.String()),
						slog.Any("panic", r),
						logger.Component("google_oauth"),
					)
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := s.afterLink(ctx, user); err != nil {
				s.logger.Error("afterLink hook failed",
					logger.UserID(user.ID.String()),
					logger.Error(err),
					logger.Component("google_oauth"),
				)
			}
		}()
	}

	return user, nil
}

func (s *googleOAuthService) handleAuth(ctx context.Context, googleUser *googleUserInfo) (*User, error) {
	user, err := s.storage.GetUserByOAuth(ctx, ProviderGoogle, googleUser.ID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check oauth link: %w", err)
	}

	_, err = s.storage.GetUserByEmail(ctx, googleUser.Email)
	if err == nil {
		return nil, ErrProviderEmailInUse // Prevent account takeover via OAuth
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	user = &User{
		ID:         uuid.New(),
		Email:      googleUser.Email,
		AuthMethod: MethodOAuthGoogle,
		IsVerified: googleUser.VerifiedEmail,
		CreatedAt:  time.Now(),
	}

	if err := s.storage.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := s.storage.StoreOAuthLink(ctx, user.ID, ProviderGoogle, googleUser.ID); err != nil {
		// Clean up user record to maintain consistency if OAuth link fails
		if deleteErr := s.storage.DeleteUser(ctx, user.ID); deleteErr != nil {
			s.logger.Error("failed to cleanup user after oauth link save failure",
				logger.UserID(user.ID.String()),
				slog.String("email", user.Email),
				slog.String("provider", ProviderGoogle),
				logger.Error(deleteErr),
				logger.Component("google_oauth"),
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
						logger.Component("google_oauth"),
					)
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := s.afterAuth(ctx, user); err != nil {
				s.logger.Error("afterAuth hook failed",
					logger.UserID(user.ID.String()),
					logger.Error(err),
					logger.Component("google_oauth"),
				)
			}
		}()
	}

	return user, nil
}

func (s *googleOAuthService) generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
}

func (s *googleOAuthService) fetchGoogleUser(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google api returned status %d", resp.StatusCode)
	}

	var user googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}
