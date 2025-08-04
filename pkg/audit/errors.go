package audit

import "errors"

var (
	// ErrStorageNotAvailable indicates the storage backend is unavailable
	ErrStorageNotAvailable = errors.New("storage backend is unavailable")

	// ErrInvalidEvent indicates the event data is invalid
	ErrInvalidEvent = errors.New("invalid event data")
)
