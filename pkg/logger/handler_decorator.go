package logger

import (
	"context"
	"log/slog"
)

// ContextExtractor extracts a slog attribute from context.
type ContextExtractor func(ctx context.Context) (slog.Attr, bool)

// LogHandlerDecorator wraps a slog.Handler and injects attributes from context.
type LogHandlerDecorator struct {
	next       slog.Handler
	extractors []ContextExtractor
}

// NewLogHandlerDecorator creates a new decorated handler.
func NewLogHandlerDecorator(next slog.Handler, extractors ...ContextExtractor) slog.Handler {
	clean := make([]ContextExtractor, 0, len(extractors))
	for _, ex := range extractors {
		if ex != nil {
			clean = append(clean, ex)
		}
	}
	return &LogHandlerDecorator{next: next, extractors: clean}
}

// Enabled implements slog.Handler.
func (h *LogHandlerDecorator) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (h *LogHandlerDecorator) Handle(ctx context.Context, rec slog.Record) error {
	for _, ex := range h.extractors {
		if attr, ok := ex(ctx); ok {
			rec.AddAttrs(attr)
		}
	}
	return h.next.Handle(ctx, rec)
}

// WithAttrs implements slog.Handler.
func (h *LogHandlerDecorator) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandlerDecorator{
		next:       h.next.WithAttrs(attrs),
		extractors: h.extractors,
	}
}

// WithGroup implements slog.Handler.
func (h *LogHandlerDecorator) WithGroup(name string) slog.Handler {
	return &LogHandlerDecorator{
		next:       h.next.WithGroup(name),
		extractors: h.extractors,
	}
}
