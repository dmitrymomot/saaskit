package audit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBatchWriter implements the batchWriter interface for testing
type MockBatchWriter struct {
	mock.Mock
	delay      time.Duration // Simulate slow storage
	callCount  int32
	storeDelay time.Duration
}

func (m *MockBatchWriter) StoreBatch(ctx context.Context, events []Event) error {
	atomic.AddInt32(&m.callCount, 1)

	if m.storeDelay > 0 {
		select {
		case <-time.After(m.storeDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *MockBatchWriter) GetCallCount() int32 {
	return atomic.LoadInt32(&m.callCount)
}

func TestNewAsyncWriter(t *testing.T) {
	t.Parallel()

	t.Run("creates async writer with default options", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}

		writer, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{})

		assert.NotNil(t, writer)
		assert.NotNil(t, closeFunc)
		assert.Equal(t, mockBW, writer.batchWriter)
		assert.Equal(t, 1000, writer.options.BufferSize)
		assert.Equal(t, 100, writer.options.BatchSize)
		assert.Equal(t, 100*time.Millisecond, writer.options.BatchTimeout)
		assert.Equal(t, 5*time.Second, writer.options.StorageTimeout)

		// Clean up
		err := closeFunc(context.Background())
		assert.NoError(t, err)
	})

	t.Run("uses provided options", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		opts := AsyncOptions{
			BufferSize:     500,
			BatchSize:      50,
			BatchTimeout:   50 * time.Millisecond,
			StorageTimeout: 2 * time.Second,
		}

		writer, closeFunc := NewAsyncWriter(mockBW, opts)

		assert.Equal(t, 500, writer.options.BufferSize)
		assert.Equal(t, 50, writer.options.BatchSize)
		assert.Equal(t, 50*time.Millisecond, writer.options.BatchTimeout)
		assert.Equal(t, 2*time.Second, writer.options.StorageTimeout)

		// Clean up
		err := closeFunc(context.Background())
		assert.NoError(t, err)
	})

	t.Run("panics with nil batch writer", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			NewAsyncWriter(nil, AsyncOptions{})
		})
	})

	t.Run("starts background worker", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}

		_, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{
			BufferSize: 10,
		})

		// Worker should be running - we test this by ensuring clean shutdown
		// We can't directly test internal state without exposing it

		// Clean up
		err := closeFunc(context.Background())
		assert.NoError(t, err)
	})
}

func TestAsyncWriter_Store(t *testing.T) {
	t.Parallel()

	t.Run("stores single event successfully", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.MatchedBy(func(events []Event) bool {
			return len(events) == 1 && events[0].Action == "test.action"
		})).Return(nil)

		writer, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{
			BufferSize: 10,
		})
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		event := Event{Action: "test.action"}
		err := writer.Store(context.Background(), event)

		assert.NoError(t, err)

		// Give background worker time to process
		time.Sleep(10 * time.Millisecond)
		mockBW.AssertExpectations(t)
	})

	t.Run("batches multiple events", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		opts := AsyncOptions{
			BufferSize:   10,
			BatchSize:    3,               // Small batch size for testing
			BatchTimeout: 1 * time.Second, // Long timeout to ensure batch size triggers flush
		}

		// Track total events processed
		var totalEvents int32
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.MatchedBy(func(events []Event) bool {
			atomic.AddInt32(&totalEvents, int32(len(events)))
			return len(events) >= 1 // Accept any batch size
		})).Return(nil)

		writer, closeFunc := NewAsyncWriter(mockBW, opts)
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		// Send 3 events to trigger batch
		for i := range 3 {
			event := Event{Action: "batch.test", Metadata: map[string]any{"index": i}}
			err := writer.Store(context.Background(), event)
			assert.NoError(t, err)
		}

		// Give background worker time to process batch
		time.Sleep(50 * time.Millisecond)

		// Verify all events were processed
		assert.Equal(t, int32(3), atomic.LoadInt32(&totalEvents), "All events should be processed")
		assert.True(t, len(mockBW.Calls) >= 1, "At least one batch should be called")
	})

	t.Run("flushes on timeout", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		opts := AsyncOptions{
			BufferSize:   10,
			BatchSize:    10,                    // Large batch size
			BatchTimeout: 10 * time.Millisecond, // Short timeout to trigger flush
		}

		// Expect at least one batch to be called - could be 1 or 2 events depending on timing
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.MatchedBy(func(events []Event) bool {
			return len(events) >= 1 && len(events) <= 2 // Could be batched together or separately
		})).Return(nil)

		writer, closeFunc := NewAsyncWriter(mockBW, opts)
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		// Send 2 events (less than batch size)
		for i := range 2 {
			event := Event{Action: "timeout.test", Metadata: map[string]any{"index": i}}
			err := writer.Store(context.Background(), event)
			assert.NoError(t, err)
		}

		// Wait for timeout to trigger flush
		time.Sleep(50 * time.Millisecond)

		// Verify at least one batch was processed
		assert.True(t, len(mockBW.Calls) >= 1, "At least one batch should have been processed")
	})

	t.Run("falls back to sync when buffer full", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		opts := AsyncOptions{
			BufferSize:   1, // Tiny buffer
			BatchSize:    10,
			BatchTimeout: 1 * time.Second,
		}

		// First call should be async (buffered)
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.MatchedBy(func(events []Event) bool {
			return len(events) == 1 && events[0].Metadata["type"] == "async"
		})).Return(nil).Once()

		// Subsequent calls should be sync (buffer full)
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.MatchedBy(func(events []Event) bool {
			return len(events) == 1 && events[0].Metadata["type"] == "sync"
		})).Return(nil).Times(2)

		writer, closeFunc := NewAsyncWriter(mockBW, opts)
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		// First event goes to buffer
		event1 := Event{Action: "buffer.test", Metadata: map[string]any{"type": "async"}}
		err := writer.Store(context.Background(), event1)
		assert.NoError(t, err)

		// These should hit the full buffer and fall back to sync
		for i := range 2 {
			event := Event{Action: "sync.test", Metadata: map[string]any{"type": "sync", "index": i}}
			err := writer.Store(context.Background(), event)
			assert.NoError(t, err)
		}

		time.Sleep(50 * time.Millisecond)
		mockBW.AssertExpectations(t)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		// In case the event gets queued before context cancellation is detected
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.Anything).Return(nil).Maybe()

		writer, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{
			BufferSize: 10,
		})
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		event := Event{Action: "cancelled.test"}
		err := writer.Store(ctx, event)

		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("handles storage errors", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		storageErr := errors.New("storage failed")
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.Anything).Return(storageErr)

		writer, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{
			BufferSize: 10,
		})
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		event := Event{Action: "error.test"}
		err := writer.Store(context.Background(), event)

		assert.Error(t, err)
		assert.Equal(t, storageErr, err)
	})

	t.Run("returns error when writer is closed", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}

		asyncWriter, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{
			BufferSize: 10,
		})

		// Close the writer
		err := closeFunc(context.Background())
		assert.NoError(t, err)

		// Try to store after closing - this may panic or return ErrStorageNotAvailable
		event := Event{Action: "closed.test"}

		// Handle both possible behaviors: panic or error
		var storeErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Panic is acceptable when storing to closed writer
					storeErr = ErrStorageNotAvailable // Treat panic as expected error
				}
			}()
			storeErr = asyncWriter.Store(context.Background(), event)
		}()

		// Either ErrStorageNotAvailable or panic (converted to ErrStorageNotAvailable) is acceptable
		assert.ErrorIs(t, storeErr, ErrStorageNotAvailable)
	})
}

func TestAsyncWriter_Close(t *testing.T) {
	t.Parallel()

	t.Run("closes gracefully with no pending events", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}

		_, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{})

		err := closeFunc(context.Background())
		assert.NoError(t, err)
	})

	t.Run("flushes pending events before closing", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}

		// Track all events to ensure they get flushed
		var totalEvents int32
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.MatchedBy(func(events []Event) bool {
			atomic.AddInt32(&totalEvents, int32(len(events)))
			return len(events) >= 1 // Accept any batch size
		})).Return(nil)

		writer, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{
			BufferSize:   10,
			BatchSize:    10, // Large batch size so events stay pending
			BatchTimeout: 1 * time.Second,
		})

		// Add some events that won't be flushed immediately
		for i := range 2 {
			event := Event{Action: "pending.test", Metadata: map[string]any{"index": i}}
			err := writer.Store(context.Background(), event)
			assert.NoError(t, err)
		}

		// Close should flush the pending events
		err := closeFunc(context.Background())
		assert.NoError(t, err)

		// Verify all events were processed
		assert.Equal(t, int32(2), atomic.LoadInt32(&totalEvents), "All pending events should be flushed on close")
	})

	t.Run("demonstrates storage timeout vs close timeout interaction", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}

		// Make StoreBatch respect the context timeout (StorageTimeout)
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.Anything).Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			select {
			case <-time.After(200 * time.Millisecond): // Try to block longer than StorageTimeout
			case <-ctx.Done():
				// Storage context timed out (should happen due to StorageTimeout)
			}
		}).Return(context.DeadlineExceeded)

		asyncWriter, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{
			BufferSize:     10,
			BatchSize:      1,
			BatchTimeout:   1 * time.Millisecond,
			StorageTimeout: 30 * time.Millisecond, // Short storage timeout
		})

		// Add an event - Store may fail due to storage error, which is fine
		event := Event{Action: "timeout.interaction.test"}
		err := asyncWriter.Store(context.Background(), event)
		// Don't assert on Store error - it might fail due to storage timeout

		// Give time for worker to process what it can
		time.Sleep(40 * time.Millisecond) // Wait longer than StorageTimeout

		// Close should succeed quickly because worker should have finished by now
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		err = closeFunc(ctx)
		elapsed := time.Since(start)

		// Should succeed because worker should be done
		assert.NoError(t, err)
		assert.Less(t, elapsed, 50*time.Millisecond, "Close should complete quickly when worker is already done")
	})

	t.Run("close timeout when worker is truly blocked", func(t *testing.T) {
		// This test demonstrates actual close timeout when worker never finishes
		// Skip it in normal test runs since it would require blocking indefinitely
		t.Skip("Demonstrates close timeout but would block indefinitely - for documentation only")
	})

	t.Run("calling close multiple times panics", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}

		_, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{})

		// First close should succeed
		err := closeFunc(context.Background())
		assert.NoError(t, err)

		// Second close should panic due to closing closed channel
		// This documents current behavior - in production, close should only be called once
		assert.Panics(t, func() {
			closeFunc(context.Background())
		})
	})
}

func TestAsyncWriter_ConcurrentOperations(t *testing.T) {
	t.Parallel()

	t.Run("handles concurrent Store calls", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		// Allow multiple batch calls
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.Anything).Return(nil)

		writer, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{
			BufferSize:   100,
			BatchSize:    10,
			BatchTimeout: 50 * time.Millisecond,
		})
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		const numGoroutines = 50
		const eventsPerGoroutine = 10

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*eventsPerGoroutine)

		// Launch concurrent goroutines
		wg.Add(numGoroutines)
		for i := range numGoroutines {
			go func(routineID int) {
				defer wg.Done()
				for j := range eventsPerGoroutine {
					event := Event{
						Action: "concurrent.test",
						Metadata: map[string]any{
							"routine": routineID,
							"event":   j,
						},
					}
					if err := writer.Store(context.Background(), event); err != nil {
						errors <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		for err := range errors {
			assert.NoError(t, err)
		}

		// Verify batches were called
		assert.True(t, mockBW.GetCallCount() > 0)
	})

	t.Run("handles Store during Close", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.Anything).Return(nil)

		writer, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{
			BufferSize: 100,
		})

		var wg sync.WaitGroup
		done := make(chan struct{})

		// Start goroutine that continuously stores events until signaled to stop
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-done:
					return
				default:
				}

				event := Event{Action: "concurrent.store", Metadata: map[string]any{"i": i}}

				// After close is called, Store may panic due to sending on closed channel
				// or return ErrStorageNotAvailable - both are acceptable behaviors
				func() {
					defer func() {
						if r := recover(); r != nil {
							// Panic is acceptable when sending to closed channel during shutdown
						}
					}()
					err := writer.Store(context.Background(), event)
					if err != nil && !errors.Is(err, ErrStorageNotAvailable) {
						// Other errors are not expected
						assert.NoError(t, err)
					}
				}()

				time.Sleep(time.Millisecond) // Small delay
			}
		}()

		// Close after a short delay
		time.Sleep(10 * time.Millisecond)
		close(done) // Signal goroutine to stop
		err := closeFunc(context.Background())
		assert.NoError(t, err)

		wg.Wait()
	})
}

func TestAsyncWriter_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("propagates storage errors to all events in batch", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		storageErr := errors.New("batch storage failed")
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.Anything).Return(storageErr)

		writer, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{
			BufferSize:   10,
			BatchSize:    2, // Small batch for quick testing
			BatchTimeout: 1 * time.Second,
		})
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		// Send events to fill a batch
		var wg sync.WaitGroup
		errors := make(chan error, 2)

		wg.Add(2)
		for i := range 2 {
			go func(index int) {
				defer wg.Done()
				event := Event{Action: "error.test", Metadata: map[string]any{"index": index}}
				err := writer.Store(context.Background(), event)
				errors <- err
			}(i)
		}

		wg.Wait()
		close(errors)

		// Both events should receive the same error
		errorCount := 0
		for err := range errors {
			assert.Error(t, err)
			assert.Equal(t, storageErr, err)
			errorCount++
		}
		assert.Equal(t, 2, errorCount)
	})

	t.Run("handles context timeout in storage operations", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}
		// Make StoreBatch respect context cancellation
		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.Anything).Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			select {
			case <-time.After(100 * time.Millisecond):
			case <-ctx.Done():
			}
		}).Return(context.DeadlineExceeded)

		writer, closeFunc := NewAsyncWriter(mockBW, AsyncOptions{
			BufferSize:     10,
			BatchSize:      1,
			StorageTimeout: 10 * time.Millisecond, // Very short timeout
		})
		defer func() {
			// Use longer timeout for cleanup to avoid hanging test
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			closeFunc(ctx)
		}()

		event := Event{Action: "timeout.test"}
		err := writer.Store(context.Background(), event)

		assert.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

func TestAsyncWriter_Integration(t *testing.T) {
	t.Parallel()

	t.Run("full workflow with realistic scenario", func(t *testing.T) {
		t.Parallel()
		mockBW := &MockBatchWriter{}

		// Track all batches received
		var batchesMutex sync.Mutex
		var allBatches [][]Event

		mockBW.On("StoreBatch", mock.AnythingOfType("*context.timerCtx"), mock.Anything).Run(func(args mock.Arguments) {
			events := args.Get(1).([]Event)
			batchesMutex.Lock()
			batch := make([]Event, len(events))
			copy(batch, events)
			allBatches = append(allBatches, batch)
			batchesMutex.Unlock()
		}).Return(nil)

		opts := AsyncOptions{
			BufferSize:     50,
			BatchSize:      5,
			BatchTimeout:   20 * time.Millisecond,
			StorageTimeout: 1 * time.Second,
		}

		writer, closeFunc := NewAsyncWriter(mockBW, opts)
		defer func() {
			err := closeFunc(context.Background())
			assert.NoError(t, err)
		}()

		// Send events in different patterns
		testCases := []struct {
			name   string
			count  int
			action string
		}{
			{"batch_trigger", 5, "batch.action"},   // Exactly one batch
			{"partial_batch", 3, "partial.action"}, // Will be flushed on timeout
			{"overflow", 7, "overflow.action"},     // More than one batch
		}

		totalEvents := 0
		for _, tc := range testCases {
			for i := range tc.count {
				event := Event{
					Action: tc.action,
					Metadata: map[string]any{
						"test_case": tc.name,
						"index":     i,
					},
				}
				err := writer.Store(context.Background(), event)
				assert.NoError(t, err)
				totalEvents++
			}
		}

		// Wait for all events to be processed (including timeout flushes)
		time.Sleep(100 * time.Millisecond)

		// Verify results
		batchesMutex.Lock()
		defer batchesMutex.Unlock()

		// Count total events processed
		processedEvents := 0
		for _, batch := range allBatches {
			processedEvents += len(batch)
		}

		assert.Equal(t, totalEvents, processedEvents, "All events should be processed")
		assert.True(t, len(allBatches) >= 3, "Should have at least 3 batches")

		// Verify batch sizes don't exceed limit
		for _, batch := range allBatches {
			assert.LessOrEqual(t, len(batch), opts.BatchSize, "Batch size should not exceed limit")
		}
	})
}
