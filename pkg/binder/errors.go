package binder

import "errors"

// Common binding errors
var (
	ErrUnsupportedMediaType = errors.New("unsupported media type")
	ErrFailedToParseJSON    = errors.New("failed to parse JSON request body")
	ErrFailedToParseForm    = errors.New("failed to parse form data")
	ErrFailedToParseQuery   = errors.New("failed to parse query parameters")
	ErrFailedToParsePath    = errors.New("failed to parse path parameters")
	ErrMissingContentType   = errors.New("missing content type")
)
