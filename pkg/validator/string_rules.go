package validator

import (
	"fmt"
	"strings"
)

// RequiredString validates that a string is not empty after trimming whitespace.
func RequiredString(field, value string) Rule {
	return Rule{
		Check: func() bool {
			return strings.TrimSpace(value) != ""
		},
		Error: ValidationError{
			Field:          field,
			Message:        "field is required",
			TranslationKey: "validation.required",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// MinLenString validates that a string has at least the minimum length.
func MinLenString(field, value string, min int) Rule {
	return Rule{
		Check: func() bool {
			return len(value) >= min
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be at least %d characters long", min),
			TranslationKey: "validation.min_length",
			TranslationValues: map[string]any{
				"field": field,
				"min":   min,
			},
		},
	}
}

// MaxLenString validates that a string has at most the maximum length.
func MaxLenString(field, value string, max int) Rule {
	return Rule{
		Check: func() bool {
			return len(value) <= max
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be at most %d characters long", max),
			TranslationKey: "validation.max_length",
			TranslationValues: map[string]any{
				"field": field,
				"max":   max,
			},
		},
	}
}

// LenString validates that a string has exactly the specified length.
func LenString(field, value string, exact int) Rule {
	return Rule{
		Check: func() bool {
			return len(value) == exact
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be exactly %d characters long", exact),
			TranslationKey: "validation.exact_length",
			TranslationValues: map[string]any{
				"field":  field,
				"length": exact,
			},
		},
	}
}

// Convenience aliases for common string validation cases

// Required is an alias for RequiredString for common string validation.
func Required(field, value string) Rule {
	return RequiredString(field, value)
}

// MinLen is an alias for MinLenString for common string validation.
func MinLen(field, value string, min int) Rule {
	return MinLenString(field, value, min)
}

// MaxLen is an alias for MaxLenString for common string validation.
func MaxLen(field, value string, max int) Rule {
	return MaxLenString(field, value, max)
}

// Len is an alias for LenString for common string validation.
func Len(field, value string, exact int) Rule {
	return LenString(field, value, exact)
}
