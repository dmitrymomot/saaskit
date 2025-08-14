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

func TestNewPasswordService(t *testing.T) {
	t.Parallel()

	storage := &MockPasswordStorage{}
	tokenSecret := "test-secret"

	t.Run("creates service with defaults", func(t *testing.T) {
		t.Parallel()

		svc := NewPasswordService(storage, tokenSecret)
		require.NotNil(t, svc)

		// Cast to implementation to verify defaults
		impl := svc.(*passwordService)
		assert.Equal(t, storage, impl.storage)
		assert.Equal(t, tokenSecret, impl.tokenSecret)
		assert.Equal(t, bcrypt.DefaultCost, impl.bcryptCost)
		assert.Equal(t, 1*time.Hour, impl.resetTokenTTL)
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

		svc := NewPasswordService(storage, tokenSecret,
			WithPasswordLogger(logger),
			WithBcryptCost(customCost),
			WithResetTokenTTL(customTTL),
			WithPasswordStrength(customStrength),
		)

		impl := svc.(*passwordService)
		assert.Equal(t, logger, impl.logger)
		assert.Equal(t, customCost, impl.bcryptCost)
		assert.Equal(t, customTTL, impl.resetTokenTTL)
		assert.Equal(t, customStrength, impl.passwordStrength)
	})
}

func TestPasswordService_Register(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	t.Run("registers new user with valid password", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		email := "newuser@example.com"
		password := "SecurePass123!"

		// Mock user doesn't exist
		storage.On("GetUserByEmail", mock.Anything, email).Return(nil, ErrUserNotFound)
		storage.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
			return u.Email == email &&
				u.AuthMethod == MethodPassword &&
				!u.IsVerified &&
				u.ID.String() != ""
		})).Return(nil)
		storage.On("StorePasswordHash", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("[]uint8")).Return(nil)

		ctx := context.Background()
		user, err := svc.Register(ctx, email, password)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, MethodPassword, user.AuthMethod)
		assert.False(t, user.IsVerified)
		assert.True(t, user.CreatedAt.Before(time.Now().Add(time.Second)))

		storage.AssertExpectations(t)
	})

	t.Run("normalizes email addresses", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		inputEmail := "  Test.User+Tag@EXAMPLE.COM  "
		normalizedEmail := "test.user+tag@example.com"
		password := "SecurePass123!"

		storage.On("GetUserByEmail", mock.Anything, normalizedEmail).Return(nil, ErrUserNotFound)
		storage.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
			return u.Email == normalizedEmail
		})).Return(nil)
		storage.On("StorePasswordHash", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("[]uint8")).Return(nil)

		ctx := context.Background()
		user, err := svc.Register(ctx, inputEmail, password)

		require.NoError(t, err)
		assert.Equal(t, normalizedEmail, user.Email)

		storage.AssertExpectations(t)
	})

	t.Run("validates email format", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		testCases := []struct {
			name  string
			email string
		}{
			{"empty email", ""},
			{"invalid format", "not-an-email"},
			{"missing domain", "user@"},
			{"missing username", "@example.com"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				user, err := svc.Register(ctx, tc.email, "ValidPass123!")

				assert.Error(t, err)
				assert.Nil(t, user)
			})
		}

		storage.AssertExpectations(t)
	})

	t.Run("validates password strength", func(t *testing.T) {
		t.Parallel()

		email := "test@example.com"

		testCases := []struct {
			name     string
			password string
		}{
			{"too short", "Pass1!"},
			{"too simple", "password"},
			{"only lowercase", "passwordonly"},
			{"empty password", ""},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				storage := &MockPasswordStorage{}
				svc := NewPasswordService(storage, tokenSecret)

				// Mock that user doesn't exist for validation test
				storage.On("GetUserByEmail", mock.Anything, email).Return(nil, ErrUserNotFound).Maybe()

				ctx := context.Background()
				user, err := svc.Register(ctx, email, tc.password)

				assert.Error(t, err)
				assert.Nil(t, user)

				storage.AssertExpectations(t)
			})
		}
	})

	t.Run("rejects common passwords", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		email := "test@example.com"

		// Common passwords that should be rejected
		commonPasswords := []string{
			"password123",
			"123456789",
		}

		for _, password := range commonPasswords {
			t.Run("password: "+password, func(t *testing.T) {
				ctx := context.Background()
				user, err := svc.Register(ctx, email, password)

				assert.Error(t, err)
				assert.Nil(t, user)
			})
		}

		storage.AssertExpectations(t)
	})

	t.Run("rejects existing email", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		email := "existing@example.com"
		password := "ValidPass123!"
		existingUser := &User{ID: uuid.New(), Email: email}

		storage.On("GetUserByEmail", mock.Anything, email).Return(existingUser, nil)

		ctx := context.Background()
		user, err := svc.Register(ctx, email, password)

		assert.Equal(t, ErrEmailAlreadyExists, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
	})

	t.Run("handles storage errors", func(t *testing.T) {
		t.Parallel()

		email := "test@example.com"
		password := "ValidPass123!"

		t.Run("user lookup error", func(t *testing.T) {
			t.Parallel()

			storage := &MockPasswordStorage{}
			svc := NewPasswordService(storage, tokenSecret)

			storage.On("GetUserByEmail", mock.Anything, email).Return(nil, errors.New("db error"))

			ctx := context.Background()
			user, err := svc.Register(ctx, email, password)

			assert.Error(t, err)
			assert.Nil(t, user)
			assert.Contains(t, err.Error(), "failed to check existing user")

			storage.AssertExpectations(t)
		})

		t.Run("user creation error", func(t *testing.T) {
			t.Parallel()

			storage := &MockPasswordStorage{}
			svc := NewPasswordService(storage, tokenSecret)

			storage.On("GetUserByEmail", mock.Anything, email).Return(nil, ErrUserNotFound)
			storage.On("CreateUser", mock.Anything, mock.Anything).Return(errors.New("create error"))

			ctx := context.Background()
			user, err := svc.Register(ctx, email, password)

			assert.Error(t, err)
			assert.Nil(t, user)
			assert.Contains(t, err.Error(), "failed to create user")

			storage.AssertExpectations(t)
		})

		t.Run("password storage error with cleanup", func(t *testing.T) {
			t.Parallel()

			storage := &MockPasswordStorage{}
			svc := NewPasswordService(storage, tokenSecret)

			storage.On("GetUserByEmail", mock.Anything, email).Return(nil, ErrUserNotFound)
			storage.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
			storage.On("StorePasswordHash", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("[]uint8")).Return(errors.New("password error"))
			storage.On("DeleteUser", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)

			ctx := context.Background()
			user, err := svc.Register(ctx, email, password)

			assert.Error(t, err)
			assert.Nil(t, user)
			assert.Contains(t, err.Error(), "failed to save password")

			storage.AssertExpectations(t)
		})
	})

	t.Run("executes afterRegister hook", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		hookCalled := false

		afterRegister := func(ctx context.Context, user *User) error {
			assert.NotNil(t, user)
			hookCalled = true
			return nil
		}

		svc := NewPasswordService(storage, tokenSecret, WithAfterRegister(afterRegister))

		email := "test@example.com"
		password := "ValidPass123!"

		storage.On("GetUserByEmail", mock.Anything, email).Return(nil, ErrUserNotFound)
		storage.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
		storage.On("StorePasswordHash", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("[]uint8")).Return(nil)

		ctx := context.Background()
		user, err := svc.Register(ctx, email, password)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.True(t, hookCalled)

		storage.AssertExpectations(t)
	})
}

func TestPasswordService_Authenticate(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	t.Run("authenticates valid credentials", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		email := "user@example.com"
		password := "correct-password"
		user := &User{ID: uuid.New(), Email: email, IsVerified: true}

		// Pre-hash the password for the test
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)

		storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)
		storage.On("GetPasswordHash", mock.Anything, user.ID).Return(hash, nil)

		ctx := context.Background()
		resultUser, err := svc.Authenticate(ctx, email, password)

		require.NoError(t, err)
		require.NotNil(t, resultUser)
		assert.Equal(t, user.ID, resultUser.ID)
		assert.Equal(t, user.Email, resultUser.Email)

		storage.AssertExpectations(t)
	})

	t.Run("normalizes email during authentication", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		inputEmail := "  Test@EXAMPLE.COM  "
		normalizedEmail := "test@example.com"
		password := "correct-password"
		user := &User{ID: uuid.New(), Email: normalizedEmail}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)

		storage.On("GetUserByEmail", mock.Anything, normalizedEmail).Return(user, nil)
		storage.On("GetPasswordHash", mock.Anything, user.ID).Return(hash, nil)

		ctx := context.Background()
		resultUser, err := svc.Authenticate(ctx, inputEmail, password)

		require.NoError(t, err)
		assert.Equal(t, user.ID, resultUser.ID)

		storage.AssertExpectations(t)
	})

	t.Run("returns generic error for invalid credentials to prevent enumeration", func(t *testing.T) {
		t.Parallel()

		email := "test@example.com"
		password := "wrong-password"

		testCases := []struct {
			name      string
			mockSetup func(*MockPasswordStorage)
		}{
			{
				name: "user not found",
				mockSetup: func(storage *MockPasswordStorage) {
					storage.On("GetUserByEmail", mock.Anything, email).Return(nil, ErrUserNotFound)
				},
			},
			{
				name: "user lookup error",
				mockSetup: func(storage *MockPasswordStorage) {
					storage.On("GetUserByEmail", mock.Anything, email).Return(nil, errors.New("db error"))
				},
			},
			{
				name: "password hash not found",
				mockSetup: func(storage *MockPasswordStorage) {
					user := &User{ID: uuid.New(), Email: email}
					storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)
					storage.On("GetPasswordHash", mock.Anything, user.ID).Return(nil, errors.New("hash error"))
				},
			},
			{
				name: "password mismatch",
				mockSetup: func(storage *MockPasswordStorage) {
					user := &User{ID: uuid.New(), Email: email}
					wrongHash, _ := bcrypt.GenerateFromPassword([]byte("different-password"), bcrypt.DefaultCost)
					storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)
					storage.On("GetPasswordHash", mock.Anything, user.ID).Return(wrongHash, nil)
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				storage := &MockPasswordStorage{}
				svc := NewPasswordService(storage, tokenSecret)

				tc.mockSetup(storage)

				ctx := context.Background()
				user, err := svc.Authenticate(ctx, email, password)

				assert.Equal(t, ErrInvalidCredentials, err)
				assert.Nil(t, user)

				storage.AssertExpectations(t)
			})
		}
	})

	t.Run("executes hooks", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		beforeLoginCalled := false
		afterLoginCalled := false

		beforeLogin := func(ctx context.Context, email string) error {
			assert.Equal(t, "test@example.com", email)
			beforeLoginCalled = true
			return nil
		}

		afterLogin := func(ctx context.Context, user *User) error {
			assert.NotNil(t, user)
			afterLoginCalled = true
			return nil
		}

		svc := NewPasswordService(storage, tokenSecret,
			WithBeforeLogin(beforeLogin),
			WithAfterLogin(afterLogin),
		)

		email := "test@example.com"
		password := "correct-password"
		user := &User{ID: uuid.New(), Email: email}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)

		storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)
		storage.On("GetPasswordHash", mock.Anything, user.ID).Return(hash, nil)

		ctx := context.Background()
		_, err = svc.Authenticate(ctx, email, password)

		require.NoError(t, err)
		assert.True(t, beforeLoginCalled)
		assert.True(t, afterLoginCalled)

		storage.AssertExpectations(t)
	})

	t.Run("blocks login when beforeLogin hook fails", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}

		beforeLogin := func(ctx context.Context, email string) error {
			return errors.New("blocked by hook")
		}

		svc := NewPasswordService(storage, tokenSecret, WithBeforeLogin(beforeLogin))

		email := "test@example.com"
		password := "password"

		ctx := context.Background()
		user, err := svc.Authenticate(ctx, email, password)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "login blocked")
		assert.Nil(t, user)

		storage.AssertExpectations(t)
	})
}

func TestPasswordService_ForgotPassword(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	t.Run("generates reset token for existing user", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		email := "user@example.com"
		user := &User{ID: uuid.New(), Email: email}

		storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)

		ctx := context.Background()
		req, err := svc.ForgotPassword(ctx, email)

		require.NoError(t, err)
		require.NotNil(t, req)
		assert.Equal(t, email, req.Email)
		assert.NotEmpty(t, req.Token)
		assert.True(t, req.ExpiresAt.After(time.Now()))
		assert.True(t, req.ExpiresAt.Before(time.Now().Add(2*time.Hour)))

		storage.AssertExpectations(t)
	})

	t.Run("normalizes email addresses", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		inputEmail := "  Test@EXAMPLE.COM  "
		normalizedEmail := "test@example.com"
		user := &User{ID: uuid.New(), Email: normalizedEmail}

		storage.On("GetUserByEmail", mock.Anything, normalizedEmail).Return(user, nil)

		ctx := context.Background()
		req, err := svc.ForgotPassword(ctx, inputEmail)

		require.NoError(t, err)
		assert.Equal(t, normalizedEmail, req.Email)

		storage.AssertExpectations(t)
	})

	t.Run("handles user not found", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		email := "nonexistent@example.com"

		storage.On("GetUserByEmail", mock.Anything, email).Return(nil, ErrUserNotFound)

		ctx := context.Background()
		req, err := svc.ForgotPassword(ctx, email)

		assert.Error(t, err)
		assert.Nil(t, req)
		assert.Contains(t, err.Error(), "failed to get user")

		storage.AssertExpectations(t)
	})

	t.Run("handles storage errors", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		email := "test@example.com"

		storage.On("GetUserByEmail", mock.Anything, email).Return(nil, errors.New("db error"))

		ctx := context.Background()
		req, err := svc.ForgotPassword(ctx, email)

		assert.Error(t, err)
		assert.Nil(t, req)
		assert.Contains(t, err.Error(), "failed to get user")

		storage.AssertExpectations(t)
	})
}

func TestPasswordService_ResetPassword(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	createValidResetToken := func(userID uuid.UUID, email string, expiresIn time.Duration) string {
		payload := PasswordResetTokenPayload{
			ID:       userID.String(),
			Email:    email,
			Subject:  SubjectPasswordReset,
			ExpireAt: time.Now().Add(expiresIn).Unix(),
		}
		tokenStr, err := token.GenerateToken(payload, tokenSecret)
		require.NoError(t, err)
		return tokenStr
	}

	t.Run("resets password with valid token", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		userID := uuid.New()
		email := "user@example.com"
		newPassword := "NewSecurePass123!"
		user := &User{ID: userID, Email: email}

		validToken := createValidResetToken(userID, email, 1*time.Hour)

		storage.On("StorePasswordHash", mock.Anything, userID, mock.AnythingOfType("[]uint8")).Return(nil)
		storage.On("GetUserByID", mock.Anything, userID).Return(user, nil)

		ctx := context.Background()
		resultUser, err := svc.ResetPassword(ctx, validToken, newPassword)

		require.NoError(t, err)
		require.NotNil(t, resultUser)
		assert.Equal(t, userID, resultUser.ID)

		storage.AssertExpectations(t)
	})

	t.Run("validates new password strength", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		userID := uuid.New()
		email := "user@example.com"
		validToken := createValidResetToken(userID, email, 1*time.Hour)

		weakPasswords := []string{
			"weak",
			"password",
			"12345678",
		}

		for _, password := range weakPasswords {
			t.Run("weak password: "+password, func(t *testing.T) {
				ctx := context.Background()
				user, err := svc.ResetPassword(ctx, validToken, password)

				assert.Error(t, err)
				assert.Nil(t, user)
			})
		}

		storage.AssertExpectations(t)
	})

	t.Run("rejects invalid tokens", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		newPassword := "ValidNewPass123!"

		testCases := []struct {
			name  string
			token string
		}{
			{"empty token", ""},
			{"malformed token", "invalid-token"},
			{"wrong secret", func() string {
				payload := PasswordResetTokenPayload{
					ID:      uuid.New().String(),
					Email:   "test@example.com",
					Subject: SubjectPasswordReset,
				}
				tokenStr, _ := token.GenerateToken(payload, "wrong-secret")
				return tokenStr
			}()},
			{"wrong subject", func() string {
				payload := PasswordResetTokenPayload{
					ID:      uuid.New().String(),
					Email:   "test@example.com",
					Subject: "wrong-subject",
				}
				tokenStr, _ := token.GenerateToken(payload, tokenSecret)
				return tokenStr
			}()},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				user, err := svc.ResetPassword(ctx, tc.token, newPassword)

				assert.Equal(t, ErrTokenInvalid, err)
				assert.Nil(t, user)
			})
		}

		storage.AssertExpectations(t)
	})

	t.Run("rejects expired tokens", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		userID := uuid.New()
		email := "user@example.com"
		newPassword := "ValidNewPass123!"

		// Create expired token
		expiredToken := createValidResetToken(userID, email, -1*time.Hour)

		ctx := context.Background()
		user, err := svc.ResetPassword(ctx, expiredToken, newPassword)

		assert.Equal(t, ErrTokenExpired, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
	})

	t.Run("handles invalid user ID in token", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		newPassword := "ValidNewPass123!"

		// Create token with invalid UUID
		payload := PasswordResetTokenPayload{
			ID:       "invalid-uuid",
			Email:    "test@example.com",
			Subject:  SubjectPasswordReset,
			ExpireAt: time.Now().Add(1 * time.Hour).Unix(),
		}
		invalidToken, err := token.GenerateToken(payload, tokenSecret)
		require.NoError(t, err)

		ctx := context.Background()
		user, err := svc.ResetPassword(ctx, invalidToken, newPassword)

		assert.Equal(t, ErrTokenInvalid, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
	})

	t.Run("handles storage errors", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		email := "user@example.com"
		newPassword := "ValidNewPass123!"
		validToken := createValidResetToken(userID, email, 1*time.Hour)

		t.Run("password update error", func(t *testing.T) {
			t.Parallel()

			storage := &MockPasswordStorage{}
			svc := NewPasswordService(storage, tokenSecret)

			storage.On("StorePasswordHash", mock.Anything, userID, mock.AnythingOfType("[]uint8")).Return(errors.New("update error"))

			ctx := context.Background()
			user, err := svc.ResetPassword(ctx, validToken, newPassword)

			assert.Error(t, err)
			assert.Nil(t, user)
			assert.Contains(t, err.Error(), "failed to update password")

			storage.AssertExpectations(t)
		})

		t.Run("user lookup error", func(t *testing.T) {
			t.Parallel()

			storage := &MockPasswordStorage{}
			svc := NewPasswordService(storage, tokenSecret)

			storage.On("StorePasswordHash", mock.Anything, userID, mock.AnythingOfType("[]uint8")).Return(nil)
			storage.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("lookup error"))

			ctx := context.Background()
			user, err := svc.ResetPassword(ctx, validToken, newPassword)

			assert.Error(t, err)
			assert.Nil(t, user)

			storage.AssertExpectations(t)
		})
	})
}

func TestPasswordHashing(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	t.Run("password hashing is secure", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		svc := NewPasswordService(storage, tokenSecret)

		email := "test@example.com"
		password := "TestPassword123!"
		var capturedHash []byte

		storage.On("GetUserByEmail", mock.Anything, email).Return(nil, ErrUserNotFound)
		storage.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
		storage.On("StorePasswordHash", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("[]uint8")).Run(func(args mock.Arguments) {
			capturedHash = args.Get(2).([]byte)
		}).Return(nil)

		ctx := context.Background()
		user, err := svc.Register(ctx, email, password)

		require.NoError(t, err)
		require.NotNil(t, user)
		require.NotNil(t, capturedHash)

		// Verify the hash can verify the original password
		err = bcrypt.CompareHashAndPassword(capturedHash, []byte(password))
		assert.NoError(t, err)

		// Verify the hash cannot verify wrong password
		err = bcrypt.CompareHashAndPassword(capturedHash, []byte("wrong-password"))
		assert.Error(t, err)

		storage.AssertExpectations(t)
	})

	t.Run("custom bcrypt cost is used", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		customCost := 6 // Lower for faster tests
		svc := NewPasswordService(storage, tokenSecret, WithBcryptCost(customCost))

		email := "test@example.com"
		password := "TestPassword123!"
		var capturedHash []byte

		storage.On("GetUserByEmail", mock.Anything, email).Return(nil, ErrUserNotFound)
		storage.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
		storage.On("StorePasswordHash", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("[]uint8")).Run(func(args mock.Arguments) {
			capturedHash = args.Get(2).([]byte)
		}).Return(nil)

		ctx := context.Background()
		_, err := svc.Register(ctx, email, password)

		require.NoError(t, err)

		// Verify the cost is as expected
		cost, err := bcrypt.Cost(capturedHash)
		require.NoError(t, err)
		assert.Equal(t, customCost, cost)

		storage.AssertExpectations(t)
	})
}

func TestPasswordResetTokenSecurity(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	t.Run("reset tokens contain expected fields", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		email := "test@example.com"

		payload := PasswordResetTokenPayload{
			ID:       userID.String(),
			Email:    email,
			Subject:  SubjectPasswordReset,
			ExpireAt: time.Now().Add(1 * time.Hour).Unix(),
		}

		tokenStr, err := token.GenerateToken(payload, tokenSecret)
		require.NoError(t, err)

		// Parse the token back
		parsedPayload, err := token.ParseToken[PasswordResetTokenPayload](tokenStr, tokenSecret)
		require.NoError(t, err)

		assert.Equal(t, payload.ID, parsedPayload.ID)
		assert.Equal(t, payload.Email, parsedPayload.Email)
		assert.Equal(t, payload.Subject, parsedPayload.Subject)
		assert.Equal(t, payload.ExpireAt, parsedPayload.ExpireAt)
	})

	t.Run("custom TTL is respected", func(t *testing.T) {
		t.Parallel()

		storage := &MockPasswordStorage{}
		customTTL := 30 * time.Minute
		svc := NewPasswordService(storage, tokenSecret, WithResetTokenTTL(customTTL))

		email := "test@example.com"
		user := &User{ID: uuid.New(), Email: email}

		storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)

		ctx := context.Background()
		req, err := svc.ForgotPassword(ctx, email)

		require.NoError(t, err)
		assert.True(t, req.ExpiresAt.Before(time.Now().Add(35*time.Minute)))
		assert.True(t, req.ExpiresAt.After(time.Now().Add(25*time.Minute)))

		storage.AssertExpectations(t)
	})
}

// Test that the service correctly implements the interface
func TestPasswordServiceInterface(t *testing.T) {
	t.Parallel()

	storage := &MockPasswordStorage{}
	var svc PasswordAuthenticator = NewPasswordService(storage, "secret")

	require.NotNil(t, svc)
	// If this compiles, the interface is correctly implemented
}
