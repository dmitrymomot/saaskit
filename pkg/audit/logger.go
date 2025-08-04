package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// contextExtractor extracts string values from context.
// It returns (value, found) where found indicates if extraction succeeded.
type contextExtractor func(context.Context) (string, bool)

type logger struct {
	storage            Storage
	tenantIDExtractor  contextExtractor
	userIDExtractor    contextExtractor
	sessionIDExtractor contextExtractor
	requestIDExtractor contextExtractor
	ipExtractor        contextExtractor
	userAgentExtractor contextExtractor
	asyncBufferSize    int
	asyncOptions       AsyncOptions
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
		l.storage = newAsyncStorage(l.storage, l.asyncBufferSize, l.asyncOptions)
	}

	return l
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

	if err := event.Validate(); err != nil {
		return err
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

	if err := event.Validate(); err != nil {
		return err
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
