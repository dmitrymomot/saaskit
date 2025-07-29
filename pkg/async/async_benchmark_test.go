package async_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/async"
)

// BenchmarkAsyncOverhead measures the overhead of the Async helper with a large number of tasks.
func BenchmarkAsyncOverhead(b *testing.B) {
	ctx := context.Background()

	for b.Loop() {
		var wg sync.WaitGroup
		numTasks := 1000

		// Function to be executed asynchronously
		workFunc := func(_ context.Context, param int) (int, error) {
			// Simulate some work
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

		// Wait for all tasks to complete
		wg.Wait()

		// Optionally, retrieve results (could be omitted to focus on overhead)
		for _, future := range futures {
			_, err := future.Await()
			if err != nil {
				b.Errorf("Unexpected error: %v", err)
			}
		}
	}
}

// BenchmarkAsyncWithoutSleep measures the overhead of the Async helper without any sleep in the tasks.
func BenchmarkAsyncWithoutSleep(b *testing.B) {
	ctx := context.Background()

	for b.Loop() {
		var wg sync.WaitGroup
		numTasks := 1000

		// Function to be executed asynchronously
		workFunc := func(_ context.Context, param int) (int, error) {
			// Minimal work
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

		// Wait for all tasks to complete
		wg.Wait()

		// Optionally, retrieve results
		for _, future := range futures {
			_, err := future.Await()
			if err != nil {
				b.Errorf("Unexpected error: %v", err)
			}
		}
	}
}

// BenchmarkAsyncWithContention measures performance when tasks contend for a shared resource.
func BenchmarkAsyncWithContention(b *testing.B) {
	ctx := context.Background()

	for b.Loop() {
		var wg sync.WaitGroup
		numTasks := 1000
		var mu sync.Mutex
		counter := 0

		// Function to be executed asynchronously
		workFunc := func(_ context.Context, param int) (int, error) {
			// Simulate some work with shared resource
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

		// Wait for all tasks to complete
		wg.Wait()

		// Optionally, retrieve results
		for _, future := range futures {
			_, err := future.Await()
			if err != nil {
				b.Errorf("Unexpected error: %v", err)
			}
		}
	}
}
