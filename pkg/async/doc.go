// Package async provides simple, generic helpers for running computations asynchronously and
// waiting for their completion.
//
// The package is centred around the generic type Future that represents the eventual result of an
// asynchronous operation.  A Future can be obtained by calling Async, which starts the supplied
// function in its own goroutine and immediately returns a *Future instance.  The caller can then
// wait for completion with Await, block with a timeout using AwaitWithTimeout, or poll the state
// with IsComplete.
//
// In addition to operating on a single Future, the helpers WaitAll and WaitAny make it easy to
// coordinate multiple concurrent tasks – either collecting every result or returning the first one
// to finish.
//
// All helpers are context-aware: if the provided context is cancelled before the computation
// finishes, the underlying goroutine aborts early and the Future is completed with the context
// error.
//
// # Usage
//
//	import (
// 	    "context"
//	    "time"
//	    "github.com/dmitrymomot/saaskit/pkg/async"
//	)
//
//	func main() {
//	    ctx := context.Background()
//	    future := async.Async(ctx, 42, func(_ context.Context, v int) (string, error) {
//	        time.Sleep(100 * time.Millisecond)
//	        return fmt.Sprintf("value is %d", v), nil
//	    })
//
//	    // do other work …
//	    res, err := future.Await()
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    fmt.Println(res)
//	}
//
// # Error Handling
//
// The package does not introduce custom error types; functions return the error produced by the user
// callback or, in the case of AwaitWithTimeout, a timeout error.
//
// # Performance Considerations
//
// Futures are lightweight wrappers around goroutines and channels.  The overhead is minimal but you
// should avoid spawning an excessive number of goroutines if the workload could be better handled
// by a worker pool or other means of limiting concurrency.
//
// See the individual function-level comments for additional details and behaviour guarantees.
package async
