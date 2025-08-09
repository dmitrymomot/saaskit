package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/dmitrymomot/saaskit/pkg/logger"
	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

// Ensure oauthService implements OAuthAuthenticator.
var _ OAuthAuthenticator = (*oauthService)(nil)

type oauthService struct {
	storage      OAuthStorage
	adapter      ProviderAdapter
	logger       *slog.Logger
	stateTTL     time.Duration
	verifiedOnly bool

	// Hooks for extending OAuth behavior
	afterAuth  func(ctx context.Context, user *User) error
	beforeLink func(ctx context.Context, userID uuid.UUID) error
	afterLink  func(ctx context.Context, user *User) error
}

// OAuthOption configures an OAuth service during construction.
type OAuthOption func(*oauthService)

// WithLogger configures the logger for the OAuth service.
func WithLogger(l *slog.Logger) OAuthOption {
	return func(s *oauthService) {
		s.logger = l
	}
}

// WithStateTTL configures the TTL for state tokens used in CSRF protection.
// Typical wiring: WithStateTTL(cfg.StateTTL) where cfg comes from a provider adapter config
// (e.g., GoogleOAuthConfig, GitHubOAuthConfig) loaded via pkg/config.
func WithStateTTL(ttl time.Duration) OAuthOption {
	return func(s *oauthService) {
		s.stateTTL = ttl
	}
}

// WithVerifiedOnly enforces that only verified provider emails are accepted.
// Typical wiring: WithVerifiedOnly(cfg.VerifiedOnly) where cfg comes from a provider
// adapter config (e.g., GoogleOAuthConfig, GitHubOAuthConfig) loaded via pkg/config.
func WithVerifiedOnly(verifiedOnly bool) OAuthOption {
	return func(s *oauthService) {
		s.verifiedOnly = verifiedOnly
	}
}

// WithAfterAuth configures a hook that runs after successful OAuth authentication (async).
func WithAfterAuth(fn func(context.Context, *User) error) OAuthOption {
	return func(s *oauthService) {
		s.afterAuth = fn
	}
}

// WithBeforeLink configures a hook that runs before linking an OAuth account (sync).
func WithBeforeLink(fn func(context.Context, uuid.UUID) error) OAuthOption {
	return func(s *oauthService) {
		s.beforeLink = fn
	}
}

// WithAfterLink configures a hook that runs after successful OAuth account linking (async).
func WithAfterLink(fn func(context.Context, *User) error) OAuthOption {
	return func(s *oauthService) {
		s.afterLink = fn
	}
}

// NewOAuthService constructs a new provider-agnostic OAuth service.
// Defaults: verifiedOnly = true, stateTTL = 10 minutes, logger discards by default.
// Typical wiring from adapter configs:
//
//	oauth := NewOAuthService(
//	  storage,
//	  NewGoogleAdapter(googleCfg), // or NewGitHubAdapter(githubCfg)
//	  WithStateTTL(googleCfg.StateTTL),
//	  WithVerifiedOnly(googleCfg.VerifiedOnly),
//	  WithLogger(logger),
//	)
func NewOAuthService(storage OAuthStorage, adapter ProviderAdapter, opts ...OAuthOption) OAuthAuthenticator {
	s := &oauthService{
		storage:      storage,
		adapter:      adapter,
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		stateTTL:     10 * time.Minute,
		verifiedOnly: true,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GetAuthURL generates an OAuth authorization URL with CSRF protection via state parameter.
func (s *oauthService) GetAuthURL(ctx context.Context) (string, error) {
	state, err := s.generateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	expiresAt := time.Now().Add(s.stateTTL)
	if err := s.storage.StoreState(ctx, state, expiresAt); err != nil {
		return "", fmt.Errorf("failed to store state: %w", err)
	}

	url, err := s.adapter.AuthURL(state)
	if err != nil {
		return "", fmt.Errorf("failed to build auth url: %w", err)
	}
	return url, nil
}

// Auth handles OAuth callback - authenticates user or links to existing user.
// State validation prevents CSRF attacks by ensuring request originated from our auth flow.
func (s *oauthService) Auth(ctx context.Context, code, state string, linkToUserID *uuid.UUID) (*User, error) {
	// Consume state token (one-time use prevents replay attacks)
	if err := s.storage.ConsumeState(ctx, state); err != nil {
		if errors.Is(err, ErrStateNotFound) {
			return nil, ErrInvalidState
		}
		return nil, fmt.Errorf("failed to validate state: %w", err)
	}

	// Provider-specific: exchange code and fetch normalized profile
	profile, err := s.adapter.ResolveProfile(ctx, code)
	if err != nil {
		if errors.Is(err, ErrInvalidCode) {
			return nil, ErrInvalidCode
		}
		// Allow specific errors like ErrNoPrimaryEmail to bubble up while adding context
		return nil, fmt.Errorf("failed to resolve provider profile: %w", err)
	}

	// Validate profile data
	if profile.ProviderUserID == "" {
		return nil, fmt.Errorf("invalid profile: missing provider user ID")
	}
	if profile.Email == "" {
		return nil, fmt.Errorf("invalid profile: missing email address")
	}

	// Normalize email centrally
	profile.Email = sanitizer.NormalizeEmail(profile.Email)

	// Security: reject unverified emails to prevent account takeover
	if s.verifiedOnly && !profile.EmailVerified {
		return nil, ErrUnverifiedEmail
	}

	if linkToUserID != nil {
		return s.handleLinking(ctx, *linkToUserID, profile)
	}

	return s.handleAuth(ctx, profile)
}

// Unlink removes the OAuth link for the configured provider from a user account.
func (s *oauthService) Unlink(ctx context.Context, userID uuid.UUID) error {
	if err := s.storage.RemoveOAuthLink(ctx, userID, s.adapter.ProviderID()); err != nil {
		if errors.Is(err, ErrNoProviderLink) {
			return ErrNoProviderLink
		}
		return fmt.Errorf("failed to unlink %s account: %w", s.adapter.ProviderID(), err)
	}
	return nil
}

func (s *oauthService) handleLinking(ctx context.Context, userID uuid.UUID, profile ProviderProfile) (*User, error) {
	existingUser, err := s.storage.GetUserByOAuth(ctx, s.adapter.ProviderID(), profile.ProviderUserID)
	if err == nil {
		if existingUser.ID != userID {
			return nil, ErrProviderLinked
		}
		// Already linked to this user, nothing to do
		return existingUser, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing oauth link: %w", err)
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

	if err := s.storage.StoreOAuthLink(ctx, userID, s.adapter.ProviderID(), profile.ProviderUserID); err != nil {
		return nil, fmt.Errorf("failed to link %s account: %w", s.adapter.ProviderID(), err)
	}

	// Execute after link hook if set (only if actually linked)
	if s.afterLink != nil {
		hookCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := s.afterLink(hookCtx, user); err != nil {
			s.logger.Error("afterLink hook failed",
				logger.UserID(user.ID.String()),
				logger.Error(err),
				logger.Component("oauth"),
				slog.String("provider", s.adapter.ProviderID()),
			)
		}
	}

	return user, nil
}

func (s *oauthService) handleAuth(ctx context.Context, profile ProviderProfile) (*User, error) {
	user, err := s.storage.GetUserByOAuth(ctx, s.adapter.ProviderID(), profile.ProviderUserID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check oauth link: %w", err)
	}

	_, err = s.storage.GetUserByEmail(ctx, profile.Email)
	if err == nil {
		return nil, ErrProviderEmailInUse // Prevent account takeover via OAuth
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	user = &User{
		ID:    uuid.New(),
		Email: profile.Email,
		AuthMethod: func() string {
			switch s.adapter.ProviderID() {
			case OAuthProviderGoogle:
				return MethodOAuthGoogle
			case OAuthProviderGithub:
				return MethodOAuthGithub
			default:
				return "oauth_" + s.adapter.ProviderID()
			}
		}(),
		IsVerified: profile.EmailVerified,
		CreatedAt:  time.Now(),
	}

	if err := s.storage.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := s.storage.StoreOAuthLink(ctx, user.ID, s.adapter.ProviderID(), profile.ProviderUserID); err != nil {
		// Clean up user record to maintain consistency if OAuth link fails
		if deleteErr := s.storage.DeleteUser(ctx, user.ID); deleteErr != nil {
			s.logger.Error("failed to cleanup user after oauth link save failure",
				logger.UserID(user.ID.String()),
				slog.String("email", user.Email),
				slog.String("provider", s.adapter.ProviderID()),
				logger.Error(deleteErr),
				logger.Component("oauth"),
			)
		}
		return nil, fmt.Errorf("failed to store oauth link: %w", err)
	}

	// Execute after auth hook if set
	if s.afterAuth != nil {
		hookCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := s.afterAuth(hookCtx, user); err != nil {
			s.logger.Error("afterAuth hook failed",
				logger.UserID(user.ID.String()),
				logger.Error(err),
				logger.Component("oauth"),
				slog.String("provider", s.adapter.ProviderID()),
			)
		}
	}

	return user, nil
}

func (s *oauthService) generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
