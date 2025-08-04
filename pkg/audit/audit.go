package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SensitiveDataHasher defines the interface for hashing sensitive data
type SensitiveDataHasher interface {
	Hash(data string) string
}

type logger struct {
	storage            Storage
	tenantIDExtractor  func(context.Context) (string, bool)
	userIDExtractor    func(context.Context) (string, bool)
	sessionIDExtractor func(context.Context) (string, bool)
	sensitiveHasher    SensitiveDataHasher
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

func WithSensitiveDataHasher(h SensitiveDataHasher) Option {
	return func(l *logger) {
		l.sensitiveHasher = h
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
func WithMetadata(key string, value interface{}) EventOption {
	return func(e *Event) {
		if e.Metadata == nil {
			e.Metadata = make(map[string]interface{})
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

	// Apply one-way hashing to sensitive identifiers for privacy compliance
	// This allows correlation analysis while preventing PII exposure in audit logs
	if l.sensitiveHasher != nil {
		if event.UserID != "" {
			event.UserID = l.sensitiveHasher.Hash(event.UserID)
		}
		if event.SessionID != "" {
			event.SessionID = l.sensitiveHasher.Hash(event.SessionID)
		}
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

	// Apply one-way hashing to sensitive identifiers for privacy compliance
	// This allows correlation analysis while preventing PII exposure in audit logs
	if l.sensitiveHasher != nil {
		if event.UserID != "" {
			event.UserID = l.sensitiveHasher.Hash(event.UserID)
		}
		if event.SessionID != "" {
			event.SessionID = l.sensitiveHasher.Hash(event.SessionID)
		}
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

// Count returns the count of audit events matching the criteria.
// WARNING: This implementation loads all matching records into memory for counting.
// Production storage implementations should override with optimized COUNT queries.
func (r *reader) Count(ctx context.Context, criteria Criteria) (int64, error) {
	events, err := r.storage.Query(ctx, criteria)
	if err != nil {
		return 0, err
	}
	return int64(len(events)), nil
}
