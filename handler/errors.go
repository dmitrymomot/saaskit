package handler

import "errors"

// Predefined package errors
var (
	ErrNilResponse       = errors.New("handler returned nil response")
	ErrSSENotInitialized = errors.New("SSE not initialized for this request")
)
