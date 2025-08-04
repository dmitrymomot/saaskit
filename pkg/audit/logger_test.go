package audit_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/audit"
)

// MockStorage is a mock implementation of the Storage interface
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Store(ctx context.Context, events ...audit.Event) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *MockStorage) Query(ctx context.Context, criteria audit.Criteria) ([]audit.Event, error) {
	args := m.Called(ctx, criteria)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]audit.Event), args.Error(1)
}

func TestNewLogger(t *testing.T) {
	t.Parallel()

	t.Run("nil storage panics", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			audit.NewLogger(nil)
		})
	})

	t.Run("successful initialization", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage)

		logger := audit.NewLogger(storage)
		assert.NotNil(t, logger)
		storage.AssertExpectations(t)
	})

	t.Run("with extractors", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage)

		logger := audit.NewLogger(storage,
			audit.WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
				return "tenant-123", true
			}),
			audit.WithUserIDExtractor(func(ctx context.Context) (string, bool) {
				return "user-456", true
			}),
			audit.WithSessionIDExtractor(func(ctx context.Context) (string, bool) {
				return "session-789", true
			}),
		)
		assert.NotNil(t, logger)
		storage.AssertExpectations(t)
	})
}

func TestLogger_Log(t *testing.T) {
	t.Parallel()

	t.Run("basic logging", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage)
		storage.On("Store", mock.Anything, mock.Anything).Return(nil).Once()

		logger := audit.NewLogger(storage)
		err := logger.Log(context.Background(), "user.login")
		require.NoError(t, err)

		storage.AssertExpectations(t)
	})

	t.Run("with context extractors", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage)

		storage.On("Store", mock.Anything, mock.MatchedBy(func(events []audit.Event) bool {
			if len(events) != 1 {
				return false
			}
			e := events[0]
			return e.Action == "user.update" &&
				e.TenantID == "tenant-123" &&
				e.UserID == "user-456" &&
				e.SessionID == "session-789"
		})).Return(nil).Once()

		logger := audit.NewLogger(storage,
			audit.WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
				return "tenant-123", true
			}),
			audit.WithUserIDExtractor(func(ctx context.Context) (string, bool) {
				return "user-456", true
			}),
			audit.WithSessionIDExtractor(func(ctx context.Context) (string, bool) {
				return "session-789", true
			}),
		)

		err := logger.Log(context.Background(), "user.update")
		require.NoError(t, err)
		storage.AssertExpectations(t)
	})

	t.Run("with options", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage)

		storage.On("Store", mock.Anything, mock.MatchedBy(func(events []audit.Event) bool {
			if len(events) != 1 {
				return false
			}
			e := events[0]
			return e.Action == "payment.processed" &&
				e.Resource == "payment" &&
				e.ResourceID == "pay-123" &&
				e.Result == audit.ResultSuccess &&
				e.Metadata != nil &&
				e.Metadata["amount"] == 99.99 &&
				e.Metadata["currency"] == "USD"
		})).Return(nil).Once()

		logger := audit.NewLogger(storage)

		err := logger.Log(context.Background(), "payment.processed",
			audit.WithResource("payment", "pay-123"),
			audit.WithMetadata("amount", 99.99),
			audit.WithMetadata("currency", "USD"),
			audit.WithResult(audit.ResultSuccess),
		)
		require.NoError(t, err)
		storage.AssertExpectations(t)
	})

	t.Run("storage error", func(t *testing.T) {
		t.Parallel()
		storage := new(MockStorage)
		storage.On("Store", mock.Anything, mock.Anything).Return(errors.New("storage error")).Once()

		logger := audit.NewLogger(storage)
		err := logger.Log(context.Background(), "test.action")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "storage error")
		storage.AssertExpectations(t)
	})
}

func TestLogger_LogError(t *testing.T) {
	t.Parallel()

	storage := new(MockStorage)

	storage.On("Store", mock.Anything, mock.MatchedBy(func(events []audit.Event) bool {
		if len(events) != 1 {
			return false
		}
		e := events[0]
		return e.Action == "payment.failed" &&
			e.Result == audit.ResultError &&
			e.Error == "insufficient funds" &&
			e.Resource == "payment" &&
			e.ResourceID == "pay-456"
	})).Return(nil).Once()

	logger := audit.NewLogger(storage)

	err := logger.LogError(
		context.Background(),
		"payment.failed",
		errors.New("insufficient funds"),
		audit.WithResource("payment", "pay-456"),
	)
	require.NoError(t, err)
	storage.AssertExpectations(t)
}

func TestLogger_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	storage := new(MockStorage)
	storage.On("Store", mock.Anything, mock.Anything).Return(nil)

	logger := audit.NewLogger(storage)

	const goroutines = 10
	const logsPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			for j := range logsPerGoroutine {
				err := logger.Log(context.Background(), "concurrent.test",
					audit.WithMetadata("goroutine", id),
					audit.WithMetadata("iteration", j),
				)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify expected number of calls (goroutines * logsPerGoroutine)
	expectedCalls := goroutines * logsPerGoroutine
	storage.AssertNumberOfCalls(t, "Store", expectedCalls)
}

// Reader tests moved to reader_test.go to avoid duplication
// and maintain focused test organization
