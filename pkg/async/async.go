package async

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Future represents the result of an asynchronous computation.
type Future[U any] struct {
	result U
	err    error
	once   sync.Once
	done   chan struct{}
}

// Await waits for the asynchronous function to complete and returns its result and error.
func (f *Future[U]) Await() (U, error) {
	<-f.done
	return f.result, f.err
}

// AwaitWithTimeout waits for the asynchronous function to complete with a timeout.
// Returns the result and error if the function completes before the timeout.
// If the timeout occurs before completion, returns a timeout error.
func (f *Future[U]) AwaitWithTimeout(timeout time.Duration) (U, error) {
	select {
	case <-f.done:
		return f.result, f.err
	case <-time.After(timeout):
		var zero U
		return zero, errors.New("future: timeout waiting for completion")
	}
}

// IsComplete checks if the asynchronous function is complete without blocking.
// Returns true if the function has completed, false otherwise.
func (f *Future[U]) IsComplete() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

// Async executes a function asynchronously and returns a Future.
// The function accepts a context.Context and a parameter of any type T, and returns (U, error).
func Async[T any, U any](ctx context.Context, param T, fn func(context.Context, T) (U, error)) *Future[U] {
	f := &Future[U]{done: make(chan struct{})}

	go func() {
		defer close(f.done)

		// Early exit prevents goroutine leak when context is pre-canceled
		select {
		case <-ctx.Done():
			var zero U
			f.err = ctx.Err()
			f.result = zero
			return
		default:
		}

		// Execute function with potential for early cancellation via context
		res, err := fn(ctx, param)

		// Use sync.Once to prevent race conditions on multiple goroutine completions
		f.once.Do(func() {
			f.result = res
			f.err = err
		})
	}()

	return f
}

// WaitAll waits for all futures to complete and returns a slice of their results and an error
// if any of the futures returned an error.
func WaitAll[U any](futures ...*Future[U]) ([]U, error) {
	results := make([]U, len(futures))

	// Wait for all futures to complete
	for i, future := range futures {
		result, err := future.Await()
		results[i] = result
		if err != nil {
			return results, err
		}
	}

	return results, nil
}

// WaitAny waits for any of the futures to complete and returns the index of the completed future,
// its result, and any error it might have returned.
func WaitAny[U any](futures ...*Future[U]) (int, U, error) {
	if len(futures) == 0 {
		var zero U
		return -1, zero, errors.New("future: no futures provided to WaitAny")
	}

	// Create a channel to signal completion
	done := make(chan struct {
		index  int
		result U
		err    error
	})

	// Start a goroutine for each future
	for i, future := range futures {
		go func(index int, f *Future[U]) {
			result, err := f.Await()
			select {
			case done <- struct {
				index  int
				result U
				err    error
			}{index, result, err}:
			default:
				// Another future already completed
			}
		}(i, future)
	}

	// Wait for the first future to complete
	res := <-done
	return res.index, res.result, res.err
}
