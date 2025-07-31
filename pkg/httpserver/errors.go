package httpserver

import "errors"

var (
	// ErrStart indicates that the server failed to start.
	ErrStart = errors.New("failed to start HTTP server")
	// ErrShutdown indicates that graceful shutdown failed.
	ErrShutdown = errors.New("failed to shutdown HTTP server gracefully")
)
