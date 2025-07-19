package saaskit

import "errors"

// Predefined package errors
var (
	ErrNilResponse = errors.New("handler returned nil response")
)
