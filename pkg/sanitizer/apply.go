package sanitizer

// Apply executes multiple transformation functions sequentially on a value.
// Each transform function is applied to the result of the previous transformation.
// The type of the input value is preserved throughout the transformations.
func Apply[T any](value T, transforms ...func(T) T) T {
	result := value

	for _, transform := range transforms {
		result = transform(result)
	}

	return result
}

// Compose creates a single transformation function that applies multiple transformations in sequence.
// This is useful for creating reusable sanitization pipelines.
func Compose[T any](transforms ...func(T) T) func(T) T {
	return func(value T) T {
		return Apply(value, transforms...)
	}
}
