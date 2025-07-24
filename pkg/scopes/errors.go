package scopes

import "errors"

var (
	// ErrInvalidScope is returned when a scope is not valid
	ErrInvalidScope = errors.New("invalid scope")
	// ErrScopeNotAllowed is returned when a scope is not in the list of allowed scopes
	ErrScopeNotAllowed = errors.New("scope not allowed")
)
