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

func TestAsyncFunctionality(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	futureString := async.Async(ctx, 42, func(ctx context.Context, num int) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return fmt.Sprintf("Number: %d", num), nil
	})

	futureBool := async.Async(ctx, "test", func(ctx context.Context, s string) (bool, error) {
		time.Sleep(50 * time.Millisecond)
		return len(s) > 0, nil
	})

	type MyStruct struct {
		A int
		B int
	}
	futureInt := async.Async(ctx, MyStruct{A: 10, B: 32}, func(ctx context.Context, data MyStruct) (int, error) {
		time.Sleep(70 * time.Millisecond)
		return data.A + data.B, nil
	})

	resultString, errString := futureString.Await()
	resultBool, errBool := futureBool.Await()
	resultInt, errInt := futureInt.Await()

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

func TestAsyncContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	future := async.Async(ctx, 42, func(ctx context.Context, num int) (string, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return fmt.Sprintf("Number: %d", num), nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	})

	result, err := future.Await()

	if err == nil || err != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty result due to cancellation, got: '%s'", result)
	}
}

func TestAsyncErrorPropagation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	expectedErr := errors.New("an error occurred in the async function")

	future := async.Async(ctx, 42, func(ctx context.Context, num int) (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 0, expectedErr
	})

	result, err := future.Await()

	if err == nil || err != expectedErr {
		t.Errorf("Expected error '%v', got: %v", expectedErr, err)
	}

	if result != 0 {
		t.Errorf("Expected result 0 due to error, got: %d", result)
	}
}

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

	_, _ = future1.Await()
	_, _ = future2.Await()
	_, _ = future3.Await()

	duration := time.Since(startTime)

	// Duration should be slightly longer than the longest sleep (100ms) since futures run concurrently
	if duration > 150*time.Millisecond || duration < 100*time.Millisecond {
		t.Errorf("Expected duration between 100-150ms, got %v", duration)
	}

	expectedOrder := []string{"second", "third", "first"}
	for i, v := range expectedOrder {
		if order[i] != v {
			t.Errorf("Expected order %v, got %v", expectedOrder, order)
			break
		}
	}
}

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

	wg.Wait()

	if counter != 1000 {
		t.Errorf("Expected counter to be 1000, got %d", counter)
	}

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

func TestIsComplete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	future := async.Async(ctx, 100, func(ctx context.Context, ms int) (bool, error) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return true, nil
	})

	if future.IsComplete() {
		t.Error("Expected future to not be complete immediately")
	}

	_, err := future.Await()
	if err != nil {
		t.Errorf("Unexpected error waiting for future: %v", err)
	}

	if !future.IsComplete() {
		t.Error("Expected future to be complete after Await")
	}
}

func TestAwaitWithTimeout(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

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

func TestWaitAll(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

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

	startTime := time.Now()
	results, err := async.WaitAll(future1, future2, future3)
	duration := time.Since(startTime)

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

	// WaitAll waits for the slowest future
	if duration < 150*time.Millisecond {
		t.Errorf("Expected duration to be at least 150ms, got %v", duration)
	}
}

func TestWaitAny(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

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

	startTime := time.Now()
	index, result, err := async.WaitAny(future1, future2, future3)
	duration := time.Since(startTime)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if index != 1 || result != "fast" {
		t.Errorf("Expected index=1 and result='fast', got index=%d and result='%s'", index, result)
	}

	// WaitAny returns as soon as the fastest future completes
	if duration < 50*time.Millisecond || duration >= 100*time.Millisecond {
		t.Errorf("Expected duration to be around 50ms, got %v", duration)
	}

	// Explicitly specify the type parameter for empty futures list
	_, _, err = async.WaitAny[string]()
	if err == nil {
		t.Error("Expected error when calling WaitAny with no futures")
	}
}
