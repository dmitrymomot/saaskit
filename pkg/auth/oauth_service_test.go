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
)

func TestNewOAuthService(t *testing.T) {
	t.Parallel()

	storage := &MockOAuthStorage{}
	adapter := &MockProviderAdapter{}

	t.Run("creates service with defaults", func(t *testing.T) {
		t.Parallel()

		svc := NewOAuthService(storage, adapter)
		require.NotNil(t, svc)

		// Cast to implementation to verify defaults
		impl := svc.(*oauthService)
		assert.Equal(t, storage, impl.storage)
		assert.Equal(t, adapter, impl.adapter)
		assert.Equal(t, 10*time.Minute, impl.stateTTL)
		assert.True(t, impl.verifiedOnly)
		assert.NotNil(t, impl.logger)
	})

	t.Run("applies options correctly", func(t *testing.T) {
		t.Parallel()

		logger := slog.Default()
		customTTL := 5 * time.Minute
		verifiedOnly := false

		svc := NewOAuthService(storage, adapter,
			WithLogger(logger),
			WithStateTTL(customTTL),
			WithVerifiedOnly(verifiedOnly),
		)

		impl := svc.(*oauthService)
		assert.Equal(t, logger, impl.logger)
		assert.Equal(t, customTTL, impl.stateTTL)
		assert.False(t, impl.verifiedOnly)
	})
}

func TestOAuthService_GetAuthURL(t *testing.T) {
	t.Parallel()

	t.Run("generates auth URL successfully", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		expectedURL := "https://provider.com/oauth/authorize?state=test-state"

		storage.On("StoreState", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
		adapter.On("AuthURL", mock.AnythingOfType("string")).Return(expectedURL, nil)

		ctx := context.Background()
		authURL, err := svc.GetAuthURL(ctx)

		require.NoError(t, err)
		assert.Equal(t, expectedURL, authURL)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("generates unique state tokens", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		var capturedStates []string

		storage.On("StoreState", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Run(func(args mock.Arguments) {
			state := args.Get(1).(string)
			capturedStates = append(capturedStates, state)
		}).Return(nil).Twice()
		adapter.On("AuthURL", mock.AnythingOfType("string")).Return("https://provider.com/oauth/authorize", nil).Twice()

		ctx := context.Background()

		// Generate two URLs
		_, err1 := svc.GetAuthURL(ctx)
		_, err2 := svc.GetAuthURL(ctx)

		require.NoError(t, err1)
		require.NoError(t, err2)
		require.Len(t, capturedStates, 2)

		// States should be unique
		assert.NotEqual(t, capturedStates[0], capturedStates[1])

		// States should be base64-encoded (URL-safe)
		for _, state := range capturedStates {
			assert.NotEmpty(t, state)
			assert.True(t, len(state) > 10) // Should be reasonably long
		}

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("respects custom state TTL", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		customTTL := 5 * time.Minute
		svc := NewOAuthService(storage, adapter, WithStateTTL(customTTL))

		var capturedExpiry time.Time

		storage.On("StoreState", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Run(func(args mock.Arguments) {
			capturedExpiry = args.Get(2).(time.Time)
		}).Return(nil)
		adapter.On("AuthURL", mock.AnythingOfType("string")).Return("https://provider.com/oauth/authorize", nil)

		ctx := context.Background()
		startTime := time.Now()
		_, err := svc.GetAuthURL(ctx)

		require.NoError(t, err)

		// Verify TTL is approximately correct (within 1 second)
		expectedExpiry := startTime.Add(customTTL)
		assert.True(t, capturedExpiry.After(expectedExpiry.Add(-1*time.Second)))
		assert.True(t, capturedExpiry.Before(expectedExpiry.Add(1*time.Second)))

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("handles storage errors", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		storage.On("StoreState", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(errors.New("storage error"))

		ctx := context.Background()
		authURL, err := svc.GetAuthURL(ctx)

		assert.Error(t, err)
		assert.Empty(t, authURL)
		assert.Contains(t, err.Error(), "failed to store state")

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("handles adapter errors", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		storage.On("StoreState", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
		adapter.On("AuthURL", mock.AnythingOfType("string")).Return("", errors.New("adapter error"))

		ctx := context.Background()
		authURL, err := svc.GetAuthURL(ctx)

		assert.Error(t, err)
		assert.Empty(t, authURL)
		assert.Contains(t, err.Error(), "failed to build auth url")

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})
}

func TestOAuthService_Auth_NewUser(t *testing.T) {
	t.Parallel()

	t.Run("creates new user with verified email", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		code := "auth-code"
		state := "valid-state"
		profile := ProviderProfile{
			ProviderUserID: "provider-user-123",
			Email:          "  New.User+Tag@EXAMPLE.COM  ", // Will be normalized
			EmailVerified:  true,
		}
		normalizedEmail := "new.user+tag@example.com"

		adapter.On("ProviderID").Return("google")
		adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)

		storage.On("ConsumeState", mock.Anything, state).Return(nil)
		storage.On("GetUserByOAuth", mock.Anything, "google", "provider-user-123").Return(nil, ErrUserNotFound)
		storage.On("GetUserByEmail", mock.Anything, normalizedEmail).Return(nil, ErrUserNotFound)
		storage.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
			return u.Email == normalizedEmail &&
				u.AuthMethod == MethodOAuthGoogle &&
				u.IsVerified == true
		})).Return(nil)
		storage.On("StoreOAuthLink", mock.Anything, mock.AnythingOfType("uuid.UUID"), "google", "provider-user-123").Return(nil)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, nil)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, normalizedEmail, user.Email)
		assert.Equal(t, MethodOAuthGoogle, user.AuthMethod)
		assert.True(t, user.IsVerified)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("creates new user with unverified email when verifiedOnly is false", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter, WithVerifiedOnly(false))

		code := "auth-code"
		state := "valid-state"
		profile := ProviderProfile{
			ProviderUserID: "provider-user-123",
			Email:          "unverified@example.com",
			EmailVerified:  false,
		}

		adapter.On("ProviderID").Return("github")
		adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)

		storage.On("ConsumeState", mock.Anything, state).Return(nil)
		storage.On("GetUserByOAuth", mock.Anything, "github", "provider-user-123").Return(nil, ErrUserNotFound)
		storage.On("GetUserByEmail", mock.Anything, profile.Email).Return(nil, ErrUserNotFound)
		storage.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
			return u.AuthMethod == MethodOAuthGithub && !u.IsVerified
		})).Return(nil)
		storage.On("StoreOAuthLink", mock.Anything, mock.AnythingOfType("uuid.UUID"), "github", "provider-user-123").Return(nil)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, nil)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.False(t, user.IsVerified)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("rejects unverified email when verifiedOnly is true", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter, WithVerifiedOnly(true))

		code := "auth-code"
		state := "valid-state"
		profile := ProviderProfile{
			ProviderUserID: "provider-user-123",
			Email:          "unverified@example.com",
			EmailVerified:  false,
		}

		adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)
		storage.On("ConsumeState", mock.Anything, state).Return(nil)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, nil)

		assert.Equal(t, ErrUnverifiedEmail, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("prevents account takeover via OAuth", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		code := "auth-code"
		state := "valid-state"
		profile := ProviderProfile{
			ProviderUserID: "provider-user-123",
			Email:          "existing@example.com",
			EmailVerified:  true,
		}

		// Existing user with this email (different OAuth provider or password auth)
		existingUser := &User{ID: uuid.New(), Email: "existing@example.com"}

		adapter.On("ProviderID").Return("google")
		adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)

		storage.On("ConsumeState", mock.Anything, state).Return(nil)
		storage.On("GetUserByOAuth", mock.Anything, "google", "provider-user-123").Return(nil, ErrUserNotFound)
		storage.On("GetUserByEmail", mock.Anything, profile.Email).Return(existingUser, nil) // Email already taken

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, nil)

		assert.Equal(t, ErrProviderEmailInUse, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("cleans up user record if OAuth link storage fails", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		code := "auth-code"
		state := "valid-state"
		profile := ProviderProfile{
			ProviderUserID: "provider-user-123",
			Email:          "new@example.com",
			EmailVerified:  true,
		}

		adapter.On("ProviderID").Return("google")
		adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)

		storage.On("ConsumeState", mock.Anything, state).Return(nil)
		storage.On("GetUserByOAuth", mock.Anything, "google", "provider-user-123").Return(nil, ErrUserNotFound)
		storage.On("GetUserByEmail", mock.Anything, profile.Email).Return(nil, ErrUserNotFound)
		storage.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
		storage.On("StoreOAuthLink", mock.Anything, mock.AnythingOfType("uuid.UUID"), "google", "provider-user-123").Return(errors.New("link storage error"))
		storage.On("DeleteUser", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, nil)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "failed to store oauth link")

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("executes afterAuth hook for new users", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		hookCalled := false

		afterAuth := func(ctx context.Context, user *User) error {
			hookCalled = true
			assert.NotNil(t, user)
			return nil
		}

		svc := NewOAuthService(storage, adapter, WithAfterAuth(afterAuth))

		code := "auth-code"
		state := "valid-state"
		profile := ProviderProfile{
			ProviderUserID: "provider-user-123",
			Email:          "new@example.com",
			EmailVerified:  true,
		}

		adapter.On("ProviderID").Return("google")
		adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)

		storage.On("ConsumeState", mock.Anything, state).Return(nil)
		storage.On("GetUserByOAuth", mock.Anything, "google", "provider-user-123").Return(nil, ErrUserNotFound)
		storage.On("GetUserByEmail", mock.Anything, profile.Email).Return(nil, ErrUserNotFound)
		storage.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
		storage.On("StoreOAuthLink", mock.Anything, mock.AnythingOfType("uuid.UUID"), "google", "provider-user-123").Return(nil)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, nil)

		require.NoError(t, err)
		require.NotNil(t, user)

		// Give hook goroutine time to execute
		time.Sleep(100 * time.Millisecond)
		assert.True(t, hookCalled)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})
}

func TestOAuthService_Auth_ExistingUser(t *testing.T) {
	t.Parallel()

	t.Run("authenticates existing linked user", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		code := "auth-code"
		state := "valid-state"
		profile := ProviderProfile{
			ProviderUserID: "provider-user-123",
			Email:          "existing@example.com",
			EmailVerified:  true,
		}

		existingUser := &User{
			ID:         uuid.New(),
			Email:      "existing@example.com",
			AuthMethod: MethodOAuthGoogle,
			IsVerified: true,
		}

		adapter.On("ProviderID").Return("google")
		adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)

		storage.On("ConsumeState", mock.Anything, state).Return(nil)
		storage.On("GetUserByOAuth", mock.Anything, "google", "provider-user-123").Return(existingUser, nil)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, nil)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, existingUser.ID, user.ID)
		assert.Equal(t, existingUser.Email, user.Email)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})
}

func TestOAuthService_Auth_Linking(t *testing.T) {
	t.Parallel()

	t.Run("links OAuth account to existing user", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		code := "auth-code"
		state := "valid-state"
		linkToUserID := uuid.New()
		profile := ProviderProfile{
			ProviderUserID: "provider-user-123",
			Email:          "user@example.com",
			EmailVerified:  true,
		}

		existingUser := &User{
			ID:         linkToUserID,
			Email:      "user@example.com",
			AuthMethod: MethodPassword,
			IsVerified: true,
		}

		adapter.On("ProviderID").Return("google")
		adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)

		storage.On("ConsumeState", mock.Anything, state).Return(nil)
		storage.On("GetUserByOAuth", mock.Anything, "google", "provider-user-123").Return(nil, ErrUserNotFound) // Not linked yet
		storage.On("GetUserByID", mock.Anything, linkToUserID).Return(existingUser, nil)
		storage.On("StoreOAuthLink", mock.Anything, linkToUserID, "google", "provider-user-123").Return(nil)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, &linkToUserID)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, existingUser.ID, user.ID)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("returns existing user if OAuth account already linked to same user", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		code := "auth-code"
		state := "valid-state"
		linkToUserID := uuid.New()
		profile := ProviderProfile{
			ProviderUserID: "provider-user-123",
			Email:          "user@example.com",
			EmailVerified:  true,
		}

		existingUser := &User{
			ID:         linkToUserID,
			Email:      "user@example.com",
			AuthMethod: MethodPassword,
			IsVerified: true,
		}

		adapter.On("ProviderID").Return("google")
		adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)

		storage.On("ConsumeState", mock.Anything, state).Return(nil)
		storage.On("GetUserByOAuth", mock.Anything, "google", "provider-user-123").Return(existingUser, nil) // Already linked

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, &linkToUserID)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, existingUser.ID, user.ID)

		// Should not attempt to store link again
		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("prevents linking OAuth account already linked to different user", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		code := "auth-code"
		state := "valid-state"
		linkToUserID := uuid.New()
		differentUserID := uuid.New()
		profile := ProviderProfile{
			ProviderUserID: "provider-user-123",
			Email:          "user@example.com",
			EmailVerified:  true,
		}

		// OAuth account is already linked to a different user
		linkedUser := &User{
			ID:         differentUserID,
			Email:      "different@example.com",
			AuthMethod: MethodOAuthGoogle,
			IsVerified: true,
		}

		adapter.On("ProviderID").Return("google")
		adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)

		storage.On("ConsumeState", mock.Anything, state).Return(nil)
		storage.On("GetUserByOAuth", mock.Anything, "google", "provider-user-123").Return(linkedUser, nil)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, &linkToUserID)

		assert.Equal(t, ErrProviderLinked, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("executes linking hooks", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		beforeLinkCalled := make(chan bool, 1)
		afterLinkCalled := make(chan bool, 1)

		beforeLink := func(ctx context.Context, userID uuid.UUID) error {
			select {
			case beforeLinkCalled <- true:
			default:
			}
			return nil
		}

		afterLink := func(ctx context.Context, user *User) error {
			assert.NotNil(t, user)
			select {
			case afterLinkCalled <- true:
			default:
			}
			return nil
		}

		svc := NewOAuthService(storage, adapter,
			WithBeforeLink(beforeLink),
			WithAfterLink(afterLink),
		)

		code := "auth-code"
		state := "valid-state"
		linkToUserID := uuid.New()
		profile := ProviderProfile{
			ProviderUserID: "provider-user-123",
			Email:          "user@example.com",
			EmailVerified:  true,
		}

		existingUser := &User{ID: linkToUserID, Email: "user@example.com"}

		adapter.On("ProviderID").Return("google")
		adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)

		storage.On("ConsumeState", mock.Anything, state).Return(nil)
		storage.On("GetUserByOAuth", mock.Anything, "google", "provider-user-123").Return(nil, ErrUserNotFound)
		storage.On("GetUserByID", mock.Anything, linkToUserID).Return(existingUser, nil)
		storage.On("StoreOAuthLink", mock.Anything, linkToUserID, "google", "provider-user-123").Return(nil)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, &linkToUserID)

		require.NoError(t, err)
		require.NotNil(t, user)

		// Wait for before link hook
		select {
		case called := <-beforeLinkCalled:
			assert.True(t, called)
		case <-time.After(1 * time.Second):
			t.Fatal("Before link hook was not called within timeout")
		}

		// Wait for after link hook
		select {
		case called := <-afterLinkCalled:
			assert.True(t, called)
		case <-time.After(1 * time.Second):
			t.Fatal("After link hook was not called within timeout")
		}

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("blocks linking when beforeLink hook fails", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}

		beforeLink := func(ctx context.Context, userID uuid.UUID) error {
			return errors.New("blocked by hook")
		}

		svc := NewOAuthService(storage, adapter, WithBeforeLink(beforeLink))

		code := "auth-code"
		state := "valid-state"
		linkToUserID := uuid.New()
		profile := ProviderProfile{
			ProviderUserID: "provider-user-123",
			Email:          "user@example.com",
			EmailVerified:  true,
		}

		adapter.On("ProviderID").Return("google")
		adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)

		storage.On("ConsumeState", mock.Anything, state).Return(nil)
		storage.On("GetUserByOAuth", mock.Anything, "google", "provider-user-123").Return(nil, ErrUserNotFound)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, &linkToUserID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "link blocked")
		assert.Nil(t, user)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})
}

func TestOAuthService_Auth_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("rejects invalid state token", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		code := "auth-code"
		invalidState := "invalid-state"

		storage.On("ConsumeState", mock.Anything, invalidState).Return(ErrStateNotFound)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, invalidState, nil)

		assert.Equal(t, ErrInvalidState, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("handles state consumption errors", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		code := "auth-code"
		state := "valid-state"

		storage.On("ConsumeState", mock.Anything, state).Return(errors.New("db error"))

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, nil)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "failed to validate state")

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("handles invalid OAuth code", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		code := "invalid-code"
		state := "valid-state"

		adapter.On("ResolveProfile", mock.Anything, code).Return(ProviderProfile{}, ErrInvalidCode)
		storage.On("ConsumeState", mock.Anything, state).Return(nil)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, nil)

		assert.Equal(t, ErrInvalidCode, err)
		assert.Nil(t, user)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("handles provider-specific errors", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		code := "auth-code"
		state := "valid-state"

		adapter.On("ResolveProfile", mock.Anything, code).Return(ProviderProfile{}, ErrNoPrimaryEmail)
		storage.On("ConsumeState", mock.Anything, state).Return(nil)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, nil)

		assert.Error(t, err)
		assert.Nil(t, user)
		// Error should bubble up with context
		assert.Contains(t, err.Error(), "failed to resolve provider profile")

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("handles adapter errors", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		code := "auth-code"
		state := "valid-state"

		adapter.On("ResolveProfile", mock.Anything, code).Return(ProviderProfile{}, errors.New("network error"))
		storage.On("ConsumeState", mock.Anything, state).Return(nil)

		ctx := context.Background()
		user, err := svc.Auth(ctx, code, state, nil)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "failed to resolve provider profile")

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})
}

func TestOAuthService_Unlink(t *testing.T) {
	t.Parallel()

	t.Run("unlinks OAuth account successfully", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		userID := uuid.New()

		adapter.On("ProviderID").Return("google")
		storage.On("RemoveOAuthLink", mock.Anything, userID, "google").Return(nil)

		ctx := context.Background()
		err := svc.Unlink(ctx, userID)

		require.NoError(t, err)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("handles no provider link found", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		userID := uuid.New()

		adapter.On("ProviderID").Return("google")
		storage.On("RemoveOAuthLink", mock.Anything, userID, "google").Return(ErrNoProviderLink)

		ctx := context.Background()
		err := svc.Unlink(ctx, userID)

		assert.Equal(t, ErrNoProviderLink, err)

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})

	t.Run("handles storage errors", func(t *testing.T) {
		t.Parallel()

		storage := &MockOAuthStorage{}
		adapter := &MockProviderAdapter{}
		svc := NewOAuthService(storage, adapter)

		userID := uuid.New()

		adapter.On("ProviderID").Return("google")
		storage.On("RemoveOAuthLink", mock.Anything, userID, "google").Return(errors.New("db error"))

		ctx := context.Background()
		err := svc.Unlink(ctx, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unlink google account")

		storage.AssertExpectations(t)
		adapter.AssertExpectations(t)
	})
}

func TestOAuthService_AuthMethodMapping(t *testing.T) {
	t.Parallel()

	t.Run("maps provider IDs to auth methods correctly", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			providerID string
			authMethod string
		}{
			{"google", MethodOAuthGoogle},
			{"github", MethodOAuthGithub},
			{"custom-provider", "oauth_custom-provider"},
		}

		for _, tc := range testCases {
			t.Run(tc.providerID, func(t *testing.T) {
				storage := &MockOAuthStorage{}
				adapter := &MockProviderAdapter{}
				svc := NewOAuthService(storage, adapter)

				code := "auth-code"
				state := "valid-state"
				profile := ProviderProfile{
					ProviderUserID: "provider-user-123",
					Email:          "user@example.com",
					EmailVerified:  true,
				}

				adapter.On("ProviderID").Return(tc.providerID)
				adapter.On("ResolveProfile", mock.Anything, code).Return(profile, nil)

				storage.On("ConsumeState", mock.Anything, state).Return(nil)
				storage.On("GetUserByOAuth", mock.Anything, tc.providerID, "provider-user-123").Return(nil, ErrUserNotFound)
				storage.On("GetUserByEmail", mock.Anything, profile.Email).Return(nil, ErrUserNotFound)
				storage.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
					return u.AuthMethod == tc.authMethod
				})).Return(nil)
				storage.On("StoreOAuthLink", mock.Anything, mock.AnythingOfType("uuid.UUID"), tc.providerID, "provider-user-123").Return(nil)

				ctx := context.Background()
				user, err := svc.Auth(ctx, code, state, nil)

				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tc.authMethod, user.AuthMethod)

				storage.AssertExpectations(t)
				adapter.AssertExpectations(t)
			})
		}
	})
}

// Test that the service correctly implements the interface
func TestOAuthServiceInterface(t *testing.T) {
	t.Parallel()

	storage := &MockOAuthStorage{}
	adapter := &MockProviderAdapter{}
	var svc OAuthAuthenticator = NewOAuthService(storage, adapter)

	require.NotNil(t, svc)
	// If this compiles, the interface is correctly implemented
}
