package audit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewAsyncLogger(t *testing.T) {
	t.Parallel()

	t.Run("creates async logger with default buffer size", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		bufferSize := 1000

		logger, closeFunc := NewAsyncLogger(mockBW, bufferSize)

		assert.NotNil(t, logger)
		assert.NotNil(t, closeFunc)

		// Clean up
		err := closeFunc(context.Background())
		assert.NoError(t, err)
	})

	t.Run("creates async logger with options", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		bufferSize := 500

		tenantExtractor := func(ctx context.Context) (string, bool) {
			return "test-tenant", true
		}

		logger, closeFunc := NewAsyncLogger(mockBW, bufferSize,
			WithTenantIDExtractor(tenantExtractor),
		)

		assert.NotNil(t, logger)

		// Test that the option was applied
		event := logger.eventFromContext(context.Background())
		assert.Equal(t, "test-tenant", event.TenantID)

		// Clean up
		err := closeFunc(context.Background())
		assert.NoError(t, err)
	})

	t.Run("creates async logger with custom buffer size", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		bufferSize := 2500

		logger, closeFunc := NewAsyncLogger(mockBW, bufferSize)

		assert.NotNil(t, logger)

		// Clean up
		err := closeFunc(context.Background())
		assert.NoError(t, err)
	})
}

func TestAsyncLogger_Integration(t *testing.T) {
	t.Parallel()

	t.Run("logs events through async writer", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		mockBW.On("StoreBatch", mock.Anything, mock.MatchedBy(func(events []Event) bool {
			return len(events) == 1 &&
				events[0].Action == "async.test" &&
				events[0].Result == ResultSuccess
		})).Return(nil)

		logger, closeFunc := NewAsyncLogger(mockBW, 100)
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		err := logger.Log(context.Background(), "async.test")
		assert.NoError(t, err)

		// Give async writer time to process
		time.Sleep(10 * time.Millisecond)
		mockBW.AssertExpectations(t)
	})

	t.Run("logs error events through async writer", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		testErr := errors.New("async error")

		mockBW.On("StoreBatch", mock.Anything, mock.MatchedBy(func(events []Event) bool {
			return len(events) == 1 &&
				events[0].Action == "async.error" &&
				events[0].Result == ResultError &&
				events[0].Error == "async error"
		})).Return(nil)

		logger, closeFunc := NewAsyncLogger(mockBW, 100)
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		err := logger.LogError(context.Background(), "async.error", testErr)
		assert.NoError(t, err)

		// Give async writer time to process
		time.Sleep(10 * time.Millisecond)
		mockBW.AssertExpectations(t)
	})

	t.Run("batches multiple log entries", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}

		// Expect batches to be created
		mockBW.On("StoreBatch", mock.Anything, mock.MatchedBy(func(events []Event) bool {
			// Should receive batches with multiple events
			return len(events) >= 1
		})).Return(nil)

		// Use small batch size for quick testing
		asyncWriter, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{
			BufferSize:   50,
			BatchSize:    3,
			BatchTimeout: 100 * time.Millisecond,
		})
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		logger := NewLogger(asyncWriter)

		// Send multiple events quickly
		for i := range 5 {
			err := logger.Log(context.Background(), "batch.test",
				WithMetadata("index", i))
			assert.NoError(t, err)
		}

		// Give time for batching
		time.Sleep(50 * time.Millisecond)
		mockBW.AssertExpectations(t)
	})

	t.Run("handles high concurrency with async logger", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		mockBW.On("StoreBatch", mock.Anything, mock.Anything).Return(nil)

		logger, closeFunc := NewAsyncLogger(mockBW, 1000,
			WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
				if val := ctx.Value("tenant"); val != nil {
					return val.(string), true
				}
				return "", false
			}),
		)
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		const numGoroutines = 20
		const eventsPerGoroutine = 25

		var wg sync.WaitGroup
		errorsChan := make(chan error, numGoroutines*eventsPerGoroutine)

		wg.Add(numGoroutines)
		for i := range numGoroutines {
			go func(routineID int) {
				defer wg.Done()

				ctx := context.WithValue(context.Background(), "tenant", "tenant-"+string(rune(routineID)))

				for j := range eventsPerGoroutine {
					if j%2 == 0 {
						// Mix of Log and LogError calls
						err := logger.Log(ctx, "concurrent.success",
							WithMetadata("routine", routineID),
							WithMetadata("event", j),
						)
						if err != nil {
							errorsChan <- err
						}
					} else {
						testErr := errors.New("concurrent error")
						err := logger.LogError(ctx, "concurrent.error", testErr,
							WithMetadata("routine", routineID),
							WithMetadata("event", j),
						)
						if err != nil {
							errorsChan <- err
						}
					}
				}
			}(i)
		}

		wg.Wait()
		close(errorsChan)

		// Verify no errors occurred
		for err := range errorsChan {
			assert.NoError(t, err)
		}

		// Give time for final batch processing
		time.Sleep(100 * time.Millisecond)

		// Verify batches were called
		assert.True(t, mockBW.GetCallCount() > 0)
	})

	t.Run("graceful shutdown flushes pending events", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}

		// Track events to ensure they're flushed on shutdown
		var eventsMutex sync.Mutex
		var receivedEvents []Event

		mockBW.On("StoreBatch", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			events := args.Get(1).([]Event)
			eventsMutex.Lock()
			receivedEvents = append(receivedEvents, events...)
			eventsMutex.Unlock()
		}).Return(nil)

		logger, closeFunc := NewAsyncLogger(mockBW, 100,
			WithUserIDExtractor(func(ctx context.Context) (string, bool) {
				return "shutdown-user", true
			}),
		)

		// Send events that won't be flushed immediately due to large batch size
		ctx := context.Background()

		err1 := logger.Log(ctx, "shutdown.test1", WithMetadata("order", 1))
		assert.NoError(t, err1)

		testErr := errors.New("shutdown error")
		err2 := logger.LogError(ctx, "shutdown.test2", testErr, WithMetadata("order", 2))
		assert.NoError(t, err2)

		err3 := logger.Log(ctx, "shutdown.test3", WithMetadata("order", 3))
		assert.NoError(t, err3)

		// Close should flush all pending events
		err := closeFunc(context.Background())
		assert.NoError(t, err)

		// Verify all events were processed
		eventsMutex.Lock()
		defer eventsMutex.Unlock()

		require.Len(t, receivedEvents, 3)

		// Verify events have correct data
		actions := make([]string, len(receivedEvents))
		for i, event := range receivedEvents {
			actions[i] = event.Action
			assert.Equal(t, "shutdown-user", event.UserID)
		}

		assert.Contains(t, actions, "shutdown.test1")
		assert.Contains(t, actions, "shutdown.test2")
		assert.Contains(t, actions, "shutdown.test3")
	})

	t.Run("handles storage errors in async logger", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		storageErr := errors.New("async storage failed")
		mockBW.On("StoreBatch", mock.Anything, mock.Anything).Return(storageErr)

		logger, closeFunc := NewAsyncLogger(mockBW, 100)
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		err := logger.Log(context.Background(), "error.test")

		// Error should be propagated back from async writer
		assert.Error(t, err)
		assert.Equal(t, storageErr, err)
	})

	t.Run("async logger with all context extractors", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}

		// Capture the event to verify all fields are set
		var capturedEvent Event
		mockBW.On("StoreBatch", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			events := args.Get(1).([]Event)
			if len(events) > 0 {
				capturedEvent = events[0]
			}
		}).Return(nil)

		logger, closeFunc := NewAsyncLogger(mockBW, 100,
			WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
				return "async-tenant", true
			}),
			WithUserIDExtractor(func(ctx context.Context) (string, bool) {
				return "async-user", true
			}),
			WithSessionIDExtractor(func(ctx context.Context) (string, bool) {
				return "async-session", true
			}),
			WithRequestIDExtractor(func(ctx context.Context) (string, bool) {
				return "async-request", true
			}),
			WithIPExtractor(func(ctx context.Context) (string, bool) {
				return "192.168.1.100", true
			}),
			WithUserAgentExtractor(func(ctx context.Context) (string, bool) {
				return "AsyncBot/1.0", true
			}),
		)
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		err := logger.Log(context.Background(), "full.context.test",
			WithResource("document", "doc-async-123"),
			WithMetadata("async", true),
		)
		assert.NoError(t, err)

		// Give time for processing
		time.Sleep(20 * time.Millisecond)

		// Verify all fields were populated
		assert.Equal(t, "full.context.test", capturedEvent.Action)
		assert.Equal(t, "async-tenant", capturedEvent.TenantID)
		assert.Equal(t, "async-user", capturedEvent.UserID)
		assert.Equal(t, "async-session", capturedEvent.SessionID)
		assert.Equal(t, "async-request", capturedEvent.RequestID)
		assert.Equal(t, "192.168.1.100", capturedEvent.IP)
		assert.Equal(t, "AsyncBot/1.0", capturedEvent.UserAgent)
		assert.Equal(t, "document", capturedEvent.Resource)
		assert.Equal(t, "doc-async-123", capturedEvent.ResourceID)
		assert.Equal(t, ResultSuccess, capturedEvent.Result)
		assert.Equal(t, true, capturedEvent.Metadata["async"])
		assert.NotEmpty(t, capturedEvent.ID)
		assert.False(t, capturedEvent.CreatedAt.IsZero())

		mockBW.AssertExpectations(t)
	})
}
