package audit

import "errors"

var (
	ErrStorageNotAvailable = errors.New("audit: storage backend is unavailable")
	ErrInvalidEvent        = errors.New("audit: invalid event data")
	ErrEventValidation     = errors.New("audit: event validation failed")
	ErrStorageTimeout      = errors.New("audit: storage operation timed out")
	ErrBufferFull          = errors.New("audit: async buffer is full")
)
