package audit

import (
	"context"
	"sync"
	"time"
)

// AsyncOptions configures the batching and buffering behavior for optimal throughput.
// These settings control the tradeoff between memory usage, latency, and storage efficiency.
type AsyncOptions struct {
	BufferSize     int           // Max events queued in memory before blocking/falling back to sync writes
	BatchSize      int           // Target events per batch - optimize based on storage bulk insert performance
	BatchTimeout   time.Duration // Max time to wait for partial batches - controls worst-case latency
	StorageTimeout time.Duration // Per-batch storage timeout - prevents hanging on slow/failed storage
}

type AsyncWriter struct {
	batchWriter batchWriter
	eventChan   chan eventBatch
	done        chan struct{}
	wg          sync.WaitGroup
	options     AsyncOptions
}

type eventBatch struct {
	ctx    context.Context
	events []Event
	result chan error
}

// batchWriter provides efficient bulk storage for audit events.
// Implementations should optimize for batch inserts (e.g., SQL bulk insert, batch APIs).
// Must be idempotent and atomic - either all events succeed or all fail.
type batchWriter interface {
	StoreBatch(ctx context.Context, events []Event) error
}

// NewAsyncWriter creates an async writer that batches events for improved throughput.
// Uses a background goroutine to collect events into batches, reducing storage I/O.
// Only accepts BatchWriter since single-event writers would defeat the batching purpose.
func NewAsyncWriter(bw batchWriter, opts AsyncOptions) (*AsyncWriter, func(context.Context) error) {
	if bw == nil {
		panic("audit: batch writer cannot be nil")
	}

	// Apply defaults optimized for typical SaaS audit workloads
	if opts.BufferSize == 0 {
		opts.BufferSize = 1000 // Balance memory usage with burst capacity
	}
	if opts.BatchSize == 0 {
		opts.BatchSize = 100 // Optimize for database bulk inserts
	}
	if opts.BatchTimeout == 0 {
		opts.BatchTimeout = 100 * time.Millisecond // Ensure low latency for small volumes
	}
	if opts.StorageTimeout == 0 {
		opts.StorageTimeout = 5 * time.Second // Prevent hanging on slow storage
	}

	aw := &AsyncWriter{
		batchWriter: bw,
		eventChan:   make(chan eventBatch, opts.BufferSize),
		done:        make(chan struct{}),
		options:     opts,
	}

	aw.wg.Add(1)
	go aw.worker()

	closeFunc := func(ctx context.Context) error {
		return aw.Close(ctx)
	}

	return aw, closeFunc
}

// Store implements Writer interface
func (aw *AsyncWriter) Store(ctx context.Context, event Event) error {
	result := make(chan error, 1)

	select {
	case aw.eventChan <- eventBatch{ctx: ctx, events: []Event{event}, result: result}:
		// Event queued successfully, wait for batch processing result
		select {
		case err := <-result:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-aw.done:
		return ErrStorageNotAvailable
	default:
		// Buffer full - bypass async processing to prevent event loss
		// This maintains audit completeness at the cost of synchronous I/O
		return aw.batchWriter.StoreBatch(ctx, []Event{event})
	}
}

func (aw *AsyncWriter) worker() {
	defer aw.wg.Done()

	batchEvents := make([]Event, 0, aw.options.BatchSize)
	batchTimer := time.NewTicker(aw.options.BatchTimeout)
	defer batchTimer.Stop()

	pendingResults := make([]chan error, 0, aw.options.BatchSize)

	// flushBatch writes accumulated events to storage and notifies all waiting callers.
	// Uses background context to prevent client timeouts from cascading to storage operations.
	flushBatch := func() {
		if len(batchEvents) == 0 {
			return
		}

		// Isolate storage operations from client request contexts to prevent timeout cascades
		ctx, cancel := context.WithTimeout(context.Background(), aw.options.StorageTimeout)
		defer cancel()

		err := aw.batchWriter.StoreBatch(ctx, batchEvents)

		// Notify all requests in this batch of the storage result
		for _, resultChan := range pendingResults {
			select {
			case resultChan <- err:
			default:
				// Channel closed due to client timeout - safe to ignore
			}
		}

		// Reset batch collectors for next iteration
		clear(batchEvents)
		clear(pendingResults)
		batchEvents = batchEvents[:0]
		pendingResults = pendingResults[:0]
	}

	for {
		select {
		case batch := <-aw.eventChan:
			batchEvents = append(batchEvents, batch.events...)
			pendingResults = append(pendingResults, batch.result)

			// Flush when batch reaches target size for optimal database performance
			if len(batchEvents) >= aw.options.BatchSize {
				flushBatch()
			}

		case <-batchTimer.C:
			// Periodic flush ensures events don't sit in memory too long
			flushBatch()

		case <-aw.done:
			// Graceful shutdown: drain remaining events to prevent data loss
			close(aw.eventChan)
			for batch := range aw.eventChan {
				batchEvents = append(batchEvents, batch.events...)
				pendingResults = append(pendingResults, batch.result)
			}
			flushBatch()
			return
		}
	}
}

// Close gracefully shuts down the async writer, ensuring no events are lost.
// The context controls shutdown timeout - if exceeded, some events may remain unflushed.
// Always call this during application shutdown to prevent audit event loss.
func (aw *AsyncWriter) Close(ctx context.Context) error {
	// Signal shutdown
	close(aw.done)

	// Race between graceful shutdown completion and context timeout
	doneChan := make(chan struct{})
	go func() {
		aw.wg.Wait()
		close(doneChan)
	}()

	select {
	case <-doneChan:
		// All events successfully flushed
		return nil
	case <-ctx.Done():
		// Shutdown timeout exceeded - some events may be lost
		return ctx.Err()
	}
}
