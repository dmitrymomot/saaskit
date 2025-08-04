package audit

import (
	"context"
	"sync"
	"time"
)

// AsyncOptions configures async writer behavior
type AsyncOptions struct {
	BufferSize     int           // Size of the event buffer
	BatchSize      int           // Number of events to batch before flushing
	BatchTimeout   time.Duration // Duration to wait before flushing a partial batch
	StorageTimeout time.Duration // Timeout for storing events to the underlying storage
}

type AsyncWriter struct {
	batchWriter BatchWriter
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

// BatchWriter stores multiple audit events efficiently
type BatchWriter interface {
	StoreBatch(ctx context.Context, events []Event) error
}

// NewAsyncWriter creates an async writer that batches events
// Only accepts BatchWriter since its purpose is to optimize batch operations
func NewAsyncWriter(bw BatchWriter, opts AsyncOptions) (*AsyncWriter, func(context.Context) error) {
	if bw == nil {
		panic("audit: batch writer cannot be nil")
	}

	// Apply defaults if not specified
	if opts.BufferSize == 0 {
		opts.BufferSize = 1000
	}
	if opts.BatchSize == 0 {
		opts.BatchSize = 100
	}
	if opts.BatchTimeout == 0 {
		opts.BatchTimeout = 100 * time.Millisecond
	}
	if opts.StorageTimeout == 0 {
		opts.StorageTimeout = 5 * time.Second
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
		// Buffer is full, fall back to synchronous write
		return aw.batchWriter.StoreBatch(ctx, []Event{event})
	}
}

func (aw *AsyncWriter) worker() {
	defer aw.wg.Done()

	batchEvents := make([]Event, 0, aw.options.BatchSize)
	batchTimer := time.NewTicker(aw.options.BatchTimeout)
	defer batchTimer.Stop()

	pendingResults := make([]chan error, 0, aw.options.BatchSize)

	flushBatch := func() {
		if len(batchEvents) == 0 {
			return
		}

		// Use background context for storage to avoid cascading timeouts
		ctx, cancel := context.WithTimeout(context.Background(), aw.options.StorageTimeout)
		defer cancel()

		err := aw.batchWriter.StoreBatch(ctx, batchEvents)

		// Send result to all pending channels
		for _, resultChan := range pendingResults {
			select {
			case resultChan <- err:
			default:
				// Channel might be closed if request timed out
			}
		}

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

			// Flush if batch is getting large
			if len(batchEvents) >= aw.options.BatchSize {
				flushBatch()
			}

		case <-batchTimer.C:
			flushBatch()

		case <-aw.done:
			// Drain remaining events
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

// Close gracefully shuts down the async writer
// Respects context cancellation for timeouts
func (aw *AsyncWriter) Close(ctx context.Context) error {
	// Signal shutdown
	close(aw.done)

	// Wait for worker to finish or context to cancel
	doneChan := make(chan struct{})
	go func() {
		aw.wg.Wait()
		close(doneChan)
	}()

	select {
	case <-doneChan:
		// Graceful shutdown completed
		return nil
	case <-ctx.Done():
		// Context cancelled, shutdown may be incomplete
		return ctx.Err()
	}
}
