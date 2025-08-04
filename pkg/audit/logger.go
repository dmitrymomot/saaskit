package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Logger struct {
	writer             writer
	tenantIDExtractor  contextExtractor
	userIDExtractor    contextExtractor
	sessionIDExtractor contextExtractor
	requestIDExtractor contextExtractor
	ipExtractor        contextExtractor
	userAgentExtractor contextExtractor
}

// contextExtractor extracts string values from context.
// It returns (value, found) where found indicates if extraction succeeded.
type contextExtractor func(context.Context) (string, bool)

// writer stores single audit events
type writer interface {
	Store(ctx context.Context, event Event) error
}

// NewLogger creates a new audit Logger
func NewLogger(w writer, opts ...Option) *Logger {
	if w == nil {
		panic("audit: writer cannot be nil")
	}

	l := &Logger{writer: w}
	for _, opt := range opts {
		opt(l)
	}

	return l
}

// Log records a successful action
func (l *Logger) Log(ctx context.Context, action string, opts ...EventOption) error {
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

	return l.writer.Store(ctx, event)
}

// LogError records a failed action
func (l *Logger) LogError(ctx context.Context, action string, err error, opts ...EventOption) error {
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

	return l.writer.Store(ctx, event)
}

// eventFromContext extracts event data from context
func (l *Logger) eventFromContext(ctx context.Context) Event {
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
