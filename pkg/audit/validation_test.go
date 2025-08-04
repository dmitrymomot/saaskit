package audit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/audit"
)

// Use shared MockStorage from logger_test.go to avoid redeclaration

func TestLogger_StorageFailureHandling(t *testing.T) {
	t.Parallel()

	// BUSINESS LOGIC: Logger must handle storage failures gracefully
	// and provide clear error responses for audit compliance requirements
	storage := new(MockStorage)
	storage.On("Store", mock.Anything, mock.Anything).Return(nil).Once() // health check

	// Simulate storage failure
	storageErr := errors.New("database connection lost")
	storage.On("Store", mock.Anything, mock.Anything).Return(storageErr)

	logger := audit.NewLogger(storage)

	// Try to log an event
	err := logger.Log(context.Background(), "user.data_access",
		audit.WithMetadata("record_id", 123),
	)

	// Should propagate storage error
	require.Error(t, err)
	assert.Equal(t, storageErr, err)

	storage.AssertExpectations(t)
}

func TestLogger_ContextTimeoutHandling(t *testing.T) {
	t.Parallel()

	// BUSINESS LOGIC: Context timeouts must be handled properly to prevent
	// request hanging in web applications
	storage := new(MockStorage)
	storage.On("Store", mock.Anything, mock.Anything).Return(nil).Once() // health check

	// Simulate slow storage that would timeout by returning an error that matches context timeout
	storage.On("Store", mock.Anything, mock.Anything).
		Return(context.DeadlineExceeded).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			select {
			case <-time.After(200 * time.Millisecond):
				// Slow operation
			case <-ctx.Done():
				// Context cancelled/timeout - this is what we expect
			}
		})

	logger := audit.NewLogger(storage)

	// Use a request timeout context (common in web handlers)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := logger.Log(ctx, "user.sensitive_action")
	duration := time.Since(start)

	// Must timeout quickly and return proper error
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, duration, 100*time.Millisecond, "Must respect timeout")
}

func TestLogger_LargeMetadataHandling(t *testing.T) {
	t.Parallel()

	// BUSINESS LOGIC: System must handle large metadata without crashing
	// Important for detailed audit trails with complex data structures
	storage := new(MockStorage)
	storage.On("Store", mock.Anything, mock.Anything).Return(nil).Once() // health check

	// Verify large metadata is handled correctly
	storage.On("Store", mock.Anything, mock.MatchedBy(func(events []audit.Event) bool {
		if len(events) != 1 {
			return false
		}
		e := events[0]
		return e.Action == "bulk.data_export" &&
			len(e.Metadata) == 2 && // Should contain our metadata
			e.Metadata["export_size"] != nil &&
			e.Metadata["data_classification"] != nil
	})).Return(nil).Once()

	logger := audit.NewLogger(storage)

	err := logger.Log(context.Background(), "bulk.data_export",
		audit.WithMetadata("export_size", 1024*1024), // 1MB export
		audit.WithMetadata("data_classification", "confidential"),
	)
	require.NoError(t, err)

	storage.AssertExpectations(t)
}

func TestLogger_HealthCheckFailurePreventsCreation(t *testing.T) {
	t.Parallel()

	// BUSINESS LOGIC: Constructor must fail fast if storage is completely broken
	// This prevents runtime surprises when audit logging is first attempted
	storage := new(MockStorage)
	storage.On("Store", mock.Anything, mock.MatchedBy(func(events []audit.Event) bool {
		return len(events) == 1 && events[0].Action == "audit.health_check"
	})).Return(errors.New("storage completely unavailable"))

	// Constructor should panic if storage health check fails
	assert.Panics(t, func() {
		audit.NewLogger(storage)
	}, "Constructor must fail fast if storage is unavailable")

	storage.AssertExpectations(t)
}
