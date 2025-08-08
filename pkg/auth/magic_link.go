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

// SubjectMagicLink is the token subject used for magic link tokens.
const SubjectMagicLink = "magic_link"

// MagicLinkTokenPayload represents the JWT payload for magic link tokens.
type MagicLinkTokenPayload struct {
	ID       string `json:"id"`    // Token ID for single-use tracking
	Email    string `json:"email"` // User email
	Subject  string `json:"sub"`   // SubjectMagicLink
	ExpireAt int64  `json:"exp"`   // Unix timestamp
}

// MagicLinkRequest contains the generated magic link token and metadata.
type MagicLinkRequest struct {
	Email     string
	Token     string
	ExpiresAt time.Time
}

// MagicLinkAuthenticator defines the interface for passwordless authentication via magic links.
type MagicLinkAuthenticator interface {
	RequestMagicLink(ctx context.Context, email string) (*MagicLinkRequest, error)
	VerifyMagicLink(ctx context.Context, magicLinkToken string) (*User, error)
}

// MagicLinkStorage defines the storage interface required by magic link services.
type MagicLinkStorage interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUserVerified(ctx context.Context, id uuid.UUID, verified bool) error
}

type magicLinkService struct {
	storage      MagicLinkStorage
	tokenSecret  string
	logger       *slog.Logger
	magicLinkTTL time.Duration // TTL for magic link tokens

	// Hooks for extending magic link behavior
	afterGenerate func(ctx context.Context, user *User, token string) error
	beforeVerify  func(ctx context.Context, token string) error
	afterVerify   func(ctx context.Context, user *User) error
}

// MagicLinkOption configures a magic link service during construction.
type MagicLinkOption func(*magicLinkService)

// WithMagicLinkLogger configures the logger for the magic link service.
func WithMagicLinkLogger(logger *slog.Logger) MagicLinkOption {
	return func(s *magicLinkService) {
		s.logger = logger
	}
}

// WithMagicLinkTTL configures the time-to-live for magic link tokens.
func WithMagicLinkTTL(ttl time.Duration) MagicLinkOption {
	return func(s *magicLinkService) {
		s.magicLinkTTL = ttl
	}
}

// WithAfterGenerate configures a hook that runs after magic link generation (async).
func WithAfterGenerate(fn func(context.Context, *User, string) error) MagicLinkOption {
	return func(s *magicLinkService) {
		s.afterGenerate = fn
	}
}

// WithBeforeVerify configures a hook that runs before magic link verification (sync).
func WithBeforeVerify(fn func(context.Context, string) error) MagicLinkOption {
	return func(s *magicLinkService) {
		s.beforeVerify = fn
	}
}

// WithAfterVerify configures a hook that runs after successful magic link verification (async).
func WithAfterVerify(fn func(context.Context, *User) error) MagicLinkOption {
	return func(s *magicLinkService) {
		s.afterVerify = fn
	}
}

// NewMagicLinkService creates a magic link service with bcrypt for hashing and configurable options.
func NewMagicLinkService(storage MagicLinkStorage, tokenSecret string, opts ...MagicLinkOption) MagicLinkAuthenticator {
	s := &magicLinkService{
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
func (s *magicLinkService) RequestMagicLink(ctx context.Context, email string) (*MagicLinkRequest, error) {
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

	req := &MagicLinkRequest{
		Email:     email,
		Token:     tokenStr,
		ExpiresAt: expiresAt,
	}

	// Execute after generate hook if set
	if s.afterGenerate != nil {
		// Get user for hook
		user, _ := s.storage.GetUserByEmail(ctx, email)
		if user != nil {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						s.logger.Error("afterGenerate hook panicked",
							slog.String("email", email),
							slog.Any("panic", r),
							logger.Component("magic_link"),
						)
					}
				}()

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				if err := s.afterGenerate(ctx, user, tokenStr); err != nil {
					s.logger.Error("afterGenerate hook failed",
						slog.String("email", email),
						logger.Error(err),
						logger.Component("magic_link"),
					)
				}
			}()
		}
	}

	return req, nil
}

// VerifyMagicLink validates a magic link token and returns the authenticated user.
// Automatically marks new users as verified on first successful verification.
func (s *magicLinkService) VerifyMagicLink(ctx context.Context, magicLinkToken string) (*User, error) {
	// Execute before verify hook if set
	if s.beforeVerify != nil {
		if err := s.beforeVerify(ctx, magicLinkToken); err != nil {
			return nil, fmt.Errorf("verify blocked: %w", err)
		}
	}
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

	// Execute after verify hook if set
	if s.afterVerify != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("afterVerify hook panicked",
						logger.UserID(user.ID.String()),
						slog.Any("panic", r),
						logger.Component("magic_link"),
					)
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := s.afterVerify(ctx, user); err != nil {
				s.logger.Error("afterVerify hook failed",
					logger.UserID(user.ID.String()),
					logger.Error(err),
					logger.Component("magic_link"),
				)
			}
		}()
	}

	return user, nil
}
