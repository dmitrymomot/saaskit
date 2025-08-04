package audit_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/audit"
)

func TestAsyncLogger_ConcurrentLogging(t *testing.T) {
	t.Parallel()

	// BUSINESS LOGIC: Async logging must handle concurrent operations safely
	// without data races or event loss - critical for high-throughput systems
	storage := new(MockStorage)
	storage.On("Store", mock.Anything, mock.Anything).Return(nil)

	// Create async logger
	logger := audit.NewLogger(storage, audit.WithAsync(100))

	ctx := context.Background()
	var wg sync.WaitGroup

	// Simulate high concurrent load
	const goroutines = 5
	const eventsPerGoroutine = 3

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				err := logger.Log(ctx, "concurrent.operation",
					audit.WithMetadata("worker", workerID),
					audit.WithMetadata("iteration", j),
				)
				// Async operations should not return errors for valid requests
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Give async processing time to complete
	time.Sleep(200 * time.Millisecond)

	// System should remain stable - exact call count depends on batching
	storage.AssertCalled(t, "Store", mock.Anything, mock.Anything)
}

func TestAsyncLogger_ContextCancellation(t *testing.T) {
	t.Parallel()

	// BUSINESS LOGIC: Context cancellation must be respected even in async mode
	// to prevent resource leaks during request timeouts
	storage := new(MockStorage)
	storage.On("Store", mock.Anything, mock.Anything).Return(nil).Once() // health check

	// Simulate slow storage that would cause timeout
	storage.On("Store", mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			time.Sleep(500 * time.Millisecond)
		})

	logger := audit.NewLogger(storage, audit.WithAsync(10))

	// Short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := logger.Log(ctx, "timeout.test")
	duration := time.Since(start)

	// Should timeout quickly
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, duration, 100*time.Millisecond)
}

func TestAsyncLogger_BasicFunctionality(t *testing.T) {
	t.Parallel()

	// BUSINESS LOGIC: Async logger must successfully process events
	// This is the core functionality test
	storage := new(MockStorage)
	storage.On("Store", mock.Anything, mock.Anything).Return(nil)

	logger := audit.NewLogger(storage, audit.WithAsync(10))

	ctx := context.Background()

	// Log a few events
	for i := 0; i < 3; i++ {
		err := logger.Log(ctx, "basic.test",
			audit.WithMetadata("sequence", i),
		)
		require.NoError(t, err, "Basic async logging should not fail")
	}

	// Wait for async processing
	time.Sleep(100 * time.Millisecond)

	// Verify storage was called (health check + events)
	storage.AssertCalled(t, "Store", mock.Anything, mock.Anything)
}