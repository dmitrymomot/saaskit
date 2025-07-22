package requestid

import (
	"context"
	"log/slog"
)

// LoggerExtractor returns a ContextExtractor for the logger
func LoggerExtractor() func(ctx context.Context) (slog.Attr, bool) {
	return func(ctx context.Context) (slog.Attr, bool) {
		if requestID := FromContext(ctx); requestID != "" {
			return slog.String("request_id", requestID), true
		}
		return slog.Attr{}, false
	}
}
