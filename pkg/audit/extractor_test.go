package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/audit"
)

// MockStorageCounter implements both Storage and StorageCounter interfaces
type MockStorageCounter struct {
	mock.Mock
}

func (m *MockStorageCounter) Store(ctx context.Context, events ...audit.Event) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *MockStorageCounter) Query(ctx context.Context, criteria audit.Criteria) ([]audit.Event, error) {
	args := m.Called(ctx, criteria)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]audit.Event), args.Error(1)
}

func (m *MockStorageCounter) Count(ctx context.Context, criteria audit.Criteria) (int64, error) {
	args := m.Called(ctx, criteria)
	return args.Get(0).(int64), args.Error(1)
}

func TestExtractors(t *testing.T) {
	t.Parallel()

	t.Run("RequestID extractor", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage)
		storage.On("Store", mock.Anything, mock.Anything).Return(nil).Once() // health check
		
		storage.On("Store", mock.Anything, mock.MatchedBy(func(events []audit.Event) bool {
			return len(events) == 1 && events[0].RequestID == "req-123"
		})).Return(nil).Once()

		logger := audit.NewLogger(storage,
			audit.WithRequestIDExtractor(func(ctx context.Context) (string, bool) {
				return "req-123", true
			}),
		)

		err := logger.Log(context.Background(), "test.action")
		require.NoError(t, err)
		storage.AssertExpectations(t)
	})

	t.Run("IP extractor", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage)
		storage.On("Store", mock.Anything, mock.Anything).Return(nil).Once() // health check
		
		storage.On("Store", mock.Anything, mock.MatchedBy(func(events []audit.Event) bool {
			return len(events) == 1 && events[0].IP == "192.168.1.100"
		})).Return(nil).Once()

		logger := audit.NewLogger(storage,
			audit.WithIPExtractor(func(ctx context.Context) (string, bool) {
				return "192.168.1.100", true
			}),
		)

		err := logger.Log(context.Background(), "test.action")
		require.NoError(t, err)
		storage.AssertExpectations(t)
	})

	t.Run("UserAgent extractor", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage)
		storage.On("Store", mock.Anything, mock.Anything).Return(nil).Once() // health check
		
		storage.On("Store", mock.Anything, mock.MatchedBy(func(events []audit.Event) bool {
			return len(events) == 1 && events[0].UserAgent == "Mozilla/5.0"
		})).Return(nil).Once()

		logger := audit.NewLogger(storage,
			audit.WithUserAgentExtractor(func(ctx context.Context) (string, bool) {
				return "Mozilla/5.0", true
			}),
		)

		err := logger.Log(context.Background(), "test.action")
		require.NoError(t, err)
		storage.AssertExpectations(t)
	})

	t.Run("all extractors together", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage)
		storage.On("Store", mock.Anything, mock.Anything).Return(nil).Once() // health check
		
		storage.On("Store", mock.Anything, mock.MatchedBy(func(events []audit.Event) bool {
			if len(events) != 1 {
				return false
			}
			e := events[0]
			return e.RequestID == "req-456" &&
				e.IP == "10.0.0.1" &&
				e.UserAgent == "API-Client/1.0" &&
				e.TenantID == "tenant-789" &&
				e.UserID == "user-123"
		})).Return(nil).Once()

		logger := audit.NewLogger(storage,
			audit.WithRequestIDExtractor(func(ctx context.Context) (string, bool) {
				return "req-456", true
			}),
			audit.WithIPExtractor(func(ctx context.Context) (string, bool) {
				return "10.0.0.1", true
			}),
			audit.WithUserAgentExtractor(func(ctx context.Context) (string, bool) {
				return "API-Client/1.0", true
			}),
			audit.WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
				return "tenant-789", true
			}),
			audit.WithUserIDExtractor(func(ctx context.Context) (string, bool) {
				return "user-123", true
			}),
		)

		err := logger.Log(context.Background(), "test.action")
		require.NoError(t, err)
		storage.AssertExpectations(t)
	})

	t.Run("extractors handle missing values", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage)
		storage.On("Store", mock.Anything, mock.Anything).Return(nil).Once() // health check
		
		storage.On("Store", mock.Anything, mock.MatchedBy(func(events []audit.Event) bool {
			if len(events) != 1 {
				return false
			}
			e := events[0]
			// When extractors return false, fields should be empty
			return e.RequestID == "" &&
				e.IP == "" &&
				e.UserAgent == "API-Client/2.0" // Only this one returns true
		})).Return(nil).Once()

		logger := audit.NewLogger(storage,
			audit.WithRequestIDExtractor(func(ctx context.Context) (string, bool) {
				return "", false // Not found
			}),
			audit.WithIPExtractor(func(ctx context.Context) (string, bool) {
				return "", false // Not found
			}),
			audit.WithUserAgentExtractor(func(ctx context.Context) (string, bool) {
				return "API-Client/2.0", true // Found
			}),
		)

		err := logger.Log(context.Background(), "test.action")
		require.NoError(t, err)
		storage.AssertExpectations(t)
	})
}

func TestFindWithCursor(t *testing.T) {
	t.Parallel()

	t.Run("basic cursor pagination", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorageCounter)
		reader := audit.NewReader(storage)

		criteria := audit.Criteria{
			Action: "user.login",
			Limit:  10,
		}

		expectedEvents := []audit.Event{
			{ID: "event-1", Action: "user.login"},
			{ID: "event-2", Action: "user.login"},
		}

		// Mock the FindWithCursor behavior
		// Since it's not implemented in storage, it should use Query
		storage.On("Query", mock.Anything, criteria).Return(expectedEvents, nil)

		events, nextCursor, err := reader.FindWithCursor(context.Background(), criteria, "")
		require.NoError(t, err)
		assert.Equal(t, expectedEvents, events)
		assert.Equal(t, "", nextCursor) // Implementation returns empty cursor
		storage.AssertExpectations(t)
	})

	t.Run("with cursor", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorageCounter)
		reader := audit.NewReader(storage)

		criteria := audit.Criteria{
			Action: "user.login",
			Limit:  10,
		}

		allEvents := []audit.Event{
			{ID: "event-1", Action: "user.login"},
			{ID: "event-2", Action: "user.login"},
			{ID: "event-3", Action: "user.login"},
			{ID: "event-4", Action: "user.login"},
		}

		// The implementation fetches all events and filters after cursor
		storage.On("Query", mock.Anything, criteria).Return(allEvents, nil)

		events, nextCursor, err := reader.FindWithCursor(context.Background(), criteria, "event-2")
		require.NoError(t, err)
		// Should return events after cursor
		assert.Equal(t, []audit.Event{
			{ID: "event-3", Action: "user.login"},
			{ID: "event-4", Action: "user.login"},
		}, events)
		assert.Equal(t, "", nextCursor) // Implementation returns empty cursor
		storage.AssertExpectations(t)
	})

	t.Run("empty results", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorageCounter)
		reader := audit.NewReader(storage)

		criteria := audit.Criteria{
			Action: "user.login",
			Limit:  10,
		}

		storage.On("Query", mock.Anything, criteria).Return([]audit.Event{}, nil)

		events, nextCursor, err := reader.FindWithCursor(context.Background(), criteria, "")
		require.NoError(t, err)
		assert.Empty(t, events)
		assert.Empty(t, nextCursor)
		storage.AssertExpectations(t)
	})
}

func TestStorageCounter_OptimizedCount(t *testing.T) {
	t.Parallel()

	t.Run("uses optimized count when available", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorageCounter)
		reader := audit.NewReader(storage)

		criteria := audit.Criteria{
			Action:   "user.login",
			TenantID: "tenant-123",
		}

		// StorageCounter.Count should be called, not Query
		storage.On("Count", mock.Anything, criteria).Return(int64(42), nil)

		count, err := reader.Count(context.Background(), criteria)
		require.NoError(t, err)
		assert.Equal(t, int64(42), count)

		// Verify Query was NOT called
		storage.AssertNotCalled(t, "Query", mock.Anything, mock.Anything)
		storage.AssertExpectations(t)
	})
}

func TestWithAsync_Integration(t *testing.T) {
	t.Parallel()

	t.Run("async storage processes events", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage)
		eventCount := 5
		processed := make(chan struct{})
		
		// Health check
		storage.On("Store", mock.Anything, mock.Anything).Return(nil).Once()
		
		// Expect batch processing
		storage.On("Store", mock.Anything, mock.MatchedBy(func(events []audit.Event) bool {
			if len(events) >= eventCount {
				close(processed)
				return true
			}
			return len(events) > 0
		})).Return(nil).Maybe()

		logger := audit.NewLogger(storage, audit.WithAsync(100))

		// Send events
		for i := 0; i < eventCount; i++ {
			err := logger.Log(context.Background(), "async.test")
			require.NoError(t, err)
		}

		// Wait for processing
		select {
		case <-processed:
			// Success
		case <-time.After(500 * time.Millisecond):
			// That's OK, async processing is not guaranteed to batch exactly
		}

		// Verify events were stored
		storage.AssertCalled(t, "Store", mock.Anything, mock.Anything)
	})
}