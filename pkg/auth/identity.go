package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/token"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// EmailChangeRequest contains data for email change verification
type EmailChangeRequest struct {
	CurrentEmail string
	NewEmail     string
	Token        string
	ExpiresAt    time.Time
}

// EmailChangeTokenPayload contains the data encoded in email change tokens
type EmailChangeTokenPayload struct {
	ID       string `json:"id"`  // Identity ID
	OldEmail string `json:"old"` // Current email
	NewEmail string `json:"new"` // New email to change to
	Subject  string `json:"sub"` // SubjectEmailChange
	ExpireAt int64  `json:"exp"` // Unix timestamp
}

// IdentityStorage interface for identity service
type IdentityStorage interface {
	// Identity operations
	GetIdentityByID(ctx context.Context, id uuid.UUID) (*Identity, error)
	GetIdentityByEmail(ctx context.Context, email string) (*Identity, error)
	UpdateIdentityEmail(ctx context.Context, id uuid.UUID, email string) error

	// Password operations
	GetPasswordHash(ctx context.Context, identityID uuid.UUID) ([]byte, error)
	UpdatePasswordHash(ctx context.Context, identityID uuid.UUID, hash []byte) error
}

// IdentityService handles authenticated user account management
type IdentityService struct {
	storage         IdentityStorage
	tokenSecret     string
	bcryptCost      int
	logger          *slog.Logger
	emailChangeTTL  time.Duration // TTL for email change tokens
}

// IdentityOption is a functional option for IdentityService
type IdentityOption func(*IdentityService)

// WithIdentityLogger sets a custom logger for the service
func WithIdentityLogger(logger *slog.Logger) IdentityOption {
	return func(s *IdentityService) {
		s.logger = logger
	}
}

// WithIdentityBcryptCost sets the bcrypt cost for password hashing
func WithIdentityBcryptCost(cost int) IdentityOption {
	return func(s *IdentityService) {
		s.bcryptCost = cost
	}
}

// WithEmailChangeTTL sets the TTL for email change tokens
func WithEmailChangeTTL(ttl time.Duration) IdentityOption {
	return func(s *IdentityService) {
		s.emailChangeTTL = ttl
	}
}

// NewIdentityService creates a new identity management service
func NewIdentityService(storage IdentityStorage, tokenSecret string, opts ...IdentityOption) *IdentityService {
	s := &IdentityService{
		storage:        storage,
		tokenSecret:    tokenSecret,
		bcryptCost:     bcrypt.DefaultCost,
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)), // noop logger by default
		emailChangeTTL: 1 * time.Hour, // Default 1 hour for email change tokens
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// GetIdentity retrieves the identity information for an authenticated user
func (s *IdentityService) GetIdentity(ctx context.Context, identityID uuid.UUID) (*Identity, error) {
	identity, err := s.storage.GetIdentityByID(ctx, identityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}
	return identity, nil
}

// ChangePassword updates the password for an authenticated user
func (s *IdentityService) ChangePassword(ctx context.Context, identityID uuid.UUID, oldPassword, newPassword string) error {
	// Get current password hash
	hash, err := s.storage.GetPasswordHash(ctx, identityID)
	if err != nil {
		return ErrIdentityNotFound
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword(hash, []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.storage.UpdatePasswordHash(ctx, identityID, newHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// RequestEmailChange initiates an email change process by generating a verification token
func (s *IdentityService) RequestEmailChange(ctx context.Context, identityID uuid.UUID, newEmail, currentPassword string) (*EmailChangeRequest, error) {
	// Verify user exists and get current email
	identity, err := s.storage.GetIdentityByID(ctx, identityID)
	if err != nil {
		return nil, ErrIdentityNotFound
	}

	// Check if email is the same - no need to change
	if identity.Email == newEmail {
		return nil, ErrEmailUnchanged
	}

	// Check if new email is already taken
	_, err = s.storage.GetIdentityByEmail(ctx, newEmail)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}
	// Only proceed if error is "not found", otherwise return the error
	if !errors.Is(err, ErrIdentityNotFound) {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	// Verify current password for security
	hash, err := s.storage.GetPasswordHash(ctx, identityID)
	if err != nil {
		return nil, ErrIdentityNotFound
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(currentPassword)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Create email change token
	expiresAt := time.Now().Add(s.emailChangeTTL)
	payload := EmailChangeTokenPayload{
		ID:       identityID.String(),
		OldEmail: identity.Email,
		NewEmail: newEmail,
		Subject:  SubjectEmailChange,
		ExpireAt: expiresAt.Unix(),
	}

	// Generate signed token
	tokenStr, err := token.GenerateToken(payload, s.tokenSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate email change token: %w", err)
	}

	return &EmailChangeRequest{
		CurrentEmail: identity.Email,
		NewEmail:     newEmail,
		Token:        tokenStr,
		ExpiresAt:    expiresAt,
	}, nil
}

// ConfirmEmailChange validates the email change token and updates the email in the database
func (s *IdentityService) ConfirmEmailChange(ctx context.Context, emailChangeToken string) (*Identity, error) {
	// Parse and validate token
	payload, err := token.ParseToken[EmailChangeTokenPayload](emailChangeToken, s.tokenSecret)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// Check subject
	if payload.Subject != SubjectEmailChange {
		return nil, ErrTokenInvalid
	}

	// Check expiration
	if time.Now().Unix() > payload.ExpireAt {
		return nil, ErrTokenExpired
	}

	// Parse identity ID
	identityID, err := uuid.Parse(payload.ID)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// Verify the identity still exists and email hasn't changed
	identity, err := s.storage.GetIdentityByID(ctx, identityID)
	if err != nil {
		return nil, ErrIdentityNotFound
	}

	// Check if current email matches what was in the token
	if identity.Email != payload.OldEmail {
		return nil, ErrTokenInvalid // Email was already changed
	}

	// Check if new email is still available
	_, err = s.storage.GetIdentityByEmail(ctx, payload.NewEmail)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}
	// Only proceed if error is "not found", otherwise return the error
	if !errors.Is(err, ErrIdentityNotFound) {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	// Update email
	if err := s.storage.UpdateIdentityEmail(ctx, identityID, payload.NewEmail); err != nil {
		return nil, fmt.Errorf("failed to update email: %w", err)
	}

	// Return updated identity
	return s.storage.GetIdentityByID(ctx, identityID)
}
