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

// UserManager defines user management operations
type UserManager interface {
	GetUser(ctx context.Context, userID uuid.UUID) (*User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
	RequestEmailChange(ctx context.Context, userID uuid.UUID, newEmail, currentPassword string) (*EmailChangeRequest, error)
	ConfirmEmailChange(ctx context.Context, emailChangeToken string) (*User, error)
}

// UserStorage defines the storage operations required for user management
type UserStorage interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUserEmail(ctx context.Context, id uuid.UUID, email string) error
	GetPasswordHash(ctx context.Context, userID uuid.UUID) ([]byte, error)
	UpdatePasswordHash(ctx context.Context, userID uuid.UUID, hash []byte) error
}

// userService handles authenticated user account management
type userService struct {
	storage          UserStorage
	tokenSecret      string
	bcryptCost       int
	logger           *slog.Logger
	emailChangeTTL   time.Duration
	passwordStrength validator.PasswordStrengthConfig

	// Hooks for extending user management behavior
	beforeUpdate func(ctx context.Context, userID uuid.UUID) error
	afterUpdate  func(ctx context.Context, user *User) error
	afterDelete  func(ctx context.Context, userID uuid.UUID) error
}

type UserOption func(*userService)

// WithUserLogger sets a custom logger for the service
func WithUserLogger(logger *slog.Logger) UserOption {
	return func(s *userService) {
		s.logger = logger
	}
}

// WithUserBcryptCost sets the bcrypt cost for password hashing
func WithUserBcryptCost(cost int) UserOption {
	return func(s *userService) {
		s.bcryptCost = cost
	}
}

// WithEmailChangeTTL sets the TTL for email change tokens
func WithEmailChangeTTL(ttl time.Duration) UserOption {
	return func(s *userService) {
		s.emailChangeTTL = ttl
	}
}

// WithUserPasswordStrength sets custom password strength requirements
func WithUserPasswordStrength(config validator.PasswordStrengthConfig) UserOption {
	return func(s *userService) {
		s.passwordStrength = config
	}
}

// WithBeforeUpdate sets a hook that runs before any user update
func WithBeforeUpdate(fn func(context.Context, uuid.UUID) error) UserOption {
	return func(s *userService) {
		s.beforeUpdate = fn
	}
}

// WithAfterUpdate sets a hook that runs after successful user update
func WithAfterUpdate(fn func(context.Context, *User) error) UserOption {
	return func(s *userService) {
		s.afterUpdate = fn
	}
}

// WithAfterDelete sets a hook that runs after user deletion
func WithAfterDelete(fn func(context.Context, uuid.UUID) error) UserOption {
	return func(s *userService) {
		s.afterDelete = fn
	}
}

// NewUserService creates a new user management service
func NewUserService(storage UserStorage, tokenSecret string, opts ...UserOption) UserManager {
	s := &userService{
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
func (s *userService) GetUser(ctx context.Context, userID uuid.UUID) (*User, error) {
	user, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// ChangePassword updates the password for an authenticated user
func (s *userService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	// Execute before update hook if set
	if s.beforeUpdate != nil {
		if err := s.beforeUpdate(ctx, userID); err != nil {
			return fmt.Errorf("update blocked: %w", err)
		}
	}
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

	// Execute after update hook if set
	if s.afterUpdate != nil {
		user, _ := s.storage.GetUserByID(ctx, userID)
		if user != nil {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						s.logger.Error("afterUpdate hook panicked",
							slog.String("user_id", userID.String()),
							slog.Any("panic", r),
							slog.String("component", "user"),
						)
					}
				}()

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				if err := s.afterUpdate(ctx, user); err != nil {
					s.logger.Error("afterUpdate hook failed",
						slog.String("user_id", userID.String()),
						slog.Any("error", err),
						slog.String("component", "user"),
					)
				}
			}()
		}
	}

	return nil
}

// RequestEmailChange initiates an email change process by generating a verification token.
// Requires password verification for security to prevent unauthorized email changes.
func (s *userService) RequestEmailChange(ctx context.Context, userID uuid.UUID, newEmail, currentPassword string) (*EmailChangeRequest, error) {
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
func (s *userService) ConfirmEmailChange(ctx context.Context, emailChangeToken string) (*User, error) {
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

	// Execute before update hook if set
	if s.beforeUpdate != nil {
		if err := s.beforeUpdate(ctx, userID); err != nil {
			return nil, fmt.Errorf("update blocked: %w", err)
		}
	}

	if err := s.storage.UpdateUserEmail(ctx, userID, payload.NewEmail); err != nil {
		return nil, fmt.Errorf("failed to update email: %w", err)
	}

	updatedUser, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Execute after update hook if set
	if s.afterUpdate != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("afterUpdate hook panicked",
						slog.String("user_id", userID.String()),
						slog.Any("panic", r),
						slog.String("component", "user"),
					)
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := s.afterUpdate(ctx, updatedUser); err != nil {
				s.logger.Error("afterUpdate hook failed",
					slog.String("user_id", userID.String()),
					slog.Any("error", err),
					slog.String("component", "user"),
				)
			}
		}()
	}

	return updatedUser, nil
}
