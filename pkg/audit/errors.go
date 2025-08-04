package audit

import "errors"

var (
	// ErrStorageNotAvailable indicates the storage backend is unavailable
	ErrStorageNotAvailable = errors.New("storage backend is unavailable")

	// ErrInvalidEvent indicates the event data is invalid
	ErrInvalidEvent = errors.New("invalid event data")

	// ErrEventValidation indicates event validation failed
	ErrEventValidation = errors.New("event validation failed")

	// ErrStorageTimeout indicates a storage operation timed out
	ErrStorageTimeout = errors.New("storage operation timed out")

	// ErrBufferFull indicates the async buffer is full
	ErrBufferFull = errors.New("async buffer is full")
)
