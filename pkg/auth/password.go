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

	"github.com/dmitrymomot/saaskit/pkg/logger"
	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
	"github.com/dmitrymomot/saaskit/pkg/token"
	"github.com/dmitrymomot/saaskit/pkg/validator"
)

var (
	ErrInvalidCredentials = errors.New("auth: invalid email or password")
	ErrEmailAlreadyExists = errors.New("auth: email already registered")
	ErrUserNotFound       = errors.New("auth: user not found")
	ErrTokenInvalid       = errors.New("auth: invalid token")
	ErrTokenExpired       = errors.New("auth: token expired")
	ErrEmailUnchanged     = errors.New("auth: new email is the same as current email")
)

const (
	SubjectPasswordReset = "password_reset"
	SubjectEmailVerify   = "email_verify" // for future use
	SubjectEmailChange   = "email_change" // for email update verification
)

// PasswordResetTokenPayload contains the data encoded in password reset tokens
type PasswordResetTokenPayload struct {
	ID       string `json:"id"`    // User ID
	Email    string `json:"email"` // User email
	Subject  string `json:"sub"`   // Token subject: SubjectPasswordReset
	ExpireAt int64  `json:"exp"`   // Unix timestamp
}

// PasswordStorage defines the storage operations required for password authentication
type PasswordStorage interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	StorePasswordHash(ctx context.Context, userID uuid.UUID, hash []byte) error
	GetPasswordHash(ctx context.Context, userID uuid.UUID) ([]byte, error)
}

// PasswordService provides password-based authentication with configurable security requirements
type PasswordService struct {
	storage          PasswordStorage
	tokenSecret      string
	bcryptCost       int
	logger           *slog.Logger
	resetTokenTTL    time.Duration
	passwordStrength validator.PasswordStrengthConfig
}

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

// WithPasswordStrength sets custom password strength requirements
func WithPasswordStrength(config validator.PasswordStrengthConfig) PasswordOption {
	return func(s *PasswordService) {
		s.passwordStrength = config
	}
}

// NewPasswordService creates a new password authentication service
func NewPasswordService(storage PasswordStorage, tokenSecret string, opts ...PasswordOption) *PasswordService {
	s := &PasswordService{
		storage:       storage,
		tokenSecret:   tokenSecret,
		bcryptCost:    bcrypt.DefaultCost,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		resetTokenTTL: 1 * time.Hour,
		passwordStrength: validator.PasswordStrengthConfig{
			MinLength:      8,
			MaxLength:      128,
			MinCharClasses: 2, // Pragmatic default: requires only 2 character classes for better UX while maintaining security
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Register creates a new user with email and password
func (s *PasswordService) Register(ctx context.Context, email, password string) (*User, error) {
	email = sanitizer.NormalizeEmail(email)

	if err := validator.Apply(
		validator.ValidEmail("email", email),
		validator.StrongPassword("password", password, s.passwordStrength),
		validator.NotCommonPassword("password", password),
	); err != nil {
		return nil, err
	}

	_, err := s.storage.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		ID:         uuid.New(),
		Email:      email,
		AuthMethod: MethodPassword,
		IsVerified: false,
		CreatedAt:  time.Now(),
	}

	if err := s.storage.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := s.storage.StorePasswordHash(ctx, user.ID, hash); err != nil {
		// Clean up the user record if password storage fails to maintain consistency
		if deleteErr := s.storage.DeleteUser(ctx, user.ID); deleteErr != nil {
			s.logger.Error("failed to cleanup user after password save failure",
				logger.UserID(user.ID.String()),
				slog.String("email", user.Email),
				logger.Error(deleteErr),
				logger.Component("password"),
			)
		}
		return nil, fmt.Errorf("failed to save password: %w", err)
	}

	return user, nil
}

// Authenticate verifies email and password, returns user if valid.
// Returns generic ErrInvalidCredentials for any failure to prevent user enumeration attacks.
func (s *PasswordService) Authenticate(ctx context.Context, email, password string) (*User, error) {
	email = sanitizer.NormalizeEmail(email)

	user, err := s.storage.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	hash, err := s.storage.GetPasswordHash(ctx, user.ID)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// PasswordResetRequest contains data needed for password reset
type PasswordResetRequest struct {
	Email     string
	Token     string
	ExpiresAt time.Time
}

// ForgotPassword generates a password reset token for the given email.
// Note: Handler should implement timing attack prevention by always returning success to users.
func (s *PasswordService) ForgotPassword(ctx context.Context, email string) (*PasswordResetRequest, error) {
	email = sanitizer.NormalizeEmail(email)

	user, err := s.storage.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	expiresAt := time.Now().Add(s.resetTokenTTL)
	payload := PasswordResetTokenPayload{
		ID:       user.ID.String(),
		Email:    email,
		Subject:  SubjectPasswordReset,
		ExpireAt: expiresAt.Unix(),
	}

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
func (s *PasswordService) ResetPassword(ctx context.Context, resetToken, newPassword string) (*User, error) {
	if err := validator.Apply(
		validator.StrongPassword("password", newPassword, s.passwordStrength),
		validator.NotCommonPassword("password", newPassword),
	); err != nil {
		return nil, err
	}

	payload, err := token.ParseToken[PasswordResetTokenPayload](resetToken, s.tokenSecret)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	if payload.Subject != SubjectPasswordReset {
		return nil, ErrTokenInvalid
	}

	if time.Now().Unix() > payload.ExpireAt {
		return nil, ErrTokenExpired
	}

	userID, err := uuid.Parse(payload.ID)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.storage.StorePasswordHash(ctx, userID, hash); err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	return s.storage.GetUserByID(ctx, userID)
}
