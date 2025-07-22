package validator

import "fmt"

// RequiredSlice validates that a slice is not empty.
func RequiredSlice[T any](field string, value []T) Rule {
	return Rule{
		Check: func() bool {
			return len(value) > 0
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

// MinLenSlice validates that a slice has at least the minimum number of items.
func MinLenSlice[T any](field string, value []T, min int) Rule {
	return Rule{
		Check: func() bool {
			return len(value) >= min
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must have at least %d items", min),
			TranslationKey: "validation.min_items",
			TranslationValues: map[string]any{
				"field": field,
				"min":   min,
			},
		},
	}
}

// MaxLenSlice validates that a slice has at most the maximum number of items.
func MaxLenSlice[T any](field string, value []T, max int) Rule {
	return Rule{
		Check: func() bool {
			return len(value) <= max
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must have at most %d items", max),
			TranslationKey: "validation.max_items",
			TranslationValues: map[string]any{
				"field": field,
				"max":   max,
			},
		},
	}
}

// LenSlice validates that a slice has exactly the specified number of items.
func LenSlice[T any](field string, value []T, exact int) Rule {
	return Rule{
		Check: func() bool {
			return len(value) == exact
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must have exactly %d items", exact),
			TranslationKey: "validation.exact_items",
			TranslationValues: map[string]any{
				"field": field,
				"count": exact,
			},
		},
	}
}

// RequiredMap validates that a map is not empty.
func RequiredMap[K comparable, V any](field string, value map[K]V) Rule {
	return Rule{
		Check: func() bool {
			return len(value) > 0
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

// MinLenMap validates that a map has at least the minimum number of items.
func MinLenMap[K comparable, V any](field string, value map[K]V, min int) Rule {
	return Rule{
		Check: func() bool {
			return len(value) >= min
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must have at least %d items", min),
			TranslationKey: "validation.min_items",
			TranslationValues: map[string]any{
				"field": field,
				"min":   min,
			},
		},
	}
}

// MaxLenMap validates that a map has at most the maximum number of items.
func MaxLenMap[K comparable, V any](field string, value map[K]V, max int) Rule {
	return Rule{
		Check: func() bool {
			return len(value) <= max
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must have at most %d items", max),
			TranslationKey: "validation.max_items",
			TranslationValues: map[string]any{
				"field": field,
				"max":   max,
			},
		},
	}
}

// LenMap validates that a map has exactly the specified number of items.
func LenMap[K comparable, V any](field string, value map[K]V, exact int) Rule {
	return Rule{
		Check: func() bool {
			return len(value) == exact
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must have exactly %d items", exact),
			TranslationKey: "validation.exact_items",
			TranslationValues: map[string]any{
				"field": field,
				"count": exact,
			},
		},
	}
}
