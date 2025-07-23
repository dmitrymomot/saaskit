package session

import "errors"

var (
	// ErrInvalidSession indicates the session fingerprint doesn't match
	ErrInvalidSession = errors.New("invalid session")

	// ErrSessionExpired indicates the session has expired
	ErrSessionExpired = errors.New("session expired")

	// ErrSessionNotFound indicates no session was found
	ErrSessionNotFound = errors.New("session not found")

	// ErrTokenGeneration indicates token generation failed
	ErrTokenGeneration = errors.New("failed to generate session token")

	// ErrNoTransport indicates no transport is configured
	ErrNoTransport = errors.New("no session transport configured")

	// ErrNoStore indicates no store is configured
	ErrNoStore = errors.New("no session store configured")
)
