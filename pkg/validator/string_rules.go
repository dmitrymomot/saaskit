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

func Required(field, value string) Rule {
	return RequiredString(field, value)
}

func MinLen(field, value string, min int) Rule {
	return MinLenString(field, value, min)
}

func MaxLen(field, value string, max int) Rule {
	return MaxLenString(field, value, max)
}

func Len(field, value string, exact int) Rule {
	return LenString(field, value, exact)
}
