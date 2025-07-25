// Package queue provides a repository-agnostic task queue with first-class support for
// immediate, delayed, and periodic execution.
//
// The package is organised around three main components:
//
//   - Enqueuer   — adds one-time tasks to the queue
//   - Scheduler  — converts cron-like Schedule definitions into tasks at runtime
//   - Worker     — claims pending tasks and dispatches them to a user supplied Handler
//
// Components interact only through a set of small repository interfaces, keeping the
// business logic decoupled from persistence. This design allows you to back the queue
// with any storage engine (PostgreSQL, Redis, MongoDB, etc.) simply by implementing
// the required interfaces.
//
// # Architecture
//
//  1. The EnqueuerRepository, SchedulerRepository, and WorkerRepository interfaces
//     encapsulate all persistence concerns.
//  2. Enqueuer, Scheduler, and Worker are independent and can be deployed in
//     separate processes or services.
//  3. A Task is immutable once persisted; retry attempts are tracked via
//     RetryCount and MaxRetries fields.
//  4. Queue name and Priority allow routing of high-value work to dedicated workers.
//
// # Usage
//
// Basic one-time task:
//
//	import (
//	    "context"
//	    "time"
//
//	    "github.com/dmitrymomot/saaskit/pkg/queue"
//	)
//
//	type SendEmailPayload struct {
//	    UserID int64
//	}
//
//	func example(repo queue.EnqueuerRepository) error {
//		    e, err := queue.NewEnqueuer(repo)
//		    if err != nil {
//		        return err
//		    }
//
//		    // Execute within the next minute
//		    return e.Enqueue(context.Background(),
//		        SendEmailPayload{UserID: 42},
//		        queue.WithDelay(time.Minute),
//		    )
//	}
//
// Periodic job:
//
//	s, _ := queue.NewScheduler(repo, queue.WithCheckInterval(30*time.Second))
//
// _ = s.AddTask(
//
//	"cleanup_sessions",
//	queue.DailyAt(2, 0), // runs every day at 02:00
//	queue.WithTaskPriority(queue.PriorityLow),
//
// )
//
// go s.Start(context.Background())
//
// # Error Handling
//
// Package-level sentinel errors (e.g. ErrInvalidPriority, ErrNoHandlers) signal
// violations of business invariants and can be checked with errors.Is.
//
// # Examples
//
// Additional runnable examples live in the package's example_test.go files.
package queue
