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

	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
	"github.com/dmitrymomot/saaskit/pkg/token"
	"github.com/dmitrymomot/saaskit/pkg/validator"
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

// UserStorage defines the storage operations required for user management
type UserStorage interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUserEmail(ctx context.Context, id uuid.UUID, email string) error
	GetPasswordHash(ctx context.Context, userID uuid.UUID) ([]byte, error)
	UpdatePasswordHash(ctx context.Context, userID uuid.UUID, hash []byte) error
}

// UserService handles authenticated user account management
type UserService struct {
	storage          UserStorage
	tokenSecret      string
	bcryptCost       int
	logger           *slog.Logger
	emailChangeTTL   time.Duration
	passwordStrength validator.PasswordStrengthConfig
}

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

// WithUserPasswordStrength sets custom password strength requirements
func WithUserPasswordStrength(config validator.PasswordStrengthConfig) UserOption {
	return func(s *UserService) {
		s.passwordStrength = config
	}
}

// NewUserService creates a new user management service
func NewUserService(storage UserStorage, tokenSecret string, opts ...UserOption) *UserService {
	s := &UserService{
		storage:        storage,
		tokenSecret:    tokenSecret,
		bcryptCost:     bcrypt.DefaultCost,
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		emailChangeTTL: 1 * time.Hour,
		passwordStrength: validator.PasswordStrengthConfig{
			MinLength:      8,
			MaxLength:      128,
			MinCharClasses: 2, // Pragmatic default: requires only 2 character classes for better UX
		},
	}

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
	if err := validator.Apply(
		validator.StrongPassword("password", newPassword, s.passwordStrength),
		validator.NotCommonPassword("password", newPassword),
	); err != nil {
		return err
	}

	hash, err := s.storage.GetPasswordHash(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.storage.UpdatePasswordHash(ctx, userID, newHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// RequestEmailChange initiates an email change process by generating a verification token.
// Requires password verification for security to prevent unauthorized email changes.
func (s *UserService) RequestEmailChange(ctx context.Context, userID uuid.UUID, newEmail, currentPassword string) (*EmailChangeRequest, error) {
	newEmail = sanitizer.NormalizeEmail(newEmail)
	if err := validator.Apply(
		validator.ValidEmail("email", newEmail),
	); err != nil {
		return nil, err
	}

	user, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if user.Email == newEmail {
		return nil, ErrEmailUnchanged
	}

	// Check if new email is already taken, handling race conditions gracefully
	_, err = s.storage.GetUserByEmail(ctx, newEmail)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	hash, err := s.storage.GetPasswordHash(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(currentPassword)); err != nil {
		return nil, ErrInvalidCredentials
	}

	expiresAt := time.Now().Add(s.emailChangeTTL)
	payload := EmailChangeTokenPayload{
		ID:       userID.String(),
		OldEmail: user.Email,
		NewEmail: newEmail,
		Subject:  SubjectEmailChange,
		ExpireAt: expiresAt.Unix(),
	}

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
	payload, err := token.ParseToken[EmailChangeTokenPayload](emailChangeToken, s.tokenSecret)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	if payload.Subject != SubjectEmailChange {
		return nil, ErrTokenInvalid
	}

	if time.Now().Unix() > payload.ExpireAt {
		return nil, ErrTokenExpired
	}

	userID, err := uuid.Parse(payload.ID)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	user, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if user.Email != payload.OldEmail {
		return nil, ErrTokenInvalid // Prevents replay attacks after email already changed
	}

	// Re-check email availability to handle race conditions
	_, err = s.storage.GetUserByEmail(ctx, payload.NewEmail)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	if err := s.storage.UpdateUserEmail(ctx, userID, payload.NewEmail); err != nil {
		return nil, fmt.Errorf("failed to update email: %w", err)
	}

	return s.storage.GetUserByID(ctx, userID)
}
