package validator

import "errors"

// Common validation errors that can be used across the application.
var (
	// ErrValidationFailed is returned when validation fails but no specific error is provided.
	ErrValidationFailed = errors.New("validation failed")

	// ErrFieldRequired is returned when a required field is empty.
	ErrFieldRequired = errors.New("field is required")

	// ErrInvalidLength is returned when a field has an invalid length.
	ErrInvalidLength = errors.New("invalid length")

	// ErrInvalidValue is returned when a field has an invalid value.
	ErrInvalidValue = errors.New("invalid value")

	// ErrOutOfRange is returned when a numeric value is out of the allowed range.
	ErrOutOfRange = errors.New("value out of range")

	// ErrInvalidFormat is returned when a field has an invalid format.
	ErrInvalidFormat = errors.New("invalid format")
)
