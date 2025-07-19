package saaskit

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidationError represents field validation errors.
// It's based on url.Values to leverage built-in string slice handling.
type ValidationError url.Values

// Error implements the error interface.
// Returns a human-readable error message summarizing validation failures.
func (e ValidationError) Error() string {
	if len(e) == 0 {
		return "Validation failed"
	}

	var parts []string
	for field, messages := range e {
		if len(messages) > 0 {
			parts = append(parts, fmt.Sprintf("%s: %s", field, messages[0]))
		}
	}

	return fmt.Sprintf("validation error: %s", strings.Join(parts, ", "))
}

// NewValidationError creates a new validation error.
func NewValidationError() ValidationError {
	return make(ValidationError)
}

// Add adds an error message for a field.
func (e ValidationError) Add(field, message string) {
	url.Values(e).Add(field, message)
}

// Get returns the first error message for a field.
func (e ValidationError) Get(field string) string {
	return url.Values(e).Get(field)
}

// Has checks if a field has any errors.
func (e ValidationError) Has(field string) bool {
	return len(e[field]) > 0
}

// IsEmpty returns true if there are no validation errors.
func (e ValidationError) IsEmpty() bool {
	return len(e) == 0
}
