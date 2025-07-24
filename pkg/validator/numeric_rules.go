package validator

import "fmt"

// RequiredNum validates that a numeric value is not zero.
func RequiredNum[T Numeric](field string, value T) Rule {
	var zero T
	return Rule{
		Check: func() bool {
			return value != zero
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

// MinNum validates that a numeric value is greater than or equal to the minimum.
func MinNum[T Numeric](field string, value T, min T) Rule {
	return Rule{
		Check: func() bool {
			return value >= min
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be at least %v", min),
			TranslationKey: "validation.min",
			TranslationValues: map[string]any{
				"field": field,
				"min":   min,
			},
		},
	}
}

// MaxNum validates that a numeric value is less than or equal to the maximum.
func MaxNum[T Numeric](field string, value T, max T) Rule {
	return Rule{
		Check: func() bool {
			return value <= max
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be at most %v", max),
			TranslationKey: "validation.max",
			TranslationValues: map[string]any{
				"field": field,
				"max":   max,
			},
		},
	}
}

// Convenience aliases for common numeric validation cases

// Min is an alias for MinNum for common numeric validation.
func Min[T Numeric](field string, value T, min T) Rule {
	return MinNum(field, value, min)
}

// Max is an alias for MaxNum for common numeric validation.
func Max[T Numeric](field string, value T, max T) Rule {
	return MaxNum(field, value, max)
}
