package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/logger"
	"github.com/dmitrymomot/saaskit/pkg/token"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Password service errors
var (
	ErrInvalidCredentials = errors.New("auth: invalid email or password")
	ErrEmailAlreadyExists = errors.New("auth: email already registered")
	ErrIdentityNotFound   = errors.New("auth: identity not found")
	ErrTokenInvalid       = errors.New("auth: invalid token")
	ErrTokenExpired       = errors.New("auth: token expired")
	ErrEmailUnchanged     = errors.New("auth: new email is the same as current email")
)

// Token subjects for different authentication flows
const (
	SubjectPasswordReset = "password_reset"
	SubjectEmailVerify   = "email_verify" // for future use
	SubjectEmailChange   = "email_change" // for email update verification
)

// PasswordResetTokenPayload contains the data encoded in password reset tokens
type PasswordResetTokenPayload struct {
	ID       string `json:"id"`    // Identity ID
	Email    string `json:"email"` // User email
	Subject  string `json:"sub"`   // Token subject: SubjectPasswordReset
	ExpireAt int64  `json:"exp"`   // Unix timestamp
}

// PasswordStorage interface for password service
type PasswordStorage interface {
	// Identity operations
	CreateIdentity(ctx context.Context, identity *Identity) error
	GetIdentityByID(ctx context.Context, id uuid.UUID) (*Identity, error)
	GetIdentityByEmail(ctx context.Context, email string) (*Identity, error)
	DeleteIdentity(ctx context.Context, id uuid.UUID) error // For cleanup on failure

	// Password operations
	StorePasswordHash(ctx context.Context, identityID uuid.UUID, hash []byte) error
	GetPasswordHash(ctx context.Context, identityID uuid.UUID) ([]byte, error)
}

// PasswordService handles password-based authentication
type PasswordService struct {
	storage       PasswordStorage
	tokenSecret   string
	bcryptCost    int
	logger        *slog.Logger
	resetTokenTTL time.Duration // TTL for password reset tokens
}

// PasswordOption is a functional option for PasswordService
type PasswordOption func(*PasswordService)

// WithPasswordLogger sets a custom logger for the service
func WithPasswordLogger(logger *slog.Logger) PasswordOption {
	return func(s *PasswordService) {
		s.logger = logger
	}
}

// WithBcryptCost sets the bcrypt cost for password hashing
func WithBcryptCost(cost int) PasswordOption {
	return func(s *PasswordService) {
		s.bcryptCost = cost
	}
}

// WithResetTokenTTL sets the TTL for password reset tokens
func WithResetTokenTTL(ttl time.Duration) PasswordOption {
	return func(s *PasswordService) {
		s.resetTokenTTL = ttl
	}
}

// NewPasswordService creates a new password authentication service
func NewPasswordService(storage PasswordStorage, tokenSecret string, opts ...PasswordOption) *PasswordService {
	s := &PasswordService{
		storage:       storage,
		tokenSecret:   tokenSecret,
		bcryptCost:    bcrypt.DefaultCost,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)), // noop logger by default
		resetTokenTTL: 1 * time.Hour,                                  // Default 1 hour for reset tokens
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Register creates a new identity with email and password
func (s *PasswordService) Register(ctx context.Context, email, password string) (*Identity, error) {
	// Check if email already exists
	_, err := s.storage.GetIdentityByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}
	// Only proceed if error is "not found", otherwise return the error
	if !errors.Is(err, ErrIdentityNotFound) {
		return nil, fmt.Errorf("failed to check existing identity: %w", err)
	}

	// Hash password
	passwordBytes := []byte(password)
	defer clear(passwordBytes) // Clear password from memory
	hash, err := bcrypt.GenerateFromPassword(passwordBytes, s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create identity
	identity := &Identity{
		ID:         uuid.New(),
		Email:      email,
		AuthMethod: MethodPassword,
		IsVerified: false,
		CreatedAt:  time.Now(),
	}

	// Save identity
	if err := s.storage.CreateIdentity(ctx, identity); err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}

	// Save password hash
	if err := s.storage.StorePasswordHash(ctx, identity.ID, hash); err != nil {
		// Attempt to clean up the created identity
		if deleteErr := s.storage.DeleteIdentity(ctx, identity.ID); deleteErr != nil {
			s.logger.Error("failed to cleanup identity after password save failure",
				logger.UserID(identity.ID.String()),
				slog.String("email", identity.Email),
				logger.Error(deleteErr),
				logger.Component("password_service"),
			)
		}
		return nil, fmt.Errorf("failed to save password: %w", err)
	}

	return identity, nil
}

// Authenticate verifies email and password, returns identity if valid
func (s *PasswordService) Authenticate(ctx context.Context, email, password string) (*Identity, error) {
	// Get identity by email
	identity, err := s.storage.GetIdentityByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Get password hash
	hash, err := s.storage.GetPasswordHash(ctx, identity.ID)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	passwordBytes := []byte(password)
	defer clear(passwordBytes) // Clear password from memory
	if err := bcrypt.CompareHashAndPassword(hash, passwordBytes); err != nil {
		return nil, ErrInvalidCredentials
	}

	return identity, nil
}

// PasswordResetRequest contains data needed for password reset
type PasswordResetRequest struct {
	Email     string
	Token     string
	ExpiresAt time.Time
}

// ForgotPassword generates a password reset token for the given email
func (s *PasswordService) ForgotPassword(ctx context.Context, email string) (*PasswordResetRequest, error) {
	// Get identity by email
	identity, err := s.storage.GetIdentityByEmail(ctx, email)
	if err != nil {
		// Return the actual error - let the handler layer decide how to handle it
		// The handler should implement timing attack prevention by:
		// 1. Always returning success to the user
		// 2. Maintaining consistent response times
		// 3. Only sending emails for valid identities
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}

	// Create reset token payload
	expiresAt := time.Now().Add(s.resetTokenTTL)
	payload := PasswordResetTokenPayload{
		ID:       identity.ID.String(),
		Email:    email,
		Subject:  SubjectPasswordReset,
		ExpireAt: expiresAt.Unix(),
	}

	// Generate signed token
	tokenStr, err := token.GenerateToken(payload, s.tokenSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate reset token: %w", err)
	}

	return &PasswordResetRequest{
		Email:     email,
		Token:     tokenStr,
		ExpiresAt: expiresAt,
	}, nil
}

// ResetPassword resets the password using a valid reset token
func (s *PasswordService) ResetPassword(ctx context.Context, resetToken, newPassword string) (*Identity, error) {
	// Parse and validate token
	payload, err := token.ParseToken[PasswordResetTokenPayload](resetToken, s.tokenSecret)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// Check subject
	if payload.Subject != SubjectPasswordReset {
		return nil, ErrTokenInvalid
	}

	// Check expiration
	if time.Now().Unix() > payload.ExpireAt {
		return nil, ErrTokenExpired
	}

	// Get identity ID from token
	identityID, err := uuid.Parse(payload.ID)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.storage.StorePasswordHash(ctx, identityID, hash); err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	// Return updated identity
	return s.storage.GetIdentityByID(ctx, identityID)
}
