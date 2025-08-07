package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/dmitrymomot/saaskit/pkg/token"
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
	ID       string `json:"id"`  // User ID
	OldEmail string `json:"old"` // Current email
	NewEmail string `json:"new"` // New email to change to
	Subject  string `json:"sub"` // SubjectEmailChange
	ExpireAt int64  `json:"exp"` // Unix timestamp
}

// UserStorage interface for user service
type UserStorage interface {
	// User operations
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUserEmail(ctx context.Context, id uuid.UUID, email string) error

	// Password operations
	GetPasswordHash(ctx context.Context, userID uuid.UUID) ([]byte, error)
	UpdatePasswordHash(ctx context.Context, userID uuid.UUID, hash []byte) error
}

// UserService handles authenticated user account management
type UserService struct {
	storage        UserStorage
	tokenSecret    string
	bcryptCost     int
	logger         *slog.Logger
	emailChangeTTL time.Duration // TTL for email change tokens
}

// UserOption is a functional option for UserService
type UserOption func(*UserService)

// WithUserLogger sets a custom logger for the service
func WithUserLogger(logger *slog.Logger) UserOption {
	return func(s *UserService) {
		s.logger = logger
	}
}

// WithUserBcryptCost sets the bcrypt cost for password hashing
func WithUserBcryptCost(cost int) UserOption {
	return func(s *UserService) {
		s.bcryptCost = cost
	}
}

// WithEmailChangeTTL sets the TTL for email change tokens
func WithEmailChangeTTL(ttl time.Duration) UserOption {
	return func(s *UserService) {
		s.emailChangeTTL = ttl
	}
}

// NewUserService creates a new user management service
func NewUserService(storage UserStorage, tokenSecret string, opts ...UserOption) *UserService {
	s := &UserService{
		storage:        storage,
		tokenSecret:    tokenSecret,
		bcryptCost:     bcrypt.DefaultCost,
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)), // noop logger by default
		emailChangeTTL: 1 * time.Hour,                                  // Default 1 hour for email change tokens
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// GetUser retrieves the user information for an authenticated user
func (s *UserService) GetUser(ctx context.Context, userID uuid.UUID) (*User, error) {
	user, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// ChangePassword updates the password for an authenticated user
func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	// Get current password hash
	hash, err := s.storage.GetPasswordHash(ctx, userID)
	if err != nil {
		return ErrUserNotFound
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
	if err := s.storage.UpdatePasswordHash(ctx, userID, newHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// RequestEmailChange initiates an email change process by generating a verification token
func (s *UserService) RequestEmailChange(ctx context.Context, userID uuid.UUID, newEmail, currentPassword string) (*EmailChangeRequest, error) {
	// Verify user exists and get current email
	user, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Check if email is the same - no need to change
	if user.Email == newEmail {
		return nil, ErrEmailUnchanged
	}

	// Check if new email is already taken
	_, err = s.storage.GetUserByEmail(ctx, newEmail)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}
	// Only proceed if error is "not found", otherwise return the error
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	// Verify current password for security
	hash, err := s.storage.GetPasswordHash(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(currentPassword)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Create email change token
	expiresAt := time.Now().Add(s.emailChangeTTL)
	payload := EmailChangeTokenPayload{
		ID:       userID.String(),
		OldEmail: user.Email,
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
		CurrentEmail: user.Email,
		NewEmail:     newEmail,
		Token:        tokenStr,
		ExpiresAt:    expiresAt,
	}, nil
}

// ConfirmEmailChange validates the email change token and updates the email in the database
func (s *UserService) ConfirmEmailChange(ctx context.Context, emailChangeToken string) (*User, error) {
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

	// Parse user ID
	userID, err := uuid.Parse(payload.ID)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// Verify the user still exists and email hasn't changed
	user, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Check if current email matches what was in the token
	if user.Email != payload.OldEmail {
		return nil, ErrTokenInvalid // Email was already changed
	}

	// Check if new email is still available
	_, err = s.storage.GetUserByEmail(ctx, payload.NewEmail)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}
	// Only proceed if error is "not found", otherwise return the error
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	// Update email
	if err := s.storage.UpdateUserEmail(ctx, userID, payload.NewEmail); err != nil {
		return nil, fmt.Errorf("failed to update email: %w", err)
	}

	// Return updated user
	return s.storage.GetUserByID(ctx, userID)
}
