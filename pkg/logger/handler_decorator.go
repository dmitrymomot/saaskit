package logger

import (
	"context"
	"log/slog"
)

// ContextExtractor extracts a slog attribute from context.
type ContextExtractor func(ctx context.Context) (slog.Attr, bool)

// LogHandlerDecorator wraps a slog.Handler and injects attributes from context.
// Uses the decorator pattern for minimal performance overhead - extraction only
// occurs during actual logging, avoiding the cost of creating new handlers.
type LogHandlerDecorator struct {
	next       slog.Handler
	extractors []ContextExtractor
}

// NewLogHandlerDecorator creates a new decorated handler.
// Filters nil extractors to prevent runtime panics - defensive programming
// to handle misconfigured options gracefully.
func NewLogHandlerDecorator(next slog.Handler, extractors ...ContextExtractor) slog.Handler {
	clean := make([]ContextExtractor, 0, len(extractors))
	for _, ex := range extractors {
		if ex != nil {
			clean = append(clean, ex)
		}
	}
	return &LogHandlerDecorator{next: next, extractors: clean}
}

func (h *LogHandlerDecorator) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle extracts context attributes and delegates to the underlying handler.
// Context extraction occurs per-log-call to ensure fresh request-scoped values
// like request IDs are captured, rather than stale cached values.
func (h *LogHandlerDecorator) Handle(ctx context.Context, rec slog.Record) error {
	// Early exit when no extractors to avoid unnecessary processing
	if len(h.extractors) == 0 {
		return h.next.Handle(ctx, rec)
	}

	for _, ex := range h.extractors {
		if attr, ok := ex(ctx); ok {
			rec.AddAttrs(attr)
		}
	}
	return h.next.Handle(ctx, rec)
}

// WithAttrs creates a new decorated handler with additional static attributes.
// Preserves context extractors while delegating attribute handling to the underlying handler.
func (h *LogHandlerDecorator) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandlerDecorator{
		next:       h.next.WithAttrs(attrs),
		extractors: h.extractors,
	}
}

// WithGroup creates a new decorated handler with attribute grouping.
// Preserves context extractors while delegating grouping to the underlying handler.
func (h *LogHandlerDecorator) WithGroup(name string) slog.Handler {
	return &LogHandlerDecorator{
		next:       h.next.WithGroup(name),
		extractors: h.extractors,
	}
}
