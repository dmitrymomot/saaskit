package async_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/async"
)

// TestAsyncFunctionality tests the basic functionality of the Async helper.
func TestAsyncFunctionality(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Function that takes an int parameter and returns a string
	futureString := async.Async(ctx, 42, func(ctx context.Context, num int) (string, error) {
		// Simulate work
		time.Sleep(100 * time.Millisecond)
		return fmt.Sprintf("Number: %d", num), nil
	})

	// Function that takes a string parameter and returns a bool
	futureBool := async.Async(ctx, "test", func(ctx context.Context, s string) (bool, error) {
		// Simulate work
		time.Sleep(50 * time.Millisecond)
		return len(s) > 0, nil
	})

	// Function that takes a custom struct parameter and returns an int
	type MyStruct struct {
		A int
		B int
	}
	futureInt := async.Async(ctx, MyStruct{A: 10, B: 32}, func(ctx context.Context, data MyStruct) (int, error) {
		// Simulate work
		time.Sleep(70 * time.Millisecond)
		return data.A + data.B, nil
	})

	// Await the results
	resultString, errString := futureString.Await()
	resultBool, errBool := futureBool.Await()
	resultInt, errInt := futureInt.Await()

	// Check results
	if errString != nil || resultString != "Number: 42" {
		t.Errorf("Expected 'Number: 42', got '%s', error: %v", resultString, errString)
	}

	if errBool != nil || resultBool != true {
		t.Errorf("Expected true, got %v, error: %v", resultBool, errBool)
	}

	if errInt != nil || resultInt != 42 {
		t.Errorf("Expected 42, got %d, error: %v", resultInt, errInt)
	}
}

// TestAsyncContextCancellation tests that the Async helper handles context cancellation properly.
func TestAsyncContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	future := async.Async(ctx, 42, func(ctx context.Context, num int) (string, error) {
		// Simulate a task that takes longer than the context timeout
		select {
		case <-time.After(100 * time.Millisecond):
			return fmt.Sprintf("Number: %d", num), nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	})

	result, err := future.Await()

	// Check that the context cancellation is handled
	if err == nil || err != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty result due to cancellation, got: '%s'", result)
	}
}

// TestAsyncErrorPropagation tests that errors from the asynchronous function are propagated correctly.
func TestAsyncErrorPropagation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	expectedErr := errors.New("an error occurred in the async function")

	future := async.Async(ctx, 42, func(ctx context.Context, num int) (int, error) {
		// Simulate work and return an error
		time.Sleep(50 * time.Millisecond)
		return 0, expectedErr
	})

	result, err := future.Await()

	// Check that the error is propagated
	if err == nil || err != expectedErr {
		t.Errorf("Expected error '%v', got: %v", expectedErr, err)
	}

	if result != 0 {
		t.Errorf("Expected result 0 due to error, got: %d", result)
	}
}

// TestAsyncConcurrency tests that multiple Async calls execute concurrently.
func TestAsyncConcurrency(t *testing.T) {
	t.Parallel()
	// Note: This test has timing assertions that might be sensitive to system load when run in parallel
	ctx := context.Background()
	startTime := time.Now()

	var mu sync.Mutex
	order := []string{}

	future1 := async.Async(ctx, 1, func(ctx context.Context, num int) (int, error) {
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		order = append(order, "first")
		mu.Unlock()
		return num, nil
	})

	future2 := async.Async(ctx, 2, func(ctx context.Context, num int) (int, error) {
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		order = append(order, "second")
		mu.Unlock()
		return num, nil
	})

	future3 := async.Async(ctx, 3, func(ctx context.Context, num int) (int, error) {
		time.Sleep(70 * time.Millisecond)
		mu.Lock()
		order = append(order, "third")
		mu.Unlock()
		return num, nil
	})

	// Await the results
	_, _ = future1.Await()
	_, _ = future2.Await()
	_, _ = future3.Await()

	duration := time.Since(startTime)

	// The total duration should be slightly longer than the longest sleep (100ms)
	if duration > 150*time.Millisecond {
		t.Errorf("Expected duration around 100ms, got %v", duration)
	}

	// Check the order of completion
	expectedOrder := []string{"second", "third", "first"}
	for i, v := range expectedOrder {
		if order[i] != v {
			t.Errorf("Expected order %v, got %v", expectedOrder, order)
			break
		}
	}
}

// TestAsyncWithCustomStruct tests using custom structures as parameters and return types.
func TestAsyncWithCustomStruct(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	type Input struct {
		X int
		Y int
	}

	type Output struct {
		Sum int
	}

	future := async.Async(ctx, Input{X: 10, Y: 15}, func(ctx context.Context, in Input) (Output, error) {
		// Simulate work
		time.Sleep(50 * time.Millisecond)
		return Output{Sum: in.X + in.Y}, nil
	})

	result, err := future.Await()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Sum != 25 {
		t.Errorf("Expected sum 25, got %d", result.Sum)
	}
}

func TestAsyncConcurrentIncrement(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var wg sync.WaitGroup
	var mu sync.Mutex
	counter := 0

	increment := func(_ context.Context, delta int) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		counter += delta
		return counter, nil
	}

	futures := make([]*async.Future[int], 0)
	for range 1000 {
		wg.Add(1)
		future := async.Async(ctx, 1, func(ctx context.Context, delta int) (int, error) {
			defer wg.Done()
			return increment(ctx, delta)
		})
		futures = append(futures, future)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Check the final counter value
	if counter != 1000 {
		t.Errorf("Expected counter to be 1000, got %d", counter)
	}

	// Optionally, check the results from futures
	for _, future := range futures {
		result, err := future.Await()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result < 1 || result > 1000 {
			t.Errorf("Result out of expected range: %d", result)
		}
	}
}

// TestIsComplete tests the IsComplete method of Future.
func TestIsComplete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create a future that will take some time to complete
	future := async.Async(ctx, 100, func(ctx context.Context, ms int) (bool, error) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return true, nil
	})

	// Initially, the future should not be complete
	if future.IsComplete() {
		t.Error("Expected future to not be complete immediately")
	}

	// After waiting for the future to complete, IsComplete should return true
	_, err := future.Await()
	if err != nil {
		t.Errorf("Unexpected error waiting for future: %v", err)
	}

	if !future.IsComplete() {
		t.Error("Expected future to be complete after Await")
	}
}

// TestAwaitWithTimeout tests the AwaitWithTimeout method of Future.
func TestAwaitWithTimeout(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Test case 1: Future completes before timeout
	fastFuture := async.Async(ctx, 50, func(ctx context.Context, ms int) (string, error) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return "success", nil
	})

	result, err := fastFuture.AwaitWithTimeout(100 * time.Millisecond)
	if err != nil {
		t.Errorf("Expected no error for fast future, got: %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got: %s", result)
	}

	// Test case 2: Future does not complete before timeout
	slowFuture := async.Async(ctx, 200, func(ctx context.Context, ms int) (string, error) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return "too late", nil
	})

	result, err = slowFuture.AwaitWithTimeout(100 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error for slow future")
	}
	if result != "" {
		t.Errorf("Expected empty result for timeout, got: %s", result)
	}
}

// TestWaitAll tests the WaitAll function.
func TestWaitAll(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create multiple futures
	future1 := async.Async(ctx, 50, func(ctx context.Context, ms int) (int, error) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return 1, nil
	})

	future2 := async.Async(ctx, 100, func(ctx context.Context, ms int) (int, error) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return 2, nil
	})

	future3 := async.Async(ctx, 150, func(ctx context.Context, ms int) (int, error) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return 3, nil
	})

	// Wait for all futures to complete
	startTime := time.Now()
	results, err := async.WaitAll(future1, future2, future3)
	duration := time.Since(startTime)

	// Verify results
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	expectedResults := []int{1, 2, 3}
	for i, result := range results {
		if result != expectedResults[i] {
			t.Errorf("Expected result[%d] to be %d, got %d", i, expectedResults[i], result)
		}
	}

	// Verify that WaitAll waited for the slowest future
	if duration < 150*time.Millisecond {
		t.Errorf("Expected duration to be at least 150ms, got %v", duration)
	}
}

// TestWaitAny tests the WaitAny function.
func TestWaitAny(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create multiple futures with different completion times
	future1 := async.Async(ctx, 150, func(ctx context.Context, ms int) (string, error) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return "slow", nil
	})

	future2 := async.Async(ctx, 50, func(ctx context.Context, ms int) (string, error) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return "fast", nil
	})

	future3 := async.Async(ctx, 100, func(ctx context.Context, ms int) (string, error) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return "medium", nil
	})

	// Wait for any future to complete
	startTime := time.Now()
	index, result, err := async.WaitAny(future1, future2, future3)
	duration := time.Since(startTime)

	// Verify results
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// The fastest future should complete first
	if index != 1 || result != "fast" {
		t.Errorf("Expected index=1 and result='fast', got index=%d and result='%s'", index, result)
	}

	// Verify that WaitAny returned as soon as the fastest future completed
	if duration < 50*time.Millisecond || duration >= 100*time.Millisecond {
		t.Errorf("Expected duration to be around 50ms, got %v", duration)
	}

	// Test with empty futures list - explicitly specify the type parameter
	_, _, err = async.WaitAny[string]()
	if err == nil {
		t.Error("Expected error when calling WaitAny with no futures")
	}
}
