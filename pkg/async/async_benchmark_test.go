package async_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/async"
)

// BenchmarkAsyncOverhead measures async overhead with 1000 concurrent tasks.
func BenchmarkAsyncOverhead(b *testing.B) {
	ctx := context.Background()

	for b.Loop() {
		var wg sync.WaitGroup
		numTasks := 1000

		workFunc := func(_ context.Context, param int) (int, error) {
			time.Sleep(1 * time.Millisecond)
			return param * 2, nil
		}

		futures := make([]*async.Future[int], numTasks)
		for i := range numTasks {
			wg.Add(1)
			futures[i] = async.Async(ctx, i, func(ctx context.Context, param int) (int, error) {
				defer wg.Done()
				return workFunc(ctx, param)
			})
		}

		wg.Wait()
		for _, future := range futures {
			_, err := future.Await()
			if err != nil {
				b.Errorf("Unexpected error: %v", err)
			}
		}
	}
}

// BenchmarkAsyncWithoutSleep measures async overhead with CPU-bound tasks.
func BenchmarkAsyncWithoutSleep(b *testing.B) {
	ctx := context.Background()

	for b.Loop() {
		var wg sync.WaitGroup
		numTasks := 1000

		workFunc := func(_ context.Context, param int) (int, error) {
			return param * 2, nil
		}

		futures := make([]*async.Future[int], numTasks)
		for i := range numTasks {
			wg.Add(1)
			futures[i] = async.Async(ctx, i, func(ctx context.Context, param int) (int, error) {
				defer wg.Done()
				return workFunc(ctx, param)
			})
		}

		wg.Wait()
		for _, future := range futures {
			_, err := future.Await()
			if err != nil {
				b.Errorf("Unexpected error: %v", err)
			}
		}
	}
}

// BenchmarkAsyncWithContention measures performance under mutex contention.
func BenchmarkAsyncWithContention(b *testing.B) {
	ctx := context.Background()

	for b.Loop() {
		var wg sync.WaitGroup
		numTasks := 1000
		var mu sync.Mutex
		counter := 0

		workFunc := func(_ context.Context, param int) (int, error) {
			mu.Lock()
			counter += param
			mu.Unlock()
			return counter, nil
		}

		futures := make([]*async.Future[int], numTasks)
		for i := range numTasks {
			wg.Add(1)
			futures[i] = async.Async(ctx, i, func(ctx context.Context, param int) (int, error) {
				defer wg.Done()
				return workFunc(ctx, param)
			})
		}

		wg.Wait()
		for _, future := range futures {
			_, err := future.Await()
			if err != nil {
				b.Errorf("Unexpected error: %v", err)
			}
		}
	}
}
