package pg

import "context"

// logger defines the interface required for migration logging integration.
// Compatible with slog and other structured loggers, required for goose migration
// output routing to application logging instead of stdout/stderr.
type logger interface {
	InfoContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}
