package validator

import (
	"errors"
	"fmt"
	"strings"
)

type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// ValidationError represents a single validation error with translation support.
type ValidationError struct {
	Field             string
	Message           string
	TranslationKey    string
	TranslationValues map[string]any
}

// ValidationErrors represents a collection of validation errors.
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "validation failed"
	}

	var parts []string
	for _, err := range ve {
		parts = append(parts, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

func (ve *ValidationErrors) Add(err ValidationError) {
	*ve = append(*ve, err)
}

func (ve ValidationErrors) Has(field string) bool {
	for _, err := range ve {
		if err.Field == field {
			return true
		}
	}
	return false
}

func (ve ValidationErrors) Get(field string) []string {
	var messages []string
	for _, err := range ve {
		if err.Field == field {
			messages = append(messages, err.Message)
		}
	}
	return messages
}

func (ve ValidationErrors) GetErrors(field string) []ValidationError {
	var errors []ValidationError
	for _, err := range ve {
		if err.Field == field {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ve ValidationErrors) Fields() []string {
	var fields []string
	seen := make(map[string]bool)
	for _, err := range ve {
		if !seen[err.Field] {
			fields = append(fields, err.Field)
			seen[err.Field] = true
		}
	}
	return fields
}

func (ve ValidationErrors) IsEmpty() bool {
	return len(ve) == 0
}

func (ve ValidationErrors) GetTranslatableErrors() []ValidationError {
	return ve
}

// Rule represents a single validation rule.
type Rule struct {
	Check func() bool
	Error ValidationError
}

// Apply executes multiple validation rules and returns any validation errors.
func Apply(rules ...Rule) error {
	var errors ValidationErrors

	for _, rule := range rules {
		if !rule.Check() {
			errors = append(errors, rule.Error)
		}
	}

	if errors.IsEmpty() {
		return nil
	}

	return errors
}

// ExtractValidationErrors extracts ValidationErrors from an error.
func ExtractValidationErrors(err error) ValidationErrors {
	if err == nil {
		return nil
	}

	var validationErr ValidationErrors
	if errors.As(err, &validationErr) {
		return validationErr
	}

	return nil
}

func IsValidationError(err error) bool {
	if err == nil {
		return false
	}

	var validationErr ValidationErrors
	return errors.As(err, &validationErr)
}
