package validator

// RequiredComparable validates that a comparable value is not its zero value.
func RequiredComparable[T comparable](field string, value T) Rule {
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
