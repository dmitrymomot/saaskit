package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/dmitrymomot/saaskit/pkg/logger"
	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
	"github.com/dmitrymomot/saaskit/pkg/token"
	"github.com/dmitrymomot/saaskit/pkg/validator"
)

const SubjectMagicLink = "magic_link"

// MagicLinkTokenPayload contains the data encoded in magic link tokens
type MagicLinkTokenPayload struct {
	ID       string `json:"id"`    // Token ID for single-use tracking
	Email    string `json:"email"` // User email
	Subject  string `json:"sub"`   // SubjectMagicLink
	ExpireAt int64  `json:"exp"`   // Unix timestamp
}

// MagicLinkRequest contains the magic link data
type MagicLinkRequest struct {
	Email     string
	Token     string
	ExpiresAt time.Time
}

// MagicLinkStorage defines the storage operations required for magic link authentication
type MagicLinkStorage interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUserVerified(ctx context.Context, id uuid.UUID, verified bool) error
}

// MagicLinkService handles passwordless authentication via magic links
type MagicLinkService struct {
	storage      MagicLinkStorage
	tokenSecret  string
	logger       *slog.Logger
	magicLinkTTL time.Duration // TTL for magic link tokens
}

type MagicLinkOption func(*MagicLinkService)

// WithLogger sets a custom logger for the service
func WithLogger(logger *slog.Logger) MagicLinkOption {
	return func(s *MagicLinkService) {
		s.logger = logger
	}
}

// WithMagicLinkTTL sets the TTL for magic link tokens
func WithMagicLinkTTL(ttl time.Duration) MagicLinkOption {
	return func(s *MagicLinkService) {
		s.magicLinkTTL = ttl
	}
}

// NewMagicLinkService creates a new magic link authentication service
func NewMagicLinkService(storage MagicLinkStorage, tokenSecret string, opts ...MagicLinkOption) *MagicLinkService {
	s := &MagicLinkService{
		storage:      storage,
		tokenSecret:  tokenSecret,
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		magicLinkTTL: 15 * time.Minute, // Short TTL reduces risk without replay protection
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// RequestMagicLink generates a magic link token for the given email.
// Auto-registers new users to reduce friction in the onboarding flow.
func (s *MagicLinkService) RequestMagicLink(ctx context.Context, email string) (*MagicLinkRequest, error) {
	email = sanitizer.NormalizeEmail(email)
	if err := validator.Apply(
		validator.ValidEmail("email", email),
	); err != nil {
		return nil, err
	}

	_, err := s.storage.GetUserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, ErrUserNotFound) {
			return nil, fmt.Errorf("failed to check user: %w", err)
		}

		// Auto-register new users to minimize onboarding friction
		user := &User{
			ID:         uuid.New(),
			Email:      email,
			AuthMethod: MethodMagicLink,
			IsVerified: false, // Verified on first successful magic link authentication
			CreatedAt:  time.Now(),
		}

		if err := s.storage.CreateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	expiresAt := time.Now().Add(s.magicLinkTTL)
	payload := MagicLinkTokenPayload{
		ID:       uuid.New().String(), // Unique ID enables future replay protection
		Email:    email,
		Subject:  SubjectMagicLink,
		ExpireAt: expiresAt.Unix(),
	}

	tokenStr, err := token.GenerateToken(payload, s.tokenSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate magic link token: %w", err)
	}

	return &MagicLinkRequest{
		Email:     email,
		Token:     tokenStr,
		ExpiresAt: expiresAt,
	}, nil
}

// VerifyMagicLink validates a magic link token and returns the authenticated user
func (s *MagicLinkService) VerifyMagicLink(ctx context.Context, magicLinkToken string) (*User, error) {
	payload, err := token.ParseToken[MagicLinkTokenPayload](magicLinkToken, s.tokenSecret)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	if payload.Subject != SubjectMagicLink {
		return nil, ErrTokenInvalid
	}

	if time.Now().Unix() > payload.ExpireAt {
		return nil, ErrTokenExpired
	}

	user, err := s.storage.GetUserByEmail(ctx, payload.Email)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.IsVerified {
		if err := s.storage.UpdateUserVerified(ctx, user.ID, true); err != nil {
			s.logger.Error("failed to update user verified status",
				logger.UserID(user.ID.String()),
				slog.String("email", user.Email),
				logger.Error(err),
				logger.Component("magic_link"),
			)
		}
		user.IsVerified = true
	}

	// MVP: Replay protection via token ID tracking deferred.
	// 15-minute TTL provides reasonable security for initial launch.

	return user, nil
}
