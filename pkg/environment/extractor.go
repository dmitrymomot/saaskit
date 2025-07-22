package environment

import (
	"context"
	"log/slog"
)

// LoggerExtractor returns a ContextExtractor for the logger
func LoggerExtractor() func(ctx context.Context) (slog.Attr, bool) {
	return func(ctx context.Context) (slog.Attr, bool) {
		if env := FromContext(ctx); env != "" {
			return slog.String("env", env), true
		}
		return slog.Attr{}, false
	}
}
