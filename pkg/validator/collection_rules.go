package validator

import "fmt"

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
