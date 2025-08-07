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

// PasswordAuthenticator defines password-based authentication operations
type PasswordAuthenticator interface {
	Register(ctx context.Context, email, password string) (*User, error)
	Authenticate(ctx context.Context, email, password string) (*User, error)
	ForgotPassword(ctx context.Context, email string) (*PasswordResetRequest, error)
	ResetPassword(ctx context.Context, resetToken, newPassword string) (*User, error)
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

// passwordService provides password-based authentication with configurable security requirements
type passwordService struct {
	storage          PasswordStorage
	tokenSecret      string
	bcryptCost       int
	logger           *slog.Logger
	resetTokenTTL    time.Duration
	passwordStrength validator.PasswordStrengthConfig

	// Hooks for extending password authentication behavior
	afterRegister func(ctx context.Context, user *User) error
	beforeLogin   func(ctx context.Context, email string) error
	afterLogin    func(ctx context.Context, user *User) error
}

type PasswordOption func(*passwordService)

// WithPasswordLogger sets a custom logger for the service
func WithPasswordLogger(logger *slog.Logger) PasswordOption {
	return func(s *passwordService) {
		s.logger = logger
	}
}

// WithBcryptCost sets the bcrypt cost for password hashing
func WithBcryptCost(cost int) PasswordOption {
	return func(s *passwordService) {
		s.bcryptCost = cost
	}
}

// WithResetTokenTTL sets the TTL for password reset tokens
func WithResetTokenTTL(ttl time.Duration) PasswordOption {
	return func(s *passwordService) {
		s.resetTokenTTL = ttl
	}
}

// WithPasswordStrength sets custom password strength requirements
func WithPasswordStrength(config validator.PasswordStrengthConfig) PasswordOption {
	return func(s *passwordService) {
		s.passwordStrength = config
	}
}

// WithAfterRegister sets a hook that runs after successful registration
func WithAfterRegister(fn func(context.Context, *User) error) PasswordOption {
	return func(s *passwordService) {
		s.afterRegister = fn
	}
}

// WithBeforeLogin sets a hook that runs before login attempt
func WithBeforeLogin(fn func(context.Context, string) error) PasswordOption {
	return func(s *passwordService) {
		s.beforeLogin = fn
	}
}

// WithAfterLogin sets a hook that runs after successful login
func WithAfterLogin(fn func(context.Context, *User) error) PasswordOption {
	return func(s *passwordService) {
		s.afterLogin = fn
	}
}

// NewPasswordService creates a new password authentication service
func NewPasswordService(storage PasswordStorage, tokenSecret string, opts ...PasswordOption) PasswordAuthenticator {
	s := &passwordService{
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
func (s *passwordService) Register(ctx context.Context, email, password string) (*User, error) {
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

	// Execute after register hook if set
	if s.afterRegister != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("afterRegister hook panicked",
						logger.UserID(user.ID.String()),
						slog.Any("panic", r),
						logger.Component("password"),
					)
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := s.afterRegister(ctx, user); err != nil {
				s.logger.Error("afterRegister hook failed",
					logger.UserID(user.ID.String()),
					logger.Error(err),
					logger.Component("password"),
				)
			}
		}()
	}

	return user, nil
}

// Authenticate verifies email and password, returns user if valid.
// Returns generic ErrInvalidCredentials for any failure to prevent user enumeration attacks.
func (s *passwordService) Authenticate(ctx context.Context, email, password string) (*User, error) {
	email = sanitizer.NormalizeEmail(email)

	// Execute before login hook if set
	if s.beforeLogin != nil {
		if err := s.beforeLogin(ctx, email); err != nil {
			return nil, fmt.Errorf("login blocked: %w", err)
		}
	}

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

	// Execute after login hook if set
	if s.afterLogin != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("afterLogin hook panicked",
						logger.UserID(user.ID.String()),
						slog.Any("panic", r),
						logger.Component("password"),
					)
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := s.afterLogin(ctx, user); err != nil {
				s.logger.Error("afterLogin hook failed",
					logger.UserID(user.ID.String()),
					logger.Error(err),
					logger.Component("password"),
				)
			}
		}()
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
func (s *passwordService) ForgotPassword(ctx context.Context, email string) (*PasswordResetRequest, error) {
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
func (s *passwordService) ResetPassword(ctx context.Context, resetToken, newPassword string) (*User, error) {
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
