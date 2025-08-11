package ratelimiter_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/ratelimiter"
)

// BenchmarkTokenBucket_Allow benchmarks single token consumption
func BenchmarkTokenBucket_Allow(b *testing.B) {
	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	config := ratelimiter.Config{
		Capacity:       10000,
		RefillRate:     1000,
		RefillInterval: time.Second,
	}

	tb, err := ratelimiter.NewTokenBucket(store, config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		key := fmt.Sprintf("bench-key-%d", b.N)
		for pb.Next() {
			_, _ = tb.Allow(ctx, key)
		}
	})
}

// BenchmarkTokenBucket_AllowN benchmarks multiple token consumption
func BenchmarkTokenBucket_AllowN(b *testing.B) {
	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	config := ratelimiter.Config{
		Capacity:       10000,
		RefillRate:     1000,
		RefillInterval: time.Second,
	}

	tb, err := ratelimiter.NewTokenBucket(store, config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	tokenCounts := []int{1, 5, 10, 50}

	for _, tokens := range tokenCounts {
		b.Run(fmt.Sprintf("tokens=%d", tokens), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				key := fmt.Sprintf("bench-key-%d-%d", tokens, b.N)
				for pb.Next() {
					_, _ = tb.AllowN(ctx, key, tokens)
				}
			})
		})
	}
}

// BenchmarkTokenBucket_Status benchmarks status checks
func BenchmarkTokenBucket_Status(b *testing.B) {
	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	config := ratelimiter.Config{
		Capacity:       1000,
		RefillRate:     100,
		RefillInterval: time.Second,
	}

	tb, err := ratelimiter.NewTokenBucket(store, config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	key := "bench-status-key"

	// Pre-populate the bucket
	_, _ = tb.Allow(ctx, key)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = tb.Status(ctx, key)
		}
	})
}

// BenchmarkMemoryStore_ConsumeTokens benchmarks the store directly
func BenchmarkMemoryStore_ConsumeTokens(b *testing.B) {
	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	config := ratelimiter.Config{
		Capacity:       1000,
		RefillRate:     100,
		RefillInterval: time.Second,
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		key := fmt.Sprintf("bench-store-%d", b.N)
		for pb.Next() {
			_, _, _ = store.ConsumeTokens(ctx, key, 1, config)
		}
	})
}

// BenchmarkMemoryStore_ConcurrentAccess benchmarks concurrent operations
func BenchmarkMemoryStore_ConcurrentAccess(b *testing.B) {
	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	config := ratelimiter.Config{
		Capacity:       1000,
		RefillRate:     100,
		RefillInterval: time.Second,
	}

	ctx := context.Background()
	keys := make([]string, 100)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := keys[i%len(keys)]
			_, _, _ = store.ConsumeTokens(ctx, key, 1, config)
			i++
		}
	})
}

// BenchmarkTokenBucket_HighContention benchmarks under high contention
func BenchmarkTokenBucket_HighContention(b *testing.B) {
	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	config := ratelimiter.Config{
		Capacity:       100,
		RefillRate:     10,
		RefillInterval: time.Millisecond,
	}

	tb, err := ratelimiter.NewTokenBucket(store, config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	// Use same key to create contention
	key := "contention-key"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = tb.Allow(ctx, key)
		}
	})
}

// BenchmarkTokenBucket_MixedOperations benchmarks mixed read/write operations
func BenchmarkTokenBucket_MixedOperations(b *testing.B) {
	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	config := ratelimiter.Config{
		Capacity:       1000,
		RefillRate:     100,
		RefillInterval: time.Second,
	}

	tb, err := ratelimiter.NewTokenBucket(store, config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	key := "mixed-ops-key"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				// 10% status checks
				_, _ = tb.Status(ctx, key)
			} else if i%20 == 0 {
				// 5% consume multiple tokens
				_, _ = tb.AllowN(ctx, key, 5)
			} else {
				// 85% single token consumption
				_, _ = tb.Allow(ctx, key)
			}
			i++
		}
	})
}

// BenchmarkResult_Allowed benchmarks the Result.Allowed() method
func BenchmarkResult_Allowed(b *testing.B) {
	result := &ratelimiter.Result{
		Limit:     100,
		Remaining: 50,
		ResetAt:   time.Now().Add(time.Minute),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.Allowed()
	}
}

// BenchmarkResult_RetryAfter benchmarks the Result.RetryAfter() method
func BenchmarkResult_RetryAfter(b *testing.B) {
	result := &ratelimiter.Result{
		Limit:     100,
		Remaining: -1,
		ResetAt:   time.Now().Add(time.Minute),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.RetryAfter()
	}
}
