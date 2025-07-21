package binder

import "errors"

// Common binding errors
var (
	ErrUnsupportedMediaType = errors.New("unsupported media type")
	ErrInvalidJSON          = errors.New("invalid JSON")
	ErrInvalidForm          = errors.New("invalid form data")
	ErrInvalidQuery         = errors.New("invalid query parameter")
	ErrMissingContentType   = errors.New("missing content type")
)
