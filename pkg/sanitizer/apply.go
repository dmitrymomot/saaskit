package sanitizer

// Apply creates functional composition pipeline for sanitization transformations.
// Useful for building complex sanitization chains while maintaining type safety.
func Apply[T any](value T, transforms ...func(T) T) T {
	result := value

	for _, transform := range transforms {
		result = transform(result)
	}

	return result
}

// Compose creates reusable sanitization pipelines that can be stored and reused.
// Preferred over repeated Apply calls when the same transformation chain is used multiple times.
func Compose[T any](transforms ...func(T) T) func(T) T {
	return func(value T) T {
		return Apply(value, transforms...)
	}
}
