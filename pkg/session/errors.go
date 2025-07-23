package session

import "errors"

var (
	// ErrInvalidSession indicates the session fingerprint doesn't match
	ErrInvalidSession = errors.New("session.invalid")

	// ErrSessionExpired indicates the session has expired
	ErrSessionExpired = errors.New("session.expired")

	// ErrSessionNotFound indicates no session was found
	ErrSessionNotFound = errors.New("session.not_found")

	// ErrTokenGeneration indicates token generation failed
	ErrTokenGeneration = errors.New("session.token_generation_failed")

	// ErrNoTransport indicates no transport is configured
	ErrNoTransport = errors.New("session.no_transport")

	// ErrNoStore indicates no store is configured
	ErrNoStore = errors.New("session.no_store")
)
