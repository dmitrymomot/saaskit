package audit

import "context"

type reader struct {
	storage Storage
}

// NewReader creates a new audit reader
func NewReader(storage Storage) Reader {
	if storage == nil {
		panic("audit: storage cannot be nil")
	}
	return &reader{storage: storage}
}

// Find retrieves audit events based on the criteria
func (r *reader) Find(ctx context.Context, criteria Criteria) ([]Event, error) {
	return r.storage.Query(ctx, criteria)
}

// FindWithCursor retrieves audit events based on the criteria with cursor-based pagination
func (r *reader) FindWithCursor(ctx context.Context, criteria Criteria, cursor string) ([]Event, string, error) {
	// Pass cursor to storage via Criteria
	modifiedCriteria := criteria
	modifiedCriteria.Cursor = cursor
	if cursor != "" {
		modifiedCriteria.Offset = 0 // Reset offset when using cursor
	}

	events, err := r.storage.Query(ctx, modifiedCriteria)
	if err != nil {
		return nil, "", err
	}

	// Generate next cursor from last event ID
	nextCursor := ""
	if len(events) > 0 && len(events) == criteria.Limit {
		nextCursor = events[len(events)-1].ID
	}

	return events, nextCursor, nil
}

// Count returns the count of audit events matching the criteria.
// If the storage implements StorageCounter, it uses the optimized Count method.
// Otherwise, it falls back to loading all records and counting them in memory.
func (r *reader) Count(ctx context.Context, criteria Criteria) (int64, error) {
	if counter, ok := r.storage.(StorageCounter); ok {
		return counter.Count(ctx, criteria)
	}

	// Fallback: load all events and count them
	events, err := r.storage.Query(ctx, criteria)
	if err != nil {
		return 0, err
	}
	return int64(len(events)), nil
}
