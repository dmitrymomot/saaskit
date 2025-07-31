package scopes

import "errors"

var (
	// ErrInvalidScope is returned when a scope is not valid
	ErrInvalidScope = errors.New("scopes: invalid scope format")
	// ErrScopeNotAllowed is returned when a scope is not in the list of allowed scopes
	ErrScopeNotAllowed = errors.New("scopes: scope not in allowed list")
)
