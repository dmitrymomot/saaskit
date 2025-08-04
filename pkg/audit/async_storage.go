package audit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	// defaultBatchSize is the default number of events to batch before flushing
	defaultBatchSize = 100
	// defaultBatchTimeout is the duration to wait before flushing a partial batch
	defaultBatchTimeout = 100 * time.Millisecond
	// defaultStorageTimeout is the timeout for storing events to the underlying storage
	defaultStorageTimeout = 5 * time.Second
)

type asyncStorage struct {
	underlying Storage
	eventChan  chan eventBatch
	done       chan struct{}
	wg         sync.WaitGroup
	options    AsyncOptions
}

type eventBatch struct {
	ctx    context.Context
	events []Event
	result chan error
}

func newAsyncStorage(storage Storage, bufferSize int, opts AsyncOptions) Storage {
	as := &asyncStorage{
		underlying: storage,
		eventChan:  make(chan eventBatch, bufferSize),
		done:       make(chan struct{}),
		options:    opts,
	}

	// Apply defaults if not specified
	if as.options.BatchSize == 0 {
		as.options.BatchSize = defaultBatchSize
	}
	if as.options.BatchTimeout == 0 {
		as.options.BatchTimeout = defaultBatchTimeout
	}
	if as.options.StorageTimeout == 0 {
		as.options.StorageTimeout = defaultStorageTimeout
	}

	as.wg.Add(1)
	go as.worker()

	return as
}

func (as *asyncStorage) Store(ctx context.Context, events ...Event) error {
	result := make(chan error, 1)

	select {
	case as.eventChan <- eventBatch{ctx: ctx, events: events, result: result}:
		select {
		case err := <-result:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-as.done:
		return ErrStorageNotAvailable
	default:
		// Buffer is full, fall back to synchronous write
		return as.underlying.Store(ctx, events...)
	}
}

func (as *asyncStorage) Query(ctx context.Context, criteria Criteria) ([]Event, error) {
	// Queries are always synchronous
	return as.underlying.Query(ctx, criteria)
}

func (as *asyncStorage) worker() {
	defer as.wg.Done()

	batchEvents := make([]Event, 0, as.options.BatchSize)
	batchTimer := time.NewTicker(as.options.BatchTimeout)
	defer batchTimer.Stop()

	pendingResults := make([]chan error, 0, as.options.BatchSize)

	flushBatch := func() {
		if len(batchEvents) == 0 {
			return
		}

		// Use background context for storage to avoid cascading timeouts
		ctx, cancel := context.WithTimeout(context.Background(), as.options.StorageTimeout)
		defer cancel()

		err := as.underlying.Store(ctx, batchEvents...)

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
		case batch := <-as.eventChan:
			batchEvents = append(batchEvents, batch.events...)
			pendingResults = append(pendingResults, batch.result)

			// Flush if batch is getting large
			if len(batchEvents) >= as.options.BatchSize {
				flushBatch()
			}

		case <-batchTimer.C:
			flushBatch()

		case <-as.done:
			// Drain remaining events
			close(as.eventChan)
			for batch := range as.eventChan {
				batchEvents = append(batchEvents, batch.events...)
				pendingResults = append(pendingResults, batch.result)
			}
			flushBatch()
			return
		}
	}
}

func (as *asyncStorage) Close() error {
	close(as.done)
	as.wg.Wait()

	if closer, ok := as.underlying.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// Ensure asyncStorage implements StorageCounter if underlying storage does
func (as *asyncStorage) Count(ctx context.Context, criteria Criteria) (int64, error) {
	if counter, ok := as.underlying.(StorageCounter); ok {
		return counter.Count(ctx, criteria)
	}
	return 0, fmt.Errorf("underlying storage does not implement StorageCounter")
}
