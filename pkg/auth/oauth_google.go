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
)

// OAuth provider constants
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

// OAuth errors
var (
	ErrInvalidState    = errors.New("oauth: invalid or expired state")
	ErrInvalidCode     = errors.New("oauth: invalid authorization code")
	ErrUnverifiedEmail = errors.New("oauth: google account email is not verified")
	ErrGoogleLinked    = errors.New("oauth: google account already linked to another user")
	ErrEmailExists     = errors.New("oauth: email already registered with different method")
	ErrNoGoogleLink    = errors.New("oauth: no google account linked")
)

// OAuthStorage interface for OAuth operations (universal for any provider)
type OAuthStorage interface {
	// User operations (reuse existing)
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error

	// Universal OAuth operations
	StoreOAuthLink(ctx context.Context, userID uuid.UUID, provider, providerUserID string) error
	GetUserByOAuth(ctx context.Context, provider, providerUserID string) (*User, error)
	RemoveOAuthLink(ctx context.Context, userID uuid.UUID, provider string) error

	// State validation (CSRF protection)
	StoreState(ctx context.Context, state string, expiresAt time.Time) error
	ConsumeState(ctx context.Context, state string) error
}

// GoogleOAuthService handles Google OAuth authentication
type GoogleOAuthService struct {
	storage      OAuthStorage
	config       GoogleOAuthConfig
	oauth2Config *oauth2.Config
	tokenSecret  string
	logger       *slog.Logger
}

// GoogleOAuthOption is a functional option for GoogleOAuthService
type GoogleOAuthOption func(*GoogleOAuthService)

// WithGoogleLogger sets a custom logger for the service
func WithGoogleLogger(logger *slog.Logger) GoogleOAuthOption {
	return func(s *GoogleOAuthService) {
		s.logger = logger
	}
}

// NewGoogleOAuthService creates a new Google OAuth service
func NewGoogleOAuthService(storage OAuthStorage, config GoogleOAuthConfig, tokenSecret string, opts ...GoogleOAuthOption) *GoogleOAuthService {
	s := &GoogleOAuthService{
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
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)), // noop logger by default
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// GetAuthURL generates an OAuth authorization URL with CSRF protection
func (s *GoogleOAuthService) GetAuthURL(ctx context.Context) (string, error) {
	// Generate secure random state
	state, err := s.generateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state with expiration
	expiresAt := time.Now().Add(s.config.StateTTL)
	if err := s.storage.StoreState(ctx, state, expiresAt); err != nil {
		return "", fmt.Errorf("failed to store state: %w", err)
	}

	// Generate OAuth URL
	url := s.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return url, nil
}

// Auth handles OAuth callback - authenticates user or links to existing user
func (s *GoogleOAuthService) Auth(ctx context.Context, code, state string, linkToUserID *uuid.UUID) (*User, error) {
	// Validate and consume state (one-time use for CSRF protection)
	if err := s.storage.ConsumeState(ctx, state); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidState
		}
		return nil, fmt.Errorf("failed to validate state: %w", err)
	}

	// Exchange code for token
	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, ErrInvalidCode
	}

	// Fetch user info from Google
	googleUser, err := s.fetchGoogleUser(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch google user: %w", err)
	}

	// Reject unverified emails (security measure) - configurable
	if s.config.VerifiedOnly && !googleUser.VerifiedEmail {
		return nil, ErrUnverifiedEmail
	}

	// Handle linking to existing user
	if linkToUserID != nil {
		return s.handleLinking(ctx, *linkToUserID, googleUser)
	}

	// Handle normal authentication flow
	return s.handleAuth(ctx, googleUser)
}

// Unlink removes Google OAuth link from a user
func (s *GoogleOAuthService) Unlink(ctx context.Context, userID uuid.UUID) error {
	if err := s.storage.RemoveOAuthLink(ctx, userID, ProviderGoogle); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrNoGoogleLink
		}
		return fmt.Errorf("failed to unlink google account: %w", err)
	}
	return nil
}

// handleLinking links Google account to existing user
func (s *GoogleOAuthService) handleLinking(ctx context.Context, userID uuid.UUID, googleUser *googleUserInfo) (*User, error) {
	// Check if Google ID is already linked to another account
	existingUser, err := s.storage.GetUserByOAuth(ctx, ProviderGoogle, googleUser.ID)
	if err == nil && existingUser.ID != userID {
		return nil, ErrGoogleLinked
	}

	// Get the user we're linking to
	user, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Store the OAuth link
	if err := s.storage.StoreOAuthLink(ctx, userID, ProviderGoogle, googleUser.ID); err != nil {
		return nil, fmt.Errorf("failed to link google account: %w", err)
	}

	return user, nil
}

// handleAuth handles normal authentication/registration flow
func (s *GoogleOAuthService) handleAuth(ctx context.Context, googleUser *googleUserInfo) (*User, error) {
	// Check if Google ID is already linked
	user, err := s.storage.GetUserByOAuth(ctx, ProviderGoogle, googleUser.ID)
	if err == nil {
		// User authenticated successfully
		return user, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check oauth link: %w", err)
	}

	// Check if email already exists
	_, err = s.storage.GetUserByEmail(ctx, googleUser.Email)
	if err == nil {
		// Email exists with different auth method
		return nil, ErrEmailExists
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	// Create new user
	user = &User{
		ID:         uuid.New(),
		Email:      googleUser.Email,
		AuthMethod: MethodOAuthGoogle,
		IsVerified: googleUser.VerifiedEmail,
		CreatedAt:  time.Now(),
	}

	// Save user
	if err := s.storage.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Store OAuth link
	if err := s.storage.StoreOAuthLink(ctx, user.ID, ProviderGoogle, googleUser.ID); err != nil {
		// Attempt to clean up the created user
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

	return user, nil
}

// generateState generates a cryptographically secure random state
func (s *GoogleOAuthService) generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// googleUserInfo represents the user data from Google API
type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
}

// fetchGoogleUser fetches user information from Google API
func (s *GoogleOAuthService) fetchGoogleUser(ctx context.Context, accessToken string) (*googleUserInfo, error) {
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
