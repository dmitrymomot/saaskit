package handler

import "errors"

// Package-level errors for common failure scenarios
var (
	// ErrNilResponse indicates a handler returned nil instead of a Response
	ErrNilResponse = errors.New("handler returned nil response")
	// ErrSSENotInitialized indicates SSE was accessed before being set up for the request
	ErrSSENotInitialized = errors.New("SSE not initialized for this request")
)
