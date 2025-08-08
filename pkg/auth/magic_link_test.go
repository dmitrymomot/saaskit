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

	"github.com/dmitrymomot/saaskit/pkg/token"
)

func TestNewMagicLinkService(t *testing.T) {
	t.Parallel()

	storage := &MockMagicLinkStorage{}
	tokenSecret := "test-secret"

	t.Run("creates service with defaults", func(t *testing.T) {
		t.Parallel()

		svc := NewMagicLinkService(storage, tokenSecret)
		require.NotNil(t, svc)

		// Cast to implementation to verify defaults
		impl := svc.(*magicLinkService)
		assert.Equal(t, storage, impl.storage)
		assert.Equal(t, tokenSecret, impl.tokenSecret)
		assert.Equal(t, 15*time.Minute, impl.magicLinkTTL)
		assert.NotNil(t, impl.logger)
	})

	t.Run("applies options correctly", func(t *testing.T) {
		t.Parallel()

		logger := slog.Default()
		customTTL := 30 * time.Minute

		svc := NewMagicLinkService(storage, tokenSecret,
			WithMagicLinkLogger(logger),
			WithMagicLinkTTL(customTTL),
		)

		impl := svc.(*magicLinkService)
		assert.Equal(t, logger, impl.logger)
		assert.Equal(t, customTTL, impl.magicLinkTTL)
	})
}

func TestMagicLinkService_RequestMagicLink(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	t.Run("generates magic link for new user", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		email := "test@example.com"

		// Mock user not found initially (triggers auto-registration)
		storage.On("GetUserByEmail", mock.Anything, email).Return(nil, ErrUserNotFound)
		storage.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
			return u.Email == email && u.AuthMethod == MethodMagicLink && !u.IsVerified
		})).Return(nil)

		ctx := context.Background()
		req, err := svc.RequestMagicLink(ctx, email)

		require.NoError(t, err)
		require.NotNil(t, req)
		assert.Equal(t, email, req.Email)
		assert.NotEmpty(t, req.Token)
		assert.True(t, req.ExpiresAt.After(time.Now()))
		assert.True(t, req.ExpiresAt.Before(time.Now().Add(20*time.Minute)))

		storage.AssertExpectations(t)
	})

	t.Run("generates magic link for existing user", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		email := "existing@example.com"
		existingUser := &User{
			ID:         uuid.New(),
			Email:      email,
			AuthMethod: MethodMagicLink,
			IsVerified: true,
			CreatedAt:  time.Now(),
		}

		storage.On("GetUserByEmail", mock.Anything, email).Return(existingUser, nil)

		ctx := context.Background()
		req, err := svc.RequestMagicLink(ctx, email)

		require.NoError(t, err)
		require.NotNil(t, req)
		assert.Equal(t, email, req.Email)
		assert.NotEmpty(t, req.Token)

		storage.AssertExpectations(t)
	})

	t.Run("normalizes email addresses", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		inputEmail := "  Test.User+Tag@EXAMPLE.COM  "
		normalizedEmail := "test.user+tag@example.com"

		storage.On("GetUserByEmail", mock.Anything, normalizedEmail).Return(nil, ErrUserNotFound)
		storage.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
			return u.Email == normalizedEmail
		})).Return(nil)

		ctx := context.Background()
		req, err := svc.RequestMagicLink(ctx, inputEmail)

		require.NoError(t, err)
		assert.Equal(t, normalizedEmail, req.Email)

		storage.AssertExpectations(t)
	})

	t.Run("validates email format", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

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
				req, err := svc.RequestMagicLink(ctx, tc.email)

				assert.Error(t, err)
				assert.Nil(t, req)
			})
		}

		storage.AssertExpectations(t)
	})

	t.Run("handles storage errors", func(t *testing.T) {
		t.Parallel()

		email := "test@example.com"

		t.Run("get user error", func(t *testing.T) {
			t.Parallel()

			storage := &MockMagicLinkStorage{}
			svc := NewMagicLinkService(storage, tokenSecret)

			storage.On("GetUserByEmail", mock.Anything, email).Return(nil, errors.New("db error"))

			ctx := context.Background()
			req, err := svc.RequestMagicLink(ctx, email)

			assert.Error(t, err)
			assert.Nil(t, req)
			assert.Contains(t, err.Error(), "failed to check user")

			storage.AssertExpectations(t)
		})

		t.Run("create user error", func(t *testing.T) {
			t.Parallel()

			storage := &MockMagicLinkStorage{}
			svc := NewMagicLinkService(storage, tokenSecret)

			storage.On("GetUserByEmail", mock.Anything, email).Return(nil, ErrUserNotFound)
			storage.On("CreateUser", mock.Anything, mock.Anything).Return(errors.New("create error"))

			ctx := context.Background()
			req, err := svc.RequestMagicLink(ctx, email)

			assert.Error(t, err)
			assert.Nil(t, req)
			assert.Contains(t, err.Error(), "failed to create user")

			storage.AssertExpectations(t)
		})
	})

	t.Run("executes afterGenerate hook", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		hookCalled := make(chan bool, 1)

		afterGenerate := func(ctx context.Context, user *User, token string) error {
			assert.NotNil(t, user)
			assert.NotEmpty(t, token)
			select {
			case hookCalled <- true:
			default:
			}
			return nil
		}

		svc := NewMagicLinkService(storage, tokenSecret, WithAfterGenerate(afterGenerate))

		email := "test@example.com"
		existingUser := &User{ID: uuid.New(), Email: email}

		storage.On("GetUserByEmail", mock.Anything, email).Return(existingUser, nil)

		ctx := context.Background()
		_, err := svc.RequestMagicLink(ctx, email)

		require.NoError(t, err)

		// Wait for hook to execute
		select {
		case called := <-hookCalled:
			assert.True(t, called)
		case <-time.After(1 * time.Second):
			t.Fatal("Hook was not called within timeout")
		}

		storage.AssertExpectations(t)
	})
}

func TestMagicLinkService_VerifyMagicLink(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	createValidToken := func(email string, expiresIn time.Duration) string {
		payload := MagicLinkTokenPayload{
			ID:       uuid.New().String(),
			Email:    email,
			Subject:  SubjectMagicLink,
			ExpireAt: time.Now().Add(expiresIn).Unix(),
		}
		tokenStr, err := token.GenerateToken(payload, tokenSecret)
		require.NoError(t, err)
		return tokenStr
	}

	t.Run("verifies valid token for existing verified user", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		email := "verified@example.com"
		user := &User{
			ID:         uuid.New(),
			Email:      email,
			AuthMethod: MethodMagicLink,
			IsVerified: true,
			CreatedAt:  time.Now(),
		}

		validToken := createValidToken(email, 15*time.Minute)

		storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)

		ctx := context.Background()
		resultUser, err := svc.VerifyMagicLink(ctx, validToken)

		require.NoError(t, err)
		require.NotNil(t, resultUser)
		assert.Equal(t, user.ID, resultUser.ID)
		assert.Equal(t, user.Email, resultUser.Email)
		assert.True(t, resultUser.IsVerified)

		storage.AssertExpectations(t)
	})

	t.Run("verifies and updates unverified user", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		email := "unverified@example.com"
		user := &User{
			ID:         uuid.New(),
			Email:      email,
			AuthMethod: MethodMagicLink,
			IsVerified: false,
			CreatedAt:  time.Now(),
		}

		validToken := createValidToken(email, 15*time.Minute)

		storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)
		storage.On("UpdateUserVerified", mock.Anything, user.ID, true).Return(nil)

		ctx := context.Background()
		resultUser, err := svc.VerifyMagicLink(ctx, validToken)

		require.NoError(t, err)
		require.NotNil(t, resultUser)
		assert.Equal(t, user.ID, resultUser.ID)
		assert.True(t, resultUser.IsVerified) // Should be updated in the returned user

		storage.AssertExpectations(t)
	})

	t.Run("rejects invalid tokens", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		testCases := []struct {
			name  string
			token string
		}{
			{"empty token", ""},
			{"malformed token", "invalid-token"},
			{"wrong secret", func() string {
				payload := MagicLinkTokenPayload{
					ID:      uuid.New().String(),
					Email:   "test@example.com",
					Subject: SubjectMagicLink,
				}
				tokenStr, _ := token.GenerateToken(payload, "wrong-secret")
				return tokenStr
			}()},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				user, err := svc.VerifyMagicLink(ctx, tc.token)

				assert.Equal(t, ErrTokenInvalid, err)
				assert.Nil(t, user)
			})
		}

		storage.AssertExpectations(t)
	})

	t.Run("rejects wrong subject", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		// Create token with wrong subject
		payload := MagicLinkTokenPayload{
			ID:       uuid.New().String(),
			Email:    "test@example.com",
			Subject:  "wrong-subject",
			ExpireAt: time.Now().Add(15 * time.Minute).Unix(),
		}
		wrongSubjectToken, err := token.GenerateToken(payload, tokenSecret)
		require.NoError(t, err)

		ctx := context.Background()
		user, err := svc.VerifyMagicLink(ctx, wrongSubjectToken)

		assert.Equal(t, ErrTokenInvalid, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
	})

	t.Run("rejects expired tokens", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		// Create expired token
		expiredToken := createValidToken("test@example.com", -1*time.Hour)

		ctx := context.Background()
		user, err := svc.VerifyMagicLink(ctx, expiredToken)

		assert.Equal(t, ErrTokenExpired, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
	})

	t.Run("handles user not found", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		email := "nonexistent@example.com"
		validToken := createValidToken(email, 15*time.Minute)

		storage.On("GetUserByEmail", mock.Anything, email).Return(nil, ErrUserNotFound)

		ctx := context.Background()
		user, err := svc.VerifyMagicLink(ctx, validToken)

		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
	})

	t.Run("handles storage error during user lookup", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		email := "test@example.com"
		validToken := createValidToken(email, 15*time.Minute)

		storage.On("GetUserByEmail", mock.Anything, email).Return(nil, errors.New("db error"))

		ctx := context.Background()
		user, err := svc.VerifyMagicLink(ctx, validToken)

		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
	})

	t.Run("continues verification even if update verified status fails", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		email := "unverified@example.com"
		user := &User{
			ID:         uuid.New(),
			Email:      email,
			AuthMethod: MethodMagicLink,
			IsVerified: false,
			CreatedAt:  time.Now(),
		}

		validToken := createValidToken(email, 15*time.Minute)

		storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)
		storage.On("UpdateUserVerified", mock.Anything, user.ID, true).Return(errors.New("update error"))

		ctx := context.Background()
		resultUser, err := svc.VerifyMagicLink(ctx, validToken)

		require.NoError(t, err) // Should still succeed
		require.NotNil(t, resultUser)
		assert.True(t, resultUser.IsVerified) // Should be updated locally

		storage.AssertExpectations(t)
	})

	t.Run("executes beforeVerify hook", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		hookCalled := false

		beforeVerify := func(ctx context.Context, token string) error {
			hookCalled = true
			assert.NotEmpty(t, token)
			return nil
		}

		svc := NewMagicLinkService(storage, tokenSecret, WithBeforeVerify(beforeVerify))

		email := "test@example.com"
		user := &User{ID: uuid.New(), Email: email, IsVerified: true}
		validToken := createValidToken(email, 15*time.Minute)

		storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)

		ctx := context.Background()
		_, err := svc.VerifyMagicLink(ctx, validToken)

		require.NoError(t, err)
		assert.True(t, hookCalled)

		storage.AssertExpectations(t)
	})

	t.Run("blocks verification when beforeVerify hook fails", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}

		beforeVerify := func(ctx context.Context, token string) error {
			return errors.New("blocked by hook")
		}

		svc := NewMagicLinkService(storage, tokenSecret, WithBeforeVerify(beforeVerify))

		email := "test@example.com"
		validToken := createValidToken(email, 15*time.Minute)

		ctx := context.Background()
		user, err := svc.VerifyMagicLink(ctx, validToken)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "verify blocked")
		assert.Nil(t, user)

		storage.AssertExpectations(t)
	})

	t.Run("executes afterVerify hook", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		hookCalled := make(chan bool, 1)

		afterVerify := func(ctx context.Context, user *User) error {
			assert.NotNil(t, user)
			select {
			case hookCalled <- true:
			default:
			}
			return nil
		}

		svc := NewMagicLinkService(storage, tokenSecret, WithAfterVerify(afterVerify))

		email := "test@example.com"
		user := &User{ID: uuid.New(), Email: email, IsVerified: true}
		validToken := createValidToken(email, 15*time.Minute)

		storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)

		ctx := context.Background()
		_, err := svc.VerifyMagicLink(ctx, validToken)

		require.NoError(t, err)

		// Wait for hook to execute
		select {
		case called := <-hookCalled:
			assert.True(t, called)
		case <-time.After(1 * time.Second):
			t.Fatal("Hook was not called within timeout")
		}

		storage.AssertExpectations(t)
	})
}

func TestMagicLinkTokenValidation(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	t.Run("generated tokens are valid", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		email := "test@example.com"
		user := &User{ID: uuid.New(), Email: email, IsVerified: true}

		storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)

		ctx := context.Background()

		// Generate a magic link
		req, err := svc.RequestMagicLink(ctx, email)
		require.NoError(t, err)

		// Verify the generated token is valid
		resultUser, err := svc.VerifyMagicLink(ctx, req.Token)
		require.NoError(t, err)
		assert.Equal(t, user.ID, resultUser.ID)

		storage.AssertExpectations(t)
	})

	t.Run("token contains expected fields", func(t *testing.T) {
		t.Parallel()

		email := "test@example.com"
		payload := MagicLinkTokenPayload{
			ID:       uuid.New().String(),
			Email:    email,
			Subject:  SubjectMagicLink,
			ExpireAt: time.Now().Add(15 * time.Minute).Unix(),
		}

		tokenStr, err := token.GenerateToken(payload, tokenSecret)
		require.NoError(t, err)

		// Parse the token back
		parsedPayload, err := token.ParseToken[MagicLinkTokenPayload](tokenStr, tokenSecret)
		require.NoError(t, err)

		assert.Equal(t, payload.ID, parsedPayload.ID)
		assert.Equal(t, payload.Email, parsedPayload.Email)
		assert.Equal(t, payload.Subject, parsedPayload.Subject)
		assert.Equal(t, payload.ExpireAt, parsedPayload.ExpireAt)
	})
}

func TestMagicLinkSecurityFeatures(t *testing.T) {
	t.Parallel()

	const tokenSecret = "test-secret-32-chars-long-12345"

	t.Run("tokens have unique IDs for replay protection readiness", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		svc := NewMagicLinkService(storage, tokenSecret)

		email := "test@example.com"
		user := &User{ID: uuid.New(), Email: email, IsVerified: true}

		storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil).Twice()

		ctx := context.Background()

		// Generate two tokens
		req1, err := svc.RequestMagicLink(ctx, email)
		require.NoError(t, err)

		req2, err := svc.RequestMagicLink(ctx, email)
		require.NoError(t, err)

		// Tokens should be different
		assert.NotEqual(t, req1.Token, req2.Token)

		// Parse both tokens to check IDs
		payload1, err := token.ParseToken[MagicLinkTokenPayload](req1.Token, tokenSecret)
		require.NoError(t, err)

		payload2, err := token.ParseToken[MagicLinkTokenPayload](req2.Token, tokenSecret)
		require.NoError(t, err)

		// IDs should be unique
		assert.NotEqual(t, payload1.ID, payload2.ID)

		storage.AssertExpectations(t)
	})

	t.Run("short TTL reduces attack window", func(t *testing.T) {
		t.Parallel()

		storage := &MockMagicLinkStorage{}
		shortTTL := 1 * time.Minute
		svc := NewMagicLinkService(storage, tokenSecret, WithMagicLinkTTL(shortTTL))

		email := "test@example.com"
		user := &User{ID: uuid.New(), Email: email}

		storage.On("GetUserByEmail", mock.Anything, email).Return(user, nil)

		ctx := context.Background()
		req, err := svc.RequestMagicLink(ctx, email)

		require.NoError(t, err)
		assert.True(t, req.ExpiresAt.Before(time.Now().Add(2*time.Minute)))
		assert.True(t, req.ExpiresAt.After(time.Now().Add(30*time.Second)))

		storage.AssertExpectations(t)
	})
}

// Test that the service correctly implements the interface
func TestMagicLinkServiceInterface(t *testing.T) {
	t.Parallel()

	storage := &MockMagicLinkStorage{}
	var svc MagicLinkAuthenticator = NewMagicLinkService(storage, "secret")

	require.NotNil(t, svc)
	// If this compiles, the interface is correctly implemented
}
