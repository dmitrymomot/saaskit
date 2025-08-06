package auth

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/logger"
	"github.com/dmitrymomot/saaskit/pkg/token"
	"github.com/google/uuid"
)

// Token subject for magic link authentication
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

// MagicLinkStorage interface for magic link service
type MagicLinkStorage interface {
	// Identity operations
	GetIdentityByEmail(ctx context.Context, email string) (*Identity, error)
	CreateIdentity(ctx context.Context, identity *Identity) error
	UpdateIdentityVerified(ctx context.Context, id uuid.UUID, verified bool) error
}

// MagicLinkService handles passwordless authentication via magic links
type MagicLinkService struct {
	storage     MagicLinkStorage
	tokenSecret string
	logger      *slog.Logger
}

// MagicLinkOption is a functional option for MagicLinkService
type MagicLinkOption func(*MagicLinkService)

// WithLogger sets a custom logger for the service
func WithLogger(logger *slog.Logger) MagicLinkOption {
	return func(s *MagicLinkService) {
		s.logger = logger
	}
}

// NewMagicLinkService creates a new magic link authentication service
func NewMagicLinkService(storage MagicLinkStorage, tokenSecret string, opts ...MagicLinkOption) *MagicLinkService {
	s := &MagicLinkService{
		storage:     storage,
		tokenSecret: tokenSecret,
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)), // noop logger by default
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// RequestMagicLink generates a magic link token for the given email
func (s *MagicLinkService) RequestMagicLink(ctx context.Context, email string) (*MagicLinkRequest, error) {
	// Check if identity exists, create if not (auto-registration)
	_, err := s.storage.GetIdentityByEmail(ctx, email)
	if err != nil {
		// Create new identity if doesn't exist
		identity := &Identity{
			ID:         uuid.New(),
			Email:      email,
			AuthMethod: MethodMagicLink,
			IsVerified: false, // Will be verified on first successful login
			CreatedAt:  time.Now(),
		}

		if err := s.storage.CreateIdentity(ctx, identity); err != nil {
			return nil, fmt.Errorf("failed to create identity: %w", err)
		}
	}

	// Create magic link token with short expiry (15 minutes)
	expiresAt := time.Now().Add(15 * time.Minute)
	payload := MagicLinkTokenPayload{
		ID:       uuid.New().String(), // Unique token ID for single-use tracking
		Email:    email,
		Subject:  SubjectMagicLink,
		ExpireAt: expiresAt.Unix(),
	}

	// Generate signed token
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

// VerifyMagicLink validates a magic link token and returns the authenticated identity
func (s *MagicLinkService) VerifyMagicLink(ctx context.Context, magicLinkToken string) (*Identity, error) {
	// Parse and validate token
	var payload MagicLinkTokenPayload
	payload, err := token.ParseToken[MagicLinkTokenPayload](magicLinkToken, s.tokenSecret)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// Check subject
	if payload.Subject != SubjectMagicLink {
		return nil, ErrTokenInvalid
	}

	// Check expiration
	if time.Now().Unix() > payload.ExpireAt {
		return nil, ErrTokenExpired
	}

	// Get identity by email
	identity, err := s.storage.GetIdentityByEmail(ctx, payload.Email)
	if err != nil {
		return nil, ErrIdentityNotFound
	}

	// Mark identity as verified on successful magic link authentication
	if !identity.IsVerified {
		if err := s.storage.UpdateIdentityVerified(ctx, identity.ID, true); err != nil {
			s.logger.Error("failed to update identity verified status",
				logger.UserID(identity.ID.String()),
				slog.String("email", identity.Email),
				logger.Error(err),
				logger.Component("magic_link"),
			)
		}
		identity.IsVerified = true
	}

	// Note: Token ID (payload.ID) could be tracked in storage to prevent
	// replay attacks, but for MVP the short 15-minute expiry is sufficient

	return identity, nil
}