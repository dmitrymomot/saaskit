package httpserver

import "errors"

var (
	// ErrStart indicates that the server failed to start.
	ErrStart = errors.New("httpserver: start failed")
	// ErrShutdown indicates that graceful shutdown failed.
	ErrShutdown = errors.New("httpserver: shutdown failed")
)
