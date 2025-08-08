package auth

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/dmitrymomot/saaskit/pkg/token"
	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestNewUserService(t *testing.T) {
	t.Parallel()

	storage := &MockUserStorage{}
	tokenSecret := "test-secret"

	t.Run("creates service with defaults", func(t *testing.T) {
		t.Parallel()

		svc := NewUserService(storage, tokenSecret)
		require.NotNil(t, svc)

		// Cast to implementation to verify defaults
		impl := svc.(*userService)
		assert.Equal(t, storage, impl.storage)
		assert.Equal(t, tokenSecret, impl.tokenSecret)
		assert.Equal(t, bcrypt.DefaultCost, impl.bcryptCost)
		assert.Equal(t, 1*time.Hour, impl.emailChangeTTL)
		assert.NotNil(t, impl.logger)
		assert.Equal(t, 8, impl.passwordStrength.MinLength)
		assert.Equal(t, 128, impl.passwordStrength.MaxLength)
		assert.Equal(t, 2, impl.passwordStrength.MinCharClasses)
	})

	t.Run("applies options correctly", func(t *testing.T) {
		t.Parallel()

		logger := slog.Default()
		customCost := 6
		customTTL := 2 * time.Hour
		customStrength := validator.PasswordStrengthConfig{
			MinLength:      12,
			MaxLength:      256,
			MinCharClasses: 3,
		}

		svc := NewUserService(storage, tokenSecret,
			WithUserLogger(logger),
			WithUserBcryptCost(customCost),
			WithEmailChangeTTL(customTTL),
			WithUserPasswordStrength(customStrength),
		)

		impl := svc.(*userService)
		assert.Equal(t, logger, impl.logger)
		assert.Equal(t, customCost, impl.bcryptCost)
		assert.Equal(t, customTTL, impl.emailChangeTTL)
		assert.Equal(t, customStrength, impl.passwordStrength)
	})
}

func TestUserService_GetUser(t *testing.T) {
	t.Parallel()

	t.Run("retrieves user successfully", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, "secret")

		userID := uuid.New()
		expectedUser := &User{
			ID:         userID,
			Email:      "test@example.com",
			AuthMethod: MethodPassword,
			IsVerified: true,
			CreatedAt:  time.Now(),
		}

		storage.On("GetUserByID", mock.Anything, userID).Return(expectedUser, nil)

		ctx := context.Background()
		user, err := svc.GetUser(ctx, userID)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Email, user.Email)

		storage.AssertExpectations(t)
	})

	t.Run("handles storage errors", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, "secret")

		userID := uuid.New()

		storage.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("db error"))

		ctx := context.Background()
		user, err := svc.GetUser(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "failed to get user")

		storage.AssertExpectations(t)
	})
}

func TestUserService_ChangePassword(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	t.Run("changes password successfully", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		oldPassword := "OldPassword123!"
		newPassword := "NewPassword456!"

		// Pre-hash the old password for the test
		oldHash, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
		require.NoError(t, err)

		storage.On("GetPasswordHash", mock.Anything, userID).Return(oldHash, nil)
		storage.On("UpdatePasswordHash", mock.Anything, userID, mock.AnythingOfType("[]uint8")).Return(nil)

		ctx := context.Background()
		err = svc.ChangePassword(ctx, userID, oldPassword, newPassword)

		require.NoError(t, err)

		storage.AssertExpectations(t)
	})

	t.Run("validates new password strength", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		oldPassword := "OldPassword123!"

		weakPasswords := []string{
			"weak",
			"password",
			"12345678",
		}

		for _, newPassword := range weakPasswords {
			t.Run("weak password: "+newPassword, func(t *testing.T) {
				ctx := context.Background()
				err := svc.ChangePassword(ctx, userID, oldPassword, newPassword)

				assert.Error(t, err)
			})
		}

		storage.AssertExpectations(t)
	})

	t.Run("rejects incorrect current password", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		wrongOldPassword := "WrongPassword123!"
		newPassword := "NewPassword456!"

		// Hash a different password
		actualHash, err := bcrypt.GenerateFromPassword([]byte("ActualPassword123!"), bcrypt.DefaultCost)
		require.NoError(t, err)

		storage.On("GetPasswordHash", mock.Anything, userID).Return(actualHash, nil)

		ctx := context.Background()
		err = svc.ChangePassword(ctx, userID, wrongOldPassword, newPassword)

		assert.Equal(t, ErrInvalidCredentials, err)

		storage.AssertExpectations(t)
	})

	t.Run("handles user not found", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		oldPassword := "OldPassword123!"
		newPassword := "NewPassword456!"

		storage.On("GetPasswordHash", mock.Anything, userID).Return(nil, errors.New("user not found"))

		ctx := context.Background()
		err := svc.ChangePassword(ctx, userID, oldPassword, newPassword)

		assert.Equal(t, ErrUserNotFound, err)

		storage.AssertExpectations(t)
	})

	t.Run("handles storage errors", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		oldPassword := "OldPassword123!"
		newPassword := "NewPassword456!"

		oldHash, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
		require.NoError(t, err)

		storage.On("GetPasswordHash", mock.Anything, userID).Return(oldHash, nil)
		storage.On("UpdatePasswordHash", mock.Anything, userID, mock.AnythingOfType("[]uint8")).Return(errors.New("update error"))

		ctx := context.Background()
		err = svc.ChangePassword(ctx, userID, oldPassword, newPassword)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update password")

		storage.AssertExpectations(t)
	})

	t.Run("executes hooks", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		beforeUpdateCalled := false
		afterUpdateCalled := false

		beforeUpdate := func(ctx context.Context, userID uuid.UUID) error {
			beforeUpdateCalled = true
			return nil
		}

		afterUpdate := func(ctx context.Context, user *User) error {
			assert.NotNil(t, user)
			afterUpdateCalled = true
			return nil
		}

		svc := NewUserService(storage, tokenSecret,
			WithBeforeUpdate(beforeUpdate),
			WithAfterUpdate(afterUpdate),
		)

		userID := uuid.New()
		oldPassword := "OldPassword123!"
		newPassword := "NewPassword456!"
		user := &User{ID: userID, Email: "test@example.com"}

		oldHash, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
		require.NoError(t, err)

		storage.On("GetPasswordHash", mock.Anything, userID).Return(oldHash, nil)
		storage.On("UpdatePasswordHash", mock.Anything, userID, mock.AnythingOfType("[]uint8")).Return(nil)
		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)

		ctx := context.Background()
		err = svc.ChangePassword(ctx, userID, oldPassword, newPassword)

		require.NoError(t, err)
		assert.True(t, beforeUpdateCalled)
		assert.True(t, afterUpdateCalled)

		storage.AssertExpectations(t)
	})

	t.Run("blocks update when beforeUpdate hook fails", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}

		beforeUpdate := func(ctx context.Context, userID uuid.UUID) error {
			return errors.New("blocked by hook")
		}

		svc := NewUserService(storage, tokenSecret, WithBeforeUpdate(beforeUpdate))

		userID := uuid.New()
		oldPassword := "OldPassword123!"
		newPassword := "NewPassword456!"

		ctx := context.Background()
		err := svc.ChangePassword(ctx, userID, oldPassword, newPassword)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update blocked")

		storage.AssertExpectations(t)
	})
}

func TestUserService_RequestEmailChange(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	t.Run("generates email change token successfully", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		currentEmail := "current@example.com"
		newEmail := "new@example.com"
		password := "ValidPassword123!"

		user := &User{ID: userID, Email: currentEmail}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)

		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)
		storage.On("GetUserByEmail", mock.Anything, newEmail).Return(nil, ErrUserNotFound) // New email available
		storage.On("GetPasswordHash", mock.Anything, userID).Return(passwordHash, nil)

		ctx := context.Background()
		req, err := svc.RequestEmailChange(ctx, userID, newEmail, password)

		require.NoError(t, err)
		require.NotNil(t, req)
		assert.Equal(t, currentEmail, req.CurrentEmail)
		assert.Equal(t, newEmail, req.NewEmail)
		assert.NotEmpty(t, req.Token)
		assert.True(t, req.ExpiresAt.After(time.Now()))
		assert.True(t, req.ExpiresAt.Before(time.Now().Add(2*time.Hour)))

		storage.AssertExpectations(t)
	})

	t.Run("normalizes new email addresses", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		currentEmail := "current@example.com"
		inputNewEmail := "  New.Email+Tag@EXAMPLE.COM  "
		normalizedNewEmail := "new.email+tag@example.com"
		password := "ValidPassword123!"

		user := &User{ID: userID, Email: currentEmail}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)

		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)
		storage.On("GetUserByEmail", mock.Anything, normalizedNewEmail).Return(nil, ErrUserNotFound)
		storage.On("GetPasswordHash", mock.Anything, userID).Return(passwordHash, nil)

		ctx := context.Background()
		req, err := svc.RequestEmailChange(ctx, userID, inputNewEmail, password)

		require.NoError(t, err)
		assert.Equal(t, normalizedNewEmail, req.NewEmail)

		storage.AssertExpectations(t)
	})

	t.Run("validates new email format", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		password := "ValidPassword123!"

		invalidEmails := []string{
			"",
			"not-an-email",
			"user@",
			"@example.com",
		}

		for _, email := range invalidEmails {
			t.Run("invalid email: "+email, func(t *testing.T) {
				ctx := context.Background()
				req, err := svc.RequestEmailChange(ctx, userID, email, password)

				assert.Error(t, err)
				assert.Nil(t, req)
			})
		}

		storage.AssertExpectations(t)
	})

	t.Run("rejects unchanged email", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		email := "same@example.com"
		password := "ValidPassword123!"

		user := &User{ID: userID, Email: email}

		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)

		ctx := context.Background()
		req, err := svc.RequestEmailChange(ctx, userID, email, password)

		assert.Equal(t, ErrEmailUnchanged, err)
		assert.Nil(t, req)

		storage.AssertExpectations(t)
	})

	t.Run("rejects existing email", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		currentEmail := "current@example.com"
		newEmail := "existing@example.com"
		password := "ValidPassword123!"

		user := &User{ID: userID, Email: currentEmail}
		existingUser := &User{ID: uuid.New(), Email: newEmail}

		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)
		storage.On("GetUserByEmail", mock.Anything, newEmail).Return(existingUser, nil) // Email already taken

		ctx := context.Background()
		req, err := svc.RequestEmailChange(ctx, userID, newEmail, password)

		assert.Equal(t, ErrEmailAlreadyExists, err)
		assert.Nil(t, req)

		storage.AssertExpectations(t)
	})

	t.Run("rejects incorrect password", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		currentEmail := "current@example.com"
		newEmail := "new@example.com"
		wrongPassword := "WrongPassword123!"

		user := &User{ID: userID, Email: currentEmail}
		// Hash a different password
		actualPasswordHash, err := bcrypt.GenerateFromPassword([]byte("ActualPassword123!"), bcrypt.DefaultCost)
		require.NoError(t, err)

		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)
		storage.On("GetUserByEmail", mock.Anything, newEmail).Return(nil, ErrUserNotFound)
		storage.On("GetPasswordHash", mock.Anything, userID).Return(actualPasswordHash, nil)

		ctx := context.Background()
		req, err := svc.RequestEmailChange(ctx, userID, newEmail, wrongPassword)

		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, req)

		storage.AssertExpectations(t)
	})

	t.Run("handles user not found", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		newEmail := "new@example.com"
		password := "ValidPassword123!"

		storage.On("GetUserByID", mock.Anything, userID).Return(nil, ErrUserNotFound)

		ctx := context.Background()
		req, err := svc.RequestEmailChange(ctx, userID, newEmail, password)

		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, req)

		storage.AssertExpectations(t)
	})

	t.Run("handles storage errors", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		newEmail := "new@example.com"
		password := "ValidPassword123!"

		t.Run("user lookup error", func(t *testing.T) {
			t.Parallel()

			storage := &MockUserStorage{}
			svc := NewUserService(storage, tokenSecret)

			storage.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("db error"))

			ctx := context.Background()
			req, err := svc.RequestEmailChange(ctx, userID, newEmail, password)

			assert.Equal(t, ErrUserNotFound, err)
			assert.Nil(t, req)

			storage.AssertExpectations(t)
		})

		t.Run("email availability check error", func(t *testing.T) {
			t.Parallel()

			storage := &MockUserStorage{}
			svc := NewUserService(storage, tokenSecret)

			user := &User{ID: userID, Email: "current@example.com"}

			storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)
			storage.On("GetUserByEmail", mock.Anything, newEmail).Return(nil, errors.New("db error"))

			ctx := context.Background()
			req, err := svc.RequestEmailChange(ctx, userID, newEmail, password)

			assert.Error(t, err)
			assert.Nil(t, req)
			assert.Contains(t, err.Error(), "failed to check existing email")

			storage.AssertExpectations(t)
		})
	})
}

func TestUserService_ConfirmEmailChange(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	createValidEmailChangeToken := func(userID uuid.UUID, oldEmail, newEmail string, expiresIn time.Duration) string {
		payload := EmailChangeTokenPayload{
			ID:       userID.String(),
			OldEmail: oldEmail,
			NewEmail: newEmail,
			Subject:  SubjectEmailChange,
			ExpireAt: time.Now().Add(expiresIn).Unix(),
		}
		tokenStr, err := token.GenerateToken(payload, tokenSecret)
		require.NoError(t, err)
		return tokenStr
	}

	t.Run("confirms email change successfully", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		oldEmail := "old@example.com"
		newEmail := "new@example.com"

		user := &User{ID: userID, Email: oldEmail}
		updatedUser := &User{ID: userID, Email: newEmail}

		validToken := createValidEmailChangeToken(userID, oldEmail, newEmail, 1*time.Hour)

		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil).Once()
		storage.On("GetUserByEmail", mock.Anything, newEmail).Return(nil, ErrUserNotFound) // New email still available
		storage.On("UpdateUserEmail", mock.Anything, userID, newEmail).Return(nil)
		storage.On("GetUserByID", mock.Anything, userID).Return(updatedUser, nil).Once()

		ctx := context.Background()
		resultUser, err := svc.ConfirmEmailChange(ctx, validToken)

		require.NoError(t, err)
		require.NotNil(t, resultUser)
		assert.Equal(t, userID, resultUser.ID)
		assert.Equal(t, newEmail, resultUser.Email)

		storage.AssertExpectations(t)
	})

	t.Run("rejects invalid tokens", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		testCases := []struct {
			name  string
			token string
		}{
			{"empty token", ""},
			{"malformed token", "invalid-token"},
			{"wrong secret", func() string {
				payload := EmailChangeTokenPayload{
					ID:       uuid.New().String(),
					OldEmail: "old@example.com",
					NewEmail: "new@example.com",
					Subject:  SubjectEmailChange,
				}
				tokenStr, _ := token.GenerateToken(payload, "wrong-secret")
				return tokenStr
			}()},
			{"wrong subject", func() string {
				payload := EmailChangeTokenPayload{
					ID:       uuid.New().String(),
					OldEmail: "old@example.com",
					NewEmail: "new@example.com",
					Subject:  "wrong-subject",
				}
				tokenStr, _ := token.GenerateToken(payload, tokenSecret)
				return tokenStr
			}()},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				user, err := svc.ConfirmEmailChange(ctx, tc.token)

				assert.Equal(t, ErrTokenInvalid, err)
				assert.Nil(t, user)
			})
		}

		storage.AssertExpectations(t)
	})

	t.Run("rejects expired tokens", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		oldEmail := "old@example.com"
		newEmail := "new@example.com"

		// Create expired token
		expiredToken := createValidEmailChangeToken(userID, oldEmail, newEmail, -1*time.Hour)

		ctx := context.Background()
		user, err := svc.ConfirmEmailChange(ctx, expiredToken)

		assert.Equal(t, ErrTokenExpired, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
	})

	t.Run("prevents replay attacks after email already changed", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		oldEmail := "old@example.com"
		newEmail := "new@example.com"

		// User already has the new email (email was already changed)
		user := &User{ID: userID, Email: newEmail}

		validToken := createValidEmailChangeToken(userID, oldEmail, newEmail, 1*time.Hour)

		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)

		ctx := context.Background()
		resultUser, err := svc.ConfirmEmailChange(ctx, validToken)

		assert.Equal(t, ErrTokenInvalid, err)
		assert.Nil(t, resultUser)

		storage.AssertExpectations(t)
	})

	t.Run("handles race condition when new email becomes unavailable", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		userID := uuid.New()
		oldEmail := "old@example.com"
		newEmail := "new@example.com"

		user := &User{ID: userID, Email: oldEmail}
		existingUser := &User{ID: uuid.New(), Email: newEmail}

		validToken := createValidEmailChangeToken(userID, oldEmail, newEmail, 1*time.Hour)

		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)
		storage.On("GetUserByEmail", mock.Anything, newEmail).Return(existingUser, nil) // Email taken by someone else

		ctx := context.Background()
		resultUser, err := svc.ConfirmEmailChange(ctx, validToken)

		assert.Equal(t, ErrEmailAlreadyExists, err)
		assert.Nil(t, resultUser)

		storage.AssertExpectations(t)
	})

	t.Run("handles invalid user ID in token", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		svc := NewUserService(storage, tokenSecret)

		// Create token with invalid UUID
		payload := EmailChangeTokenPayload{
			ID:       "invalid-uuid",
			OldEmail: "old@example.com",
			NewEmail: "new@example.com",
			Subject:  SubjectEmailChange,
			ExpireAt: time.Now().Add(1 * time.Hour).Unix(),
		}
		invalidToken, err := token.GenerateToken(payload, tokenSecret)
		require.NoError(t, err)

		ctx := context.Background()
		user, err := svc.ConfirmEmailChange(ctx, invalidToken)

		assert.Equal(t, ErrTokenInvalid, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
	})

	t.Run("handles storage errors", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		oldEmail := "old@example.com"
		newEmail := "new@example.com"
		validToken := createValidEmailChangeToken(userID, oldEmail, newEmail, 1*time.Hour)

		t.Run("user lookup error", func(t *testing.T) {
			t.Parallel()

			storage := &MockUserStorage{}
			svc := NewUserService(storage, tokenSecret)

			storage.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("db error"))

			ctx := context.Background()
			user, err := svc.ConfirmEmailChange(ctx, validToken)

			assert.Equal(t, ErrUserNotFound, err)
			assert.Nil(t, user)

			storage.AssertExpectations(t)
		})

		t.Run("email update error", func(t *testing.T) {
			t.Parallel()

			storage := &MockUserStorage{}
			svc := NewUserService(storage, tokenSecret)

			user := &User{ID: userID, Email: oldEmail}

			storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)
			storage.On("GetUserByEmail", mock.Anything, newEmail).Return(nil, ErrUserNotFound)
			storage.On("UpdateUserEmail", mock.Anything, userID, newEmail).Return(errors.New("update error"))

			ctx := context.Background()
			resultUser, err := svc.ConfirmEmailChange(ctx, validToken)

			assert.Error(t, err)
			assert.Nil(t, resultUser)
			assert.Contains(t, err.Error(), "failed to update email")

			storage.AssertExpectations(t)
		})
	})

	t.Run("executes hooks", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		beforeUpdateCalled := false
		afterUpdateCalled := false

		beforeUpdate := func(ctx context.Context, userID uuid.UUID) error {
			beforeUpdateCalled = true
			return nil
		}

		afterUpdate := func(ctx context.Context, user *User) error {
			assert.NotNil(t, user)
			afterUpdateCalled = true
			return nil
		}

		svc := NewUserService(storage, tokenSecret,
			WithBeforeUpdate(beforeUpdate),
			WithAfterUpdate(afterUpdate),
		)

		userID := uuid.New()
		oldEmail := "old@example.com"
		newEmail := "new@example.com"

		user := &User{ID: userID, Email: oldEmail}
		updatedUser := &User{ID: userID, Email: newEmail}

		validToken := createValidEmailChangeToken(userID, oldEmail, newEmail, 1*time.Hour)

		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil).Once()
		storage.On("GetUserByEmail", mock.Anything, newEmail).Return(nil, ErrUserNotFound)
		storage.On("UpdateUserEmail", mock.Anything, userID, newEmail).Return(nil)
		storage.On("GetUserByID", mock.Anything, userID).Return(updatedUser, nil).Once()

		ctx := context.Background()
		_, err := svc.ConfirmEmailChange(ctx, validToken)

		require.NoError(t, err)
		assert.True(t, beforeUpdateCalled)
		assert.True(t, afterUpdateCalled)

		storage.AssertExpectations(t)
	})

	t.Run("blocks change when beforeUpdate hook fails", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}

		beforeUpdate := func(ctx context.Context, userID uuid.UUID) error {
			return errors.New("blocked by hook")
		}

		svc := NewUserService(storage, tokenSecret, WithBeforeUpdate(beforeUpdate))

		userID := uuid.New()
		oldEmail := "old@example.com"
		newEmail := "new@example.com"

		user := &User{ID: userID, Email: oldEmail}
		validToken := createValidEmailChangeToken(userID, oldEmail, newEmail, 1*time.Hour)

		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)
		storage.On("GetUserByEmail", mock.Anything, newEmail).Return(nil, ErrUserNotFound)

		ctx := context.Background()
		resultUser, err := svc.ConfirmEmailChange(ctx, validToken)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update blocked")
		assert.Nil(t, resultUser)

		storage.AssertExpectations(t)
	})
}

func TestEmailChangeTokenSecurity(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	t.Run("email change tokens contain expected fields", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		oldEmail := "old@example.com"
		newEmail := "new@example.com"

		payload := EmailChangeTokenPayload{
			ID:       userID.String(),
			OldEmail: oldEmail,
			NewEmail: newEmail,
			Subject:  SubjectEmailChange,
			ExpireAt: time.Now().Add(1 * time.Hour).Unix(),
		}

		tokenStr, err := token.GenerateToken(payload, tokenSecret)
		require.NoError(t, err)

		// Parse the token back
		parsedPayload, err := token.ParseToken[EmailChangeTokenPayload](tokenStr, tokenSecret)
		require.NoError(t, err)

		assert.Equal(t, payload.ID, parsedPayload.ID)
		assert.Equal(t, payload.OldEmail, parsedPayload.OldEmail)
		assert.Equal(t, payload.NewEmail, parsedPayload.NewEmail)
		assert.Equal(t, payload.Subject, parsedPayload.Subject)
		assert.Equal(t, payload.ExpireAt, parsedPayload.ExpireAt)
	})

	t.Run("custom TTL is respected", func(t *testing.T) {
		t.Parallel()

		storage := &MockUserStorage{}
		customTTL := 30 * time.Minute
		svc := NewUserService(storage, tokenSecret, WithEmailChangeTTL(customTTL))

		userID := uuid.New()
		currentEmail := "current@example.com"
		newEmail := "new@example.com"
		password := "ValidPassword123!"

		user := &User{ID: userID, Email: currentEmail}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)

		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)
		storage.On("GetUserByEmail", mock.Anything, newEmail).Return(nil, ErrUserNotFound)
		storage.On("GetPasswordHash", mock.Anything, userID).Return(passwordHash, nil)

		ctx := context.Background()
		req, err := svc.RequestEmailChange(ctx, userID, newEmail, password)

		require.NoError(t, err)
		assert.True(t, req.ExpiresAt.Before(time.Now().Add(35*time.Minute)))
		assert.True(t, req.ExpiresAt.After(time.Now().Add(25*time.Minute)))

		storage.AssertExpectations(t)
	})
}

// Test that the service correctly implements the interface
func TestUserServiceInterface(t *testing.T) {
	t.Parallel()

	storage := &MockUserStorage{}
	var svc UserManager = NewUserService(storage, "secret")

	require.NotNil(t, svc)
	// If this compiles, the interface is correctly implemented
}
