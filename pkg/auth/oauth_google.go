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

var (
	ErrInvalidState    = errors.New("oauth: invalid or expired state")
	ErrInvalidCode     = errors.New("oauth: invalid authorization code")
	ErrUnverifiedEmail = errors.New("oauth: google account email is not verified")
	ErrGoogleLinked    = errors.New("oauth: google account already linked to another user")
	ErrEmailExists     = errors.New("oauth: email already registered with different method")
	ErrNoGoogleLink    = errors.New("oauth: no google account linked")
)

// OAuthStorage defines storage operations for OAuth authentication (provider-agnostic)
type OAuthStorage interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	StoreOAuthLink(ctx context.Context, userID uuid.UUID, provider, providerUserID string) error
	GetUserByOAuth(ctx context.Context, provider, providerUserID string) (*User, error)
	RemoveOAuthLink(ctx context.Context, userID uuid.UUID, provider string) error
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
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// GetAuthURL generates an OAuth authorization URL with CSRF protection via state parameter
func (s *GoogleOAuthService) GetAuthURL(ctx context.Context) (string, error) {
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
func (s *GoogleOAuthService) Auth(ctx context.Context, code, state string, linkToUserID *uuid.UUID) (*User, error) {
	// Consume state token (one-time use prevents replay attacks)
	if err := s.storage.ConsumeState(ctx, state); err != nil {
		if errors.Is(err, ErrUserNotFound) {
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
func (s *GoogleOAuthService) Unlink(ctx context.Context, userID uuid.UUID) error {
	if err := s.storage.RemoveOAuthLink(ctx, userID, ProviderGoogle); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrNoGoogleLink
		}
		return fmt.Errorf("failed to unlink google account: %w", err)
	}
	return nil
}

func (s *GoogleOAuthService) handleLinking(ctx context.Context, userID uuid.UUID, googleUser *googleUserInfo) (*User, error) {
	existingUser, err := s.storage.GetUserByOAuth(ctx, ProviderGoogle, googleUser.ID)
	if err == nil && existingUser.ID != userID {
		return nil, ErrGoogleLinked
	}

	user, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := s.storage.StoreOAuthLink(ctx, userID, ProviderGoogle, googleUser.ID); err != nil {
		return nil, fmt.Errorf("failed to link google account: %w", err)
	}

	return user, nil
}

func (s *GoogleOAuthService) handleAuth(ctx context.Context, googleUser *googleUserInfo) (*User, error) {
	user, err := s.storage.GetUserByOAuth(ctx, ProviderGoogle, googleUser.ID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check oauth link: %w", err)
	}

	_, err = s.storage.GetUserByEmail(ctx, googleUser.Email)
	if err == nil {
		return nil, ErrEmailExists // Prevent account takeover via OAuth
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

	return user, nil
}

func (s *GoogleOAuthService) generateState() (string, error) {
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
