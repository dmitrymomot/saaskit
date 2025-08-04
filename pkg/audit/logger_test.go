package audit

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWriter implements the writer interface for testing
type MockWriter struct {
	mock.Mock
}

func (m *MockWriter) Store(ctx context.Context, event Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestNewLogger(t *testing.T) {
	t.Parallel()

	t.Run("creates logger with writer", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		logger := NewLogger(mockWriter)

		assert.NotNil(t, logger)
		assert.Equal(t, mockWriter, logger.writer)
	})

	t.Run("applies options during creation", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}
		tenantExtractor := func(ctx context.Context) (string, bool) {
			return "tenant-123", true
		}

		logger := NewLogger(mockWriter, WithTenantIDExtractor(tenantExtractor))

		assert.NotNil(t, logger)
		assert.NotNil(t, logger.tenantIDExtractor)

		// Test that extractor was set correctly
		tenant, ok := logger.tenantIDExtractor(context.Background())
		assert.True(t, ok)
		assert.Equal(t, "tenant-123", tenant)
	})

	t.Run("panics with nil writer", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			NewLogger(nil)
		})
	})

	t.Run("applies multiple options", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		logger := NewLogger(mockWriter,
			WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
				return "tenant", true
			}),
			WithUserIDExtractor(func(ctx context.Context) (string, bool) {
				return "user", true
			}),
		)

		assert.NotNil(t, logger.tenantIDExtractor)
		assert.NotNil(t, logger.userIDExtractor)
	})
}

func TestLogger_Log(t *testing.T) {
	t.Parallel()

	t.Run("logs successful action", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}
		logger := NewLogger(mockWriter)
		ctx := context.Background()

		// Expect Store to be called with proper event
		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			return event.Action == "user.login" &&
				event.Result == ResultSuccess &&
				event.ID != "" && // ID should be generated
				!event.CreatedAt.IsZero() // CreatedAt should be set
		})).Return(nil)

		err := logger.Log(ctx, "user.login")

		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("applies event options", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}
		logger := NewLogger(mockWriter)
		ctx := context.Background()

		expectedResource := "document"
		expectedResourceID := "doc-123"

		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			return event.Action == "document.create" &&
				event.Resource == expectedResource &&
				event.ResourceID == expectedResourceID &&
				event.Metadata["size"] == 1024
		})).Return(nil)

		err := logger.Log(ctx, "document.create",
			WithResource(expectedResource, expectedResourceID),
			WithMetadata("size", 1024),
		)

		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("extracts context values", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		logger := NewLogger(mockWriter,
			WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
				if val := ctx.Value("tenant_id"); val != nil {
					return val.(string), true
				}
				return "", false
			}),
			WithUserIDExtractor(func(ctx context.Context) (string, bool) {
				if val := ctx.Value("user_id"); val != nil {
					return val.(string), true
				}
				return "", false
			}),
		)

		ctx := context.WithValue(context.Background(), "tenant_id", "tenant-123")
		ctx = context.WithValue(ctx, "user_id", "user-456")

		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			return event.TenantID == "tenant-123" &&
				event.UserID == "user-456"
		})).Return(nil)

		err := logger.Log(ctx, "test.action")

		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("returns validation error for invalid event", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}
		logger := NewLogger(mockWriter)
		ctx := context.Background()

		// Event will be invalid due to empty action (though we pass action,
		// we could test with an option that clears it)
		err := logger.Log(ctx, "", WithResult(ResultSuccess))

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrEventValidation)
		mockWriter.AssertNotCalled(t, "Store")
	})

	t.Run("returns storage error", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}
		logger := NewLogger(mockWriter)
		ctx := context.Background()

		storageErr := errors.New("storage failed")
		mockWriter.On("Store", mock.Anything, mock.Anything).Return(storageErr)

		err := logger.Log(ctx, "test.action")

		assert.Error(t, err)
		assert.Equal(t, storageErr, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("generates unique IDs for concurrent calls", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}
		logger := NewLogger(mockWriter)
		ctx := context.Background()

		mockWriter.On("Store", mock.Anything, mock.Anything).Return(nil)

		const numCalls = 100
		done := make(chan bool, numCalls)

		for range numCalls {
			go func() {
				defer func() { done <- true }()

				// We can't easily capture the ID from the mock call in a race-safe way
				// This test mainly ensures no panics occur during concurrent access
				err := logger.Log(ctx, "concurrent.test")
				assert.NoError(t, err)
			}()
		}

		// Wait for all goroutines to complete
		for range numCalls {
			<-done
		}

		mockWriter.AssertExpectations(t)
		assert.Equal(t, numCalls, len(mockWriter.Calls))
	})
}

func TestLogger_LogError(t *testing.T) {
	t.Parallel()

	t.Run("logs error action", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}
		logger := NewLogger(mockWriter)
		ctx := context.Background()
		testErr := errors.New("test error")

		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			return event.Action == "user.login" &&
				event.Result == ResultError &&
				event.Error == "test error" &&
				event.ID != "" &&
				!event.CreatedAt.IsZero()
		})).Return(nil)

		err := logger.LogError(ctx, "user.login", testErr)

		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("applies event options to error event", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}
		logger := NewLogger(mockWriter)
		ctx := context.Background()
		testErr := errors.New("validation failed")

		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			return event.Action == "user.create" &&
				event.Result == ResultError &&
				event.Error == "validation failed" &&
				event.Resource == "user" &&
				event.ResourceID == "user-123"
		})).Return(nil)

		err := logger.LogError(ctx, "user.create", testErr,
			WithResource("user", "user-123"),
		)

		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("can override result with options", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}
		logger := NewLogger(mockWriter)
		ctx := context.Background()
		testErr := errors.New("test error")

		// Even though LogError sets ResultError, WithResult can override it
		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			return event.Result == ResultFailure // Overridden by option
		})).Return(nil)

		err := logger.LogError(ctx, "test.action", testErr, WithResult(ResultFailure))

		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("extracts context values for error events", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		logger := NewLogger(mockWriter,
			WithRequestIDExtractor(func(ctx context.Context) (string, bool) {
				if val := ctx.Value("request_id"); val != nil {
					return val.(string), true
				}
				return "", false
			}),
		)

		ctx := context.WithValue(context.Background(), "request_id", "req-789")
		testErr := errors.New("processing failed")

		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			return event.RequestID == "req-789" &&
				event.Error == "processing failed"
		})).Return(nil)

		err := logger.LogError(ctx, "process.data", testErr)

		assert.NoError(t, err)
		mockWriter.AssertExpectations(t)
	})

	t.Run("handles nil error", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}
		logger := NewLogger(mockWriter)
		ctx := context.Background()

		// This tests edge case behavior - what happens with nil error
		// Current implementation will panic on err.Error(), which is reasonable
		assert.Panics(t, func() {
			logger.LogError(ctx, "test.action", nil)
		})
	})
}

func TestLogger_eventFromContext(t *testing.T) {
	t.Parallel()

	t.Run("returns empty event when no extractors configured", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}
		logger := NewLogger(mockWriter)
		ctx := context.Background()

		event := logger.eventFromContext(ctx)

		assert.Equal(t, Event{}, event)
	})

	t.Run("extracts all configured context values", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		logger := NewLogger(mockWriter,
			WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
				return "tenant-123", true
			}),
			WithUserIDExtractor(func(ctx context.Context) (string, bool) {
				return "user-456", true
			}),
			WithSessionIDExtractor(func(ctx context.Context) (string, bool) {
				return "session-789", true
			}),
			WithRequestIDExtractor(func(ctx context.Context) (string, bool) {
				return "req-abc", true
			}),
			WithIPExtractor(func(ctx context.Context) (string, bool) {
				return "192.168.1.1", true
			}),
			WithUserAgentExtractor(func(ctx context.Context) (string, bool) {
				return "Mozilla/5.0", true
			}),
		)

		event := logger.eventFromContext(context.Background())

		assert.Equal(t, "tenant-123", event.TenantID)
		assert.Equal(t, "user-456", event.UserID)
		assert.Equal(t, "session-789", event.SessionID)
		assert.Equal(t, "req-abc", event.RequestID)
		assert.Equal(t, "192.168.1.1", event.IP)
		assert.Equal(t, "Mozilla/5.0", event.UserAgent)
	})

	t.Run("handles extractor failures gracefully", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		logger := NewLogger(mockWriter,
			WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
				return "", false // Extraction failed
			}),
			WithUserIDExtractor(func(ctx context.Context) (string, bool) {
				return "user-456", true // Extraction succeeded
			}),
		)

		event := logger.eventFromContext(context.Background())

		assert.Equal(t, "", event.TenantID)       // Failed extraction
		assert.Equal(t, "user-456", event.UserID) // Successful extraction
	})

	t.Run("handles nil extractors", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}
		logger := NewLogger(mockWriter)

		// Manually set some extractors to nil to test the nil checks
		logger.tenantIDExtractor = nil
		logger.userIDExtractor = nil

		event := logger.eventFromContext(context.Background())

		assert.Equal(t, "", event.TenantID)
		assert.Equal(t, "", event.UserID)
	})

	t.Run("extractors receive correct context", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		var receivedCtx context.Context
		logger := NewLogger(mockWriter,
			WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
				receivedCtx = ctx
				return "tenant", true
			}),
		)

		inputCtx := context.WithValue(context.Background(), "test", "value")
		logger.eventFromContext(inputCtx)

		assert.Equal(t, inputCtx, receivedCtx)
	})
}

func TestLogger_Integration(t *testing.T) {
	t.Parallel()

	t.Run("full audit workflow", func(t *testing.T) {
		t.Parallel()
		mockWriter := &MockWriter{}

		// Setup logger with all extractors
		logger := NewLogger(mockWriter,
			WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
				if val := ctx.Value("tenant"); val != nil {
					return val.(string), true
				}
				return "", false
			}),
			WithUserIDExtractor(func(ctx context.Context) (string, bool) {
				if val := ctx.Value("user"); val != nil {
					return val.(string), true
				}
				return "", false
			}),
		)

		// Setup context
		ctx := context.WithValue(context.Background(), "tenant", "acme-corp")
		ctx = context.WithValue(ctx, "user", "john-doe")

		// Test successful action
		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			return event.Action == "document.create" &&
				event.Result == ResultSuccess &&
				event.TenantID == "acme-corp" &&
				event.UserID == "john-doe" &&
				event.Resource == "document" &&
				event.ResourceID == "doc-456" &&
				event.Metadata["size"] == 2048
		})).Return(nil).Once()

		err := logger.Log(ctx, "document.create",
			WithResource("document", "doc-456"),
			WithMetadata("size", 2048),
		)
		assert.NoError(t, err)

		// Test error action
		testErr := errors.New("insufficient permissions")
		mockWriter.On("Store", mock.Anything, mock.MatchedBy(func(event Event) bool {
			return event.Action == "document.delete" &&
				event.Result == ResultError &&
				event.Error == "insufficient permissions" &&
				event.TenantID == "acme-corp" &&
				event.UserID == "john-doe"
		})).Return(nil).Once()

		err = logger.LogError(ctx, "document.delete", testErr)
		assert.NoError(t, err)

		mockWriter.AssertExpectations(t)
	})
}
