package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLogger_WithMetadataFilter(t *testing.T) {
	t.Parallel()

	t.Run("removes PII fields", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		filter := NewMetadataFilter()
		logger := NewLogger(mockWriter, WithMetadataFilter(filter))

		ctx := context.Background()
		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			// Password should be removed
			_, hasPassword := event.Metadata["password"]
			// SSN should be masked
			ssn, hasSSN := event.Metadata["ssn"]
			// API secret should pass through (not in default PII)
			apiSecret, hasAPISecret := event.Metadata["api_secret"]
			// User ID should remain
			userID, hasUserID := event.Metadata["user_id"]

			ssnStr, ssnOk := ssn.(string)

			return !hasPassword &&
				hasSSN && ssnOk && ssnStr == "12*******89" &&
				hasAPISecret && apiSecret == "sk_secret_key" &&
				hasUserID && userID == "12345"
		})).Return(nil).Once()

		err := logger.Log(ctx, "user.login",
			WithMetadata("user_id", "12345"),
			WithMetadata("password", "secretpass123"),
			WithMetadata("ssn", "123-45-6789"),
			WithMetadata("api_secret", "sk_secret_key"),
		)
		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("masks sensitive data", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		filter := NewMetadataFilter()
		logger := NewLogger(mockWriter, WithMetadataFilter(filter))

		ctx := context.Background()
		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			// Check that sensitive fields are masked/hashed
			email, hasEmail := event.Metadata["email"]
			phone, hasPhone := event.Metadata["phone"]
			creditCard, hasCreditCard := event.Metadata["credit_card"]

			// Email should be hashed (64 char hex string)
			emailStr, emailOk := email.(string)
			// Phone should be masked
			phoneStr, phoneOk := phone.(string)
			// Credit card should be masked
			creditCardStr, creditCardOk := creditCard.(string)

			return hasEmail && emailOk && len(emailStr) == 64 &&
				hasPhone && phoneOk && phoneStr == "55********90" &&
				hasCreditCard && creditCardOk && creditCardStr == "41***************11"
		})).Return(nil).Once()

		err := logger.Log(ctx, "payment.process",
			WithMetadata("email", "user@example.com"),
			WithMetadata("phone", "555-123-7890"),
			WithMetadata("credit_card", "4111-1111-1111-1111"),
		)
		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("removes api keys and tokens", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		filter := NewMetadataFilter()
		logger := NewLogger(mockWriter, WithMetadataFilter(filter))

		ctx := context.Background()
		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			// API key and access token should be removed
			_, hasAPIKey := event.Metadata["api_key"]
			_, hasAccessToken := event.Metadata["access_token"]
			// IP address is not a PII field by default, should pass through
			ipAddress, hasIPAddress := event.Metadata["ip_address"]

			return !hasAPIKey && !hasAccessToken &&
				hasIPAddress && ipAddress == "192.168.1.100"
		})).Return(nil).Once()

		err := logger.Log(ctx, "api.request",
			WithMetadata("api_key", "sk_live_abc123"),
			WithMetadata("access_token", "ghu_token123"),
			WithMetadata("ip_address", "192.168.1.100"),
		)
		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("applies custom filter rules", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		filter := NewMetadataFilter(
			WithCustomField("internal_id", FilterActionHash),
			WithCustomField("custom_secret", FilterActionRemove),
			WithCustomField("*.token", FilterActionMask),
		)
		logger := NewLogger(mockWriter, WithMetadataFilter(filter))

		ctx := context.Background()
		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			// Check custom field filtering
			internalID, hasInternalID := event.Metadata["internal_id"]
			_, hasCustomSecret := event.Metadata["custom_secret"]
			authToken, hasAuthToken := event.Metadata["auth_token"]
			refreshToken, hasRefreshToken := event.Metadata["refresh_token"]

			internalIDStr, internalIDOk := internalID.(string)
			authTokenStr, authTokenOk := authToken.(string)
			refreshTokenStr, refreshTokenOk := refreshToken.(string)

			// Debug logging
			if !hasCustomSecret {
				t.Logf("✓ custom_secret removed as expected")
			}
			if hasInternalID && internalIDOk {
				t.Logf("internal_id: %s (len=%d)", internalIDStr, len(internalIDStr))
			}
			if hasAuthToken && authTokenOk {
				t.Logf("auth_token: '%s' (len=%d)", authTokenStr, len(authTokenStr))
			}
			if hasRefreshToken && refreshTokenOk {
				t.Logf("refresh_token: %s", refreshTokenStr)
			}

			// refresh_token: wildcard *.token rule applies, so it's masked
			result := hasInternalID && internalIDOk && len(internalIDStr) == 64 &&
				!hasCustomSecret &&
				hasAuthToken && authTokenOk && authTokenStr == "ey******************Sw" &&
				hasRefreshToken && refreshTokenOk && refreshTokenStr == "re***************ue"

			if !result {
				t.Logf("Test failed:")
				t.Logf("  hasInternalID=%v, internalIDOk=%v, len(internalIDStr)=%d (expect 64)", hasInternalID, internalIDOk, len(internalIDStr))
				t.Logf("  hasCustomSecret=%v (expect false)", hasCustomSecret)
				t.Logf("  hasAuthToken=%v, authTokenOk=%v", hasAuthToken, authTokenOk)
				t.Logf("  authTokenStr=='ey******************Sw' = %v", authTokenStr == "ey******************Sw")
				t.Logf("  hasRefreshToken=%v, refreshTokenOk=%v", hasRefreshToken, refreshTokenOk)
				t.Logf("  refreshTokenStr=='re***************ue' = %v", refreshTokenStr == "re***************ue")
			}
			return result
		})).Return(nil).Once()

		err := logger.Log(ctx, "custom.action",
			WithMetadata("internal_id", "INT-12345"),
			WithMetadata("custom_secret", "super-secret"),
			WithMetadata("auth_token", "eyJhbGciOiJIUzI1NiIsSw"),
			WithMetadata("refresh_token", "refresh_token_value"),
		)
		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("allows specified fields to bypass filtering", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		filter := NewMetadataFilter(
			WithAllowedField("email"),
			WithAllowedField("phone"),
		)
		logger := NewLogger(mockWriter, WithMetadataFilter(filter))

		ctx := context.Background()
		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			// Email and phone should NOT be filtered because they're allowed
			email, hasEmail := event.Metadata["email"]
			phone, hasPhone := event.Metadata["phone"]
			// But password should still be removed
			_, hasPassword := event.Metadata["password"]

			return hasEmail && email == "user@example.com" &&
				hasPhone && phone == "555-123-4567" &&
				!hasPassword
		})).Return(nil).Once()

		err := logger.Log(ctx, "user.profile",
			WithMetadata("email", "user@example.com"),
			WithMetadata("phone", "555-123-4567"),
			WithMetadata("password", "should-be-removed"),
		)
		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("passes metadata unchanged without filter", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		// Logger without filter
		logger := NewLogger(mockWriter)

		ctx := context.Background()
		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			// All fields should be present unchanged
			password, hasPassword := event.Metadata["password"]
			email, hasEmail := event.Metadata["email"]
			apiKey, hasAPIKey := event.Metadata["api_key"]

			return hasPassword && password == "secret123" &&
				hasEmail && email == "user@example.com" &&
				hasAPIKey && apiKey == "sk_live_key"
		})).Return(nil).Once()

		err := logger.Log(ctx, "data.export",
			WithMetadata("password", "secret123"),
			WithMetadata("email", "user@example.com"),
			WithMetadata("api_key", "sk_live_key"),
		)
		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("LogError also applies filtering", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		filter := NewMetadataFilter()
		logger := NewLogger(mockWriter, WithMetadataFilter(filter))

		ctx := context.Background()
		testErr := errors.New("authentication failed")

		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			// Password should be removed even in error logs
			_, hasPassword := event.Metadata["password"]
			username, hasUsername := event.Metadata["username"]

			return event.Error == "authentication failed" &&
				!hasPassword &&
				hasUsername && username == "john.doe"
		})).Return(nil).Once()

		err := logger.LogError(ctx, "auth.failed", testErr,
			WithMetadata("username", "john.doe"),
			WithMetadata("password", "wrong-password"),
		)
		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("filter works with nested data structures", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		filter := NewMetadataFilter()
		logger := NewLogger(mockWriter, WithMetadataFilter(filter))

		ctx := context.Background()
		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			// Nested structures should be preserved as-is since they're not PII fields
			userData, hasUserData := event.Metadata["user_data"]
			config, hasConfig := event.Metadata["config"]

			// The nested structure should remain unchanged since "user_data" is not a PII field
			userDataMap, isMap := userData.(map[string]interface{})

			return hasUserData && isMap &&
				userDataMap["name"] == "John Doe" &&
				userDataMap["password"] == "secret" &&
				hasConfig && config == "safe-config"
		})).Return(nil).Once()

		err := logger.Log(ctx, "data.process",
			WithMetadata("user_data", map[string]interface{}{
				"name":     "John Doe",
				"password": "secret",
			}),
			WithMetadata("config", "safe-config"),
		)
		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("filter handles nil metadata gracefully", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		filter := NewMetadataFilter()
		logger := NewLogger(mockWriter, WithMetadataFilter(filter))

		ctx := context.Background()
		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			return event.Metadata == nil
		})).Return(nil).Once()

		err := logger.Log(ctx, "empty.action")
		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})
}

// TestAsyncLogger_MetadataFiltering verifies that async logger applies filtering
func TestAsyncLogger_MetadataFiltering(t *testing.T) {
	t.Parallel()

	mockBW := &MockBatchWriter{}
	filter := NewMetadataFilter()

	// Create async logger with filter
	logger, closeFunc := NewAsyncLogger(mockBW, 100, WithMetadataFilter(filter))

	ctx := context.Background()

	// We need to use a channel to wait for async processing
	done := make(chan bool, 1) // Buffered channel to avoid blocking
	mockBW.On("StoreBatch", mock.Anything, mock.MatchedBy(func(events []Event) bool {
		if len(events) == 0 {
			return false
		}
		event := events[0]
		// Password should be removed
		_, hasPassword := event.Metadata["password"]
		userID, hasUserID := event.Metadata["user_id"]

		result := !hasPassword && hasUserID && userID == "async-user"
		if result {
			select {
			case done <- true:
			default:
				// Channel already has a value, ignore
			}
		}
		return result
	})).Return(nil).Maybe() // May be called multiple times due to async nature

	err := logger.Log(ctx, "async.action",
		WithMetadata("user_id", "async-user"),
		WithMetadata("password", "should-be-removed"),
	)
	assert.NoError(t, err)

	// Wait for async processing
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for async log")
	}

	// Close the logger and wait for it to finish
	err = closeFunc(ctx)
	assert.NoError(t, err)

	mockBW.AssertExpectations(t)
}
