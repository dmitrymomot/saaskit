package logger

import (
	"log/slog"
	"strconv"
)

// Attribute helpers use the empty Attr pattern for nil safety.
// This allows calls like log.Info("msg", logger.Error(err)) without explicit nil checks,
// following the principle of making zero values useful.

func Group(name string, attrs ...slog.Attr) slog.Attr {
	return slog.Attr{Key: name, Value: slog.GroupValue(attrs...)}
}

// Errors groups multiple non-nil errors under the key "errors".
// Uses index-based keys to preserve error order. Returns empty Attr for all nil errors.
func Errors(errs ...error) slog.Attr {
	// Count non-nil errors first to allocate exact size
	count := 0
	for _, err := range errs {
		if err != nil {
			count++
		}
	}
	if count == 0 {
		return slog.Attr{}
	}

	as := make([]slog.Attr, 0, count)
	for i, err := range errs {
		if err != nil {
			as = append(as, slog.Any(strconv.Itoa(i), err))
		}
	}
	return slog.Attr{Key: "errors", Value: slog.GroupValue(as...)}
}

// Error creates an attribute for a single error under the key "error".
// Returns empty Attr for nil errors, enabling safe usage without nil checks.
func Error(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.Any("error", err)
}

func UserID(id any) slog.Attr {
	if id == nil {
		return slog.Attr{}
	}
	return slog.Any("user_id", id)
}

func WorkspaceID(id any) slog.Attr {
	if id == nil {
		return slog.Attr{}
	}
	return slog.Any("workspace_id", id)
}

func Role(role any) slog.Attr {
	if role == nil {
		return slog.Attr{}
	}
	return slog.Any("role", role)
}

func RequestID(id any) slog.Attr {
	if id == nil {
		return slog.Attr{}
	}
	return slog.Any("request_id", id)
}

func EventType(eventType string) slog.Attr {
	return slog.String("event_type", eventType)
}

func MessageID(id any) slog.Attr {
	if id == nil {
		return slog.Attr{}
	}
	return slog.Any("message_id", id)
}

func RetryCount(count int) slog.Attr {
	return slog.Int("retry_count", count)
}

func Duration(d any) slog.Attr {
	return slog.Any("duration", d)
}

func Component(name string) slog.Attr {
	return slog.String("component", name)
}

func Event(name string) slog.Attr {
	return slog.String("event", name)
}

func Handler(name string) slog.Attr {
	return slog.String("handler", name)
}
