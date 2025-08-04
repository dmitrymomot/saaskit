package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/audit"
)

// NoOpStorage is a storage implementation that does nothing, used for benchmarks
type NoOpStorage struct{}

func (s *NoOpStorage) Store(ctx context.Context, events ...audit.Event) error {
	return nil
}

func (s *NoOpStorage) Query(ctx context.Context, criteria audit.Criteria) ([]audit.Event, error) {
	// Return synthetic results for benchmarking
	events := make([]audit.Event, 0, criteria.Limit)
	for i := 0; i < criteria.Limit && i < 100; i++ {
		events = append(events, audit.Event{
			ID:        "test-id",
			Action:    "test.action",
			CreatedAt: time.Now(),
		})
	}
	return events, nil
}

func BenchmarkLogger_Log(b *testing.B) {
	storage := &NoOpStorage{}
	logger := audit.NewLogger(storage)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Log(ctx, "benchmark.test",
			audit.WithMetadata("iteration", i),
		)
	}
}

func BenchmarkLogger_LogWithExtractors(b *testing.B) {
	storage := &NoOpStorage{}
	logger := audit.NewLogger(storage,
		audit.WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
			return "tenant-123", true
		}),
		audit.WithUserIDExtractor(func(ctx context.Context) (string, bool) {
			return "user-456", true
		}),
	)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Log(ctx, "benchmark.test",
			audit.WithMetadata("iteration", i),
		)
	}
}

func BenchmarkAsyncLogger_Log(b *testing.B) {
	storage := &NoOpStorage{}
	logger := audit.NewLogger(storage, audit.WithAsync(1000))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Log(ctx, "benchmark.test.async",
			audit.WithMetadata("iteration", i),
		)
	}

	// Note: In real benchmarks, you'd want to ensure all async operations complete
	// For benchmark purposes, we're measuring the overhead of queueing
}

func BenchmarkAsyncLogger_LogWithOptions(b *testing.B) {
	storage := &NoOpStorage{}
	logger := audit.NewLogger(storage,
		audit.WithAsync(1000),
		audit.WithAsyncOptions(audit.AsyncOptions{
			BatchSize:      50,
			BatchTimeout:   50 * time.Millisecond,
			StorageTimeout: 2 * time.Second,
		}),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Log(ctx, "benchmark.test.async.options",
			audit.WithMetadata("iteration", i),
		)
	}
}

func BenchmarkReader_Find(b *testing.B) {
	storage := &NoOpStorage{}
	reader := audit.NewReader(storage)
	ctx := context.Background()

	criteria := audit.Criteria{
		Action: "test.action",
		Limit:  100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reader.Find(ctx, criteria)
	}
}

func BenchmarkReader_FindWithCursor(b *testing.B) {
	storage := &NoOpStorage{}
	reader := audit.NewReader(storage)
	ctx := context.Background()

	criteria := audit.Criteria{
		Action: "test.action",
		Limit:  100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = reader.FindWithCursor(ctx, criteria, "cursor-123")
	}
}

func BenchmarkEvent_Validate(b *testing.B) {
	event := audit.Event{
		ID:        "test-id",
		Action:    "test.action",
		CreatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = event.Validate()
	}
}

func BenchmarkEvent_ValidateEmpty(b *testing.B) {
	event := audit.Event{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = event.Validate()
	}
}
