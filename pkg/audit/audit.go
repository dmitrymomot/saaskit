package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type logger struct {
	storage              Storage
	tenantIDExtractor    func(context.Context) (string, bool)
	userIDExtractor      func(context.Context) (string, bool)
	sessionIDExtractor   func(context.Context) (string, bool)
	requestIDExtractor   func(context.Context) (string, bool)
	ipExtractor          func(context.Context) (string, bool)
	userAgentExtractor   func(context.Context) (string, bool)
	asyncBufferSize      int
}

type Option func(*logger)

func WithTenantIDExtractor(fn func(context.Context) (string, bool)) Option {
	return func(l *logger) {
		l.tenantIDExtractor = fn
	}
}

func WithUserIDExtractor(fn func(context.Context) (string, bool)) Option {
	return func(l *logger) {
		l.userIDExtractor = fn
	}
}

func WithSessionIDExtractor(fn func(context.Context) (string, bool)) Option {
	return func(l *logger) {
		l.sessionIDExtractor = fn
	}
}

func WithRequestIDExtractor(fn func(context.Context) (string, bool)) Option {
	return func(l *logger) {
		l.requestIDExtractor = fn
	}
}

func WithIPExtractor(fn func(context.Context) (string, bool)) Option {
	return func(l *logger) {
		l.ipExtractor = fn
	}
}

func WithUserAgentExtractor(fn func(context.Context) (string, bool)) Option {
	return func(l *logger) {
		l.userAgentExtractor = fn
	}
}

func WithAsync(bufferSize int) Option {
	return func(l *logger) {
		l.asyncBufferSize = bufferSize
	}
}

// NewLogger creates a new audit logger
func NewLogger(storage Storage, opts ...Option) Logger {
	if storage == nil {
		panic("audit: storage cannot be nil")
	}

	l := &logger{
		storage: storage,
	}

	for _, opt := range opts {
		opt(l)
	}

	if l.asyncBufferSize > 0 {
		l.storage = newAsyncStorage(l.storage, l.asyncBufferSize)
	}

	// Verify storage connectivity at initialization to fail fast if backend is unavailable
	// This prevents runtime surprises when audit logging is first attempted
	healthEvent := Event{
		ID:        uuid.New().String(),
		CreatedAt: time.Now(),
		Action:    "audit.health_check",
		TenantID:  "system",
		UserID:    "system",
		Result:    ResultSuccess,
	}

	if err := storage.Store(context.Background(), healthEvent); err != nil {
		panic(fmt.Sprintf("audit: storage health check failed: %v", err))
	}

	return l
}

// WithResource sets the resource type and ID
func WithResource(resource, id string) EventOption {
	return func(e *Event) {
		e.Resource = resource
		e.ResourceID = id
	}
}

// WithMetadata adds metadata to the event
func WithMetadata(key string, value any) EventOption {
	return func(e *Event) {
		if e.Metadata == nil {
			e.Metadata = make(map[string]any)
		}
		e.Metadata[key] = value
	}
}

// WithResult sets the event result
func WithResult(result Result) EventOption {
	return func(e *Event) {
		e.Result = result
	}
}

// Log records a successful action
func (l *logger) Log(ctx context.Context, action string, opts ...EventOption) error {
	event := l.eventFromContext(ctx)
	event.ID = uuid.New().String()
	event.CreatedAt = time.Now()
	event.Action = action
	event.Result = ResultSuccess

	for _, opt := range opts {
		opt(&event)
	}


	return l.storage.Store(ctx, event)
}

// LogError records a failed action
func (l *logger) LogError(ctx context.Context, action string, err error, opts ...EventOption) error {
	event := l.eventFromContext(ctx)
	event.ID = uuid.New().String()
	event.Action = action
	event.Result = ResultError
	event.Error = err.Error()
	event.CreatedAt = time.Now()

	for _, opt := range opts {
		opt(&event)
	}


	return l.storage.Store(ctx, event)
}

// eventFromContext extracts event data from context
func (l *logger) eventFromContext(ctx context.Context) Event {
	event := Event{}

	if l.tenantIDExtractor != nil {
		if tenantID, ok := l.tenantIDExtractor(ctx); ok {
			event.TenantID = tenantID
		}
	}

	if l.userIDExtractor != nil {
		if userID, ok := l.userIDExtractor(ctx); ok {
			event.UserID = userID
		}
	}

	if l.sessionIDExtractor != nil {
		if sessionID, ok := l.sessionIDExtractor(ctx); ok {
			event.SessionID = sessionID
		}
	}

	if l.requestIDExtractor != nil {
		if requestID, ok := l.requestIDExtractor(ctx); ok {
			event.RequestID = requestID
		}
	}

	if l.ipExtractor != nil {
		if ip, ok := l.ipExtractor(ctx); ok {
			event.IP = ip
		}
	}

	if l.userAgentExtractor != nil {
		if userAgent, ok := l.userAgentExtractor(ctx); ok {
			event.UserAgent = userAgent
		}
	}

	return event
}

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
	// For now, implement a basic cursor pagination using the ID field
	// Storage implementations can provide more sophisticated cursor support
	modifiedCriteria := criteria
	if cursor != "" {
		// For basic implementation, use cursor as an ID offset
		// Storage implementations should handle cursor in their own way
		modifiedCriteria.Offset = 0 // Reset offset when using cursor
	}
	
	events, err := r.storage.Query(ctx, modifiedCriteria)
	if err != nil {
		return nil, "", err
	}
	
	// Filter events after cursor ID if provided
	if cursor != "" {
		filtered := make([]Event, 0, len(events))
		found := false
		for _, e := range events {
			if found {
				filtered = append(filtered, e)
			}
			if e.ID == cursor {
				found = true
			}
		}
		events = filtered
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
