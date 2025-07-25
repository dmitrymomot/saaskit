package queue_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/queue"
)

// Example_oneTimeTask demonstrates enqueueing and processing a one-time task
func Example_oneTimeTask() {
	// Create memory storage
	storage := queue.NewMemoryStorage()
	defer storage.Close()

	// Create enqueuer
	enqueuer, err := queue.NewEnqueuer(storage)
	if err != nil {
		panic(err)
	}

	// Define task payload
	type EmailPayload struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	payload := EmailPayload{
		To:      "user@example.com",
		Subject: "Welcome!",
		Body:    "Thanks for signing up!",
	}

	// Enqueue task
	err = enqueuer.Enqueue(context.Background(), payload)
	if err != nil {
		panic(err)
	}

	fmt.Println("Task enqueued")

	// Create worker with no logger to avoid output noise
	worker, err := queue.NewWorker(storage,
		queue.WithMaxConcurrentTasks(1),
		queue.WithPullInterval(10*time.Millisecond),
		queue.WithWorkerLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	if err != nil {
		panic(err)
	}

	// Register handler - handler name is derived from the payload type
	handler := queue.NewTaskHandler(func(ctx context.Context, email EmailPayload) error {
		fmt.Printf("Sending email to %s: %s\n", email.To, email.Subject)
		return nil
	})
	worker.RegisterHandler(handler)

	// Start worker
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Start(ctx)
	}()

	// Wait a bit for the task to be processed
	time.Sleep(50 * time.Millisecond)
	cancel()
	wg.Wait()

	// Output:
	// Task enqueued
	// Sending email to user@example.com: Welcome!
}

// Example_scheduledTask demonstrates scheduling and processing a scheduled task
func Example_scheduledTask() {
	// Create memory storage
	storage := queue.NewMemoryStorage()
	defer storage.Close()

	// Create enqueuer
	enqueuer, err := queue.NewEnqueuer(storage)
	if err != nil {
		panic(err)
	}

	// Define task payload
	type ReportPayload struct {
		ReportType string `json:"report_type"`
		Period     string `json:"period"`
	}

	payload := ReportPayload{
		ReportType: "daily-summary",
		Period:     "2024-01-01",
	}

	// Schedule task for 50ms from now
	err = enqueuer.Enqueue(context.Background(), payload,
		queue.WithScheduledAt(time.Now().Add(50*time.Millisecond)))
	if err != nil {
		panic(err)
	}

	fmt.Println("Task scheduled")

	// Create worker
	worker, err := queue.NewWorker(storage,
		queue.WithMaxConcurrentTasks(1),
		queue.WithPullInterval(10*time.Millisecond),
		queue.WithWorkerLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	if err != nil {
		panic(err)
	}

	// Register handler - handler name is derived from the payload type
	handler := queue.NewTaskHandler(func(ctx context.Context, report ReportPayload) error {
		fmt.Printf("Generating %s report for %s\n", report.ReportType, report.Period)
		return nil
	})
	worker.RegisterHandler(handler)

	// Start worker
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Start(ctx)
	}()

	// Wait for the scheduled time and processing
	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()

	// Output:
	// Task scheduled
	// Generating daily-summary report for 2024-01-01
}
