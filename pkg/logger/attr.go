package logger

import (
	"log/slog"
	"strconv"
)

// Group creates a slog group attribute from the provided attributes.
func Group(name string, attrs ...slog.Attr) slog.Attr {
	return slog.Attr{Key: name, Value: slog.GroupValue(attrs...)}
}

// Errors groups multiple non-nil errors under the key "errors".
// If all errors are nil, it returns an empty Attr.
func Errors(errs ...error) slog.Attr {
	as := make([]slog.Attr, 0, len(errs))
	for i, err := range errs {
		if err != nil {
			as = append(as, slog.Any(strconv.Itoa(i), err))
		}
	}
	if len(as) == 0 {
		return slog.Attr{}
	}
	return slog.Attr{Key: "errors", Value: slog.GroupValue(as...)}
}

// Error creates an attribute for a single error under the key "error".
// If err is nil, it returns an empty Attr.
func Error(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.Any("error", err)
}

// UserID records the user identifier under the key "user_id".
// If id is nil, it returns an empty Attr.
func UserID(id any) slog.Attr {
	if id == nil {
		return slog.Attr{}
	}
	return slog.Any("user_id", id)
}

// WorkspaceID records the workspace identifier under the key "workspace_id".
// If id is nil, it returns an empty Attr.
func WorkspaceID(id any) slog.Attr {
	if id == nil {
		return slog.Attr{}
	}
	return slog.Any("workspace_id", id)
}

// Role records a role name under the key "role".
// If role is nil, it returns an empty Attr.
func Role(role any) slog.Attr {
	if role == nil {
		return slog.Attr{}
	}
	return slog.Any("role", role)
}

// RequestID records the request identifier under the key "request_id".
// If id is nil, it returns an empty Attr.
func RequestID(id any) slog.Attr {
	if id == nil {
		return slog.Attr{}
	}
	return slog.Any("request_id", id)
}

// EventType records the event type under the key "event_type".
func EventType(eventType string) slog.Attr {
	return slog.String("event_type", eventType)
}

// MessageID records the message identifier under the key "message_id".
// If id is nil, it returns an empty Attr.
func MessageID(id any) slog.Attr {
	if id == nil {
		return slog.Attr{}
	}
	return slog.Any("message_id", id)
}

// RetryCount records the retry count under the key "retry_count".
func RetryCount(count int) slog.Attr {
	return slog.Int("retry_count", count)
}

// Duration records a duration under the key "duration".
func Duration(d any) slog.Attr {
	return slog.Any("duration", d)
}

// Component records the component name under the key "component".
func Component(name string) slog.Attr {
	return slog.String("component", name)
}

// Event records the event name under the key "event".
func Event(name string) slog.Attr {
	return slog.String("event", name)
}

// Handler records the handler name under the key "handler".
func Handler(name string) slog.Attr {
	return slog.String("handler", name)
}
