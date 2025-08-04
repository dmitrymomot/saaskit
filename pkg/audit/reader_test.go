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

// MockStorageCounter is a mock implementation of the StorageCounter interface
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

func TestReader_PaginationWithLargeOffset(t *testing.T) {
	t.Parallel()

	// BUSINESS LOGIC: Pagination must handle large offsets gracefully
	// without performance degradation or errors
	storage := new(MockStorage)
	reader := audit.NewReader(storage)

	criteria := audit.Criteria{
		Action: "user.login",
		Offset: 50000, // Large offset that might cause issues
		Limit:  10,
	}

	// Should handle large offset without errors
	storage.On("Query", mock.Anything, criteria).Return([]audit.Event{}, nil)

	events, err := reader.Find(context.Background(), criteria)
	require.NoError(t, err)
	assert.Empty(t, events)

	storage.AssertExpectations(t)
}

func TestReader_CursorPaginationConsistency(t *testing.T) {
	t.Parallel()

	// BUSINESS LOGIC: Cursor pagination must provide consistent results
	// even when new records are inserted during pagination
	storage := new(MockStorageCounter)
	reader := audit.NewReader(storage)

	// Use fixed time to avoid race conditions
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	allEvents := []audit.Event{
		{ID: "event-1", Action: "user.login", CreatedAt: baseTime.Add(-3 * time.Hour)},
		{ID: "event-2", Action: "user.login", CreatedAt: baseTime.Add(-2 * time.Hour)},
		{ID: "event-3", Action: "user.login", CreatedAt: baseTime.Add(-1 * time.Hour)},
		{ID: "event-4", Action: "user.login", CreatedAt: baseTime},
	}

	criteria := audit.Criteria{
		Action: "user.login",
		Limit:  10,
	}

	// First call - get all events
	storage.On("Query", mock.Anything, criteria).Return(allEvents, nil).Once()

	events, nextCursor, err := reader.FindWithCursor(context.Background(), criteria, "")
	require.NoError(t, err)
	assert.Equal(t, allEvents, events)
	assert.Equal(t, "", nextCursor) // No more pages

	// Second call with cursor - storage should handle cursor-based filtering
	criteriaWithCursor := criteria
	criteriaWithCursor.Cursor = "event-2"
	criteriaWithCursor.Offset = 0

	// Storage implementation should return events after the cursor
	eventsAfterCursor := []audit.Event{
		{ID: "event-3", Action: "user.login", CreatedAt: baseTime.Add(-1 * time.Hour)},
		{ID: "event-4", Action: "user.login", CreatedAt: baseTime},
	}
	storage.On("Query", mock.Anything, criteriaWithCursor).Return(eventsAfterCursor, nil).Once()

	events, nextCursor, err = reader.FindWithCursor(context.Background(), criteria, "event-2")
	require.NoError(t, err)
	assert.Equal(t, eventsAfterCursor, events)
	assert.Equal(t, "", nextCursor)

	storage.AssertExpectations(t)
}

func TestReader_OptimizedCountVsFallback(t *testing.T) {
	t.Parallel()

	// BUSINESS LOGIC: Count operations must work correctly whether storage
	// implements optimized counting or requires fallback to query+count

	t.Run("optimized count available", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorageCounter)
		reader := audit.NewReader(storage)

		criteria := audit.Criteria{
			Action:   "payment.processed",
			TenantID: "tenant-123",
		}

		// Should use optimized Count method
		storage.On("Count", mock.Anything, criteria).Return(int64(1337), nil)

		count, err := reader.Count(context.Background(), criteria)
		require.NoError(t, err)
		assert.Equal(t, int64(1337), count)

		// Verify Query was NOT called
		storage.AssertNotCalled(t, "Query", mock.Anything, mock.Anything)
		storage.AssertExpectations(t)
	})

	t.Run("fallback to query when count not available", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage) // Does not implement StorageCounter
		reader := audit.NewReader(storage)

		criteria := audit.Criteria{
			Action: "user.login",
		}

		// Should fall back to Query and count results
		events := make([]audit.Event, 5)
		for i := range 5 {
			events[i] = audit.Event{ID: string(rune(i))}
		}
		storage.On("Query", mock.Anything, criteria).Return(events, nil)

		count, err := reader.Count(context.Background(), criteria)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)

		storage.AssertExpectations(t)
	})
}

func TestReader_StorageErrorPropagation(t *testing.T) {
	t.Parallel()

	// BUSINESS LOGIC: Storage errors must be properly propagated to callers
	// for proper error handling in application code
	storage := new(MockStorage)
	reader := audit.NewReader(storage)

	criteria := audit.Criteria{
		Action: "critical.operation",
	}

	storageErr := errors.New("database connection failed")
	storage.On("Query", mock.Anything, criteria).Return([]audit.Event(nil), storageErr)

	events, err := reader.Find(context.Background(), criteria)
	require.Error(t, err)
	assert.Equal(t, storageErr, err, "Storage errors must be propagated exactly")
	assert.Nil(t, events)

	storage.AssertExpectations(t)
}

func TestReader_EmptyResultsHandling(t *testing.T) {
	t.Parallel()

	// BUSINESS LOGIC: Empty results should be handled consistently
	// across different query scenarios
	storage := new(MockStorage)
	reader := audit.NewReader(storage)

	testCases := []struct {
		name     string
		criteria audit.Criteria
	}{
		{
			name: "no matching action",
			criteria: audit.Criteria{
				Action: "nonexistent.action",
			},
		},
		{
			name: "no matching user",
			criteria: audit.Criteria{
				UserID: "nonexistent-user",
			},
		},
		{
			name: "no matching time range",
			criteria: audit.Criteria{
				StartTime: time.Now().Add(-2 * time.Hour),
				EndTime:   time.Now().Add(-1 * time.Hour),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage.On("Query", mock.Anything, tc.criteria).Return([]audit.Event{}, nil).Once()

			events, err := reader.Find(context.Background(), tc.criteria)
			require.NoError(t, err)
			assert.Empty(t, events, "Empty results should be handled consistently")
		})
	}

	storage.AssertExpectations(t)
}
