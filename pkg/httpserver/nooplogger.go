package httpserver

import (
	"context"
	"log/slog"
)

// noopHandler is a slog.Handler that discards all logs.
type noopHandler struct{}

func (n noopHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (n noopHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (n noopHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return n }
func (n noopHandler) WithGroup(_ string) slog.Handler               { return n }

// newNoopLogger returns a slog.Logger that discards all logs.
func newNoopLogger() *slog.Logger {
	return slog.New(noopHandler{})
}
