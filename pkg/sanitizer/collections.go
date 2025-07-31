package sanitizer

import (
	"maps"
	"sort"
	"strings"
)

// FilterEmpty removes whitespace-only entries to prevent empty form fields from polluting data.
func FilterEmpty(slice []string) []string {
	result := make([]string, 0)
	for _, item := range slice {
		if strings.TrimSpace(item) != "" {
			result = append(result, item)
		}
	}
	return result
}

// Deduplicate preserves first occurrence order to maintain user intent in form submissions.
func Deduplicate[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := make([]T, 0)

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

func DeduplicateStrings(slice []string) []string {
	return Deduplicate(slice)
}

// DeduplicateStringsIgnoreCase preserves original casing of first occurrence.
func DeduplicateStringsIgnoreCase(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range slice {
		lower := strings.ToLower(item)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, item)
		}
	}

	return result
}

// LimitSliceLength prevents memory exhaustion from malicious input arrays.
func LimitSliceLength[T any](slice []T, maxLength int) []T {
	if maxLength <= 0 {
		return []T{}
	}

	if len(slice) <= maxLength {
		return slice
	}

	return slice[:maxLength]
}

// SortStrings creates sorted copy to avoid mutating input slice.
func SortStrings(slice []string) []string {
	result := make([]string, len(slice))
	copy(result, slice)
	sort.Strings(result)
	return result
}

// SortStringsIgnoreCase preserves original casing while sorting by lowercase comparison.
func SortStringsIgnoreCase(slice []string) []string {
	result := make([]string, len(slice))
	copy(result, slice)

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})

	return result
}

// FilterSliceByPattern uses case-insensitive substring matching for user-friendly filtering.
func FilterSliceByPattern(slice []string, pattern string) []string {
	result := make([]string, 0)
	for _, item := range slice {
		if !strings.Contains(strings.ToLower(item), strings.ToLower(pattern)) {
			result = append(result, item)
		}
	}
	return result
}

func TrimStringSlice(slice []string) []string {
	result := make([]string, len(slice))
	for i, item := range slice {
		result[i] = strings.TrimSpace(item)
	}
	return result
}

func ToLowerStringSlice(slice []string) []string {
	result := make([]string, len(slice))
	for i, item := range slice {
		result[i] = strings.ToLower(item)
	}
	return result
}

// CleanStringSlice applies standard form data cleanup pipeline.
func CleanStringSlice(slice []string) []string {
	return Apply(slice,
		TrimStringSlice,
		FilterEmpty,
		DeduplicateStrings,
	)
}

// SanitizeMapKeys drops entries with empty keys after sanitization to prevent key collisions.
func SanitizeMapKeys[V any](m map[string]V, sanitizer func(string) string) map[string]V {
	result := make(map[string]V)
	for k, v := range m {
		cleanKey := sanitizer(k)
		if cleanKey != "" {
			result[cleanKey] = v
		}
	}
	return result
}

func SanitizeMapValues[K comparable](m map[K]string, sanitizer func(string) string) map[K]string {
	result := make(map[K]string)
	for k, v := range m {
		result[k] = sanitizer(v)
	}
	return result
}

// FilterMapByKeys uses case-insensitive substring matching for consistent behavior.
func FilterMapByKeys[V any](m map[string]V, pattern string) map[string]V {
	result := make(map[string]V)
	lowerPattern := strings.ToLower(pattern)

	for k, v := range m {
		if !strings.Contains(strings.ToLower(k), lowerPattern) {
			result[k] = v
		}
	}

	return result
}

func FilterMapByValues[K comparable](m map[K]string, pattern string) map[K]string {
	result := make(map[K]string)
	lowerPattern := strings.ToLower(pattern)

	for k, v := range m {
		if !strings.Contains(strings.ToLower(v), lowerPattern) {
			result[k] = v
		}
	}

	return result
}

// FilterEmptyMapValues removes whitespace-only values to prevent empty data storage.
func FilterEmptyMapValues[K comparable](m map[K]string) map[K]string {
	result := make(map[K]string)
	for k, v := range m {
		if strings.TrimSpace(v) != "" {
			result[k] = v
		}
	}
	return result
}

// CleanStringMap applies standard form data cleanup: lowercase keys, trim values, remove empties.
func CleanStringMap(m map[string]string) map[string]string {
	cleaned := SanitizeMapKeys(m, func(s string) string {
		return Apply(s, Trim, ToLower)
	})

	cleaned = SanitizeMapValues(cleaned, func(s string) string {
		return Apply(s, Trim)
	})

	cleaned = FilterEmptyMapValues(cleaned)

	return cleaned
}

// LimitMapSize prevents memory exhaustion from malicious input; iteration order is random.
func LimitMapSize[K comparable, V any](m map[K]V, maxSize int) map[K]V {
	if maxSize <= 0 {
		return make(map[K]V)
	}

	if len(m) <= maxSize {
		return m
	}

	result := make(map[K]V)
	count := 0

	for k, v := range m {
		if count >= maxSize {
			break
		}
		result[k] = v
		count++
	}

	return result
}

func ExtractMapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func ExtractMapValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// MergeStringMaps applies last-writer-wins semantics for duplicate keys.
func MergeStringMaps(ms ...map[string]string) map[string]string {
	result := make(map[string]string)

	for _, m := range ms {
		maps.Copy(result, m)
	}

	return result
}

func SliceToMap[T any](slice []T) map[int]T {
	result := make(map[int]T)
	for i, item := range slice {
		result[i] = item
	}
	return result
}

// MapToSlice order is non-deterministic due to Go's map iteration randomization.
func MapToSlice[K comparable, V any](m map[K]V) []V {
	return ExtractMapValues(m)
}

func FilterSlice[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0)
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

func TransformSlice[T any, R any](slice []T, transform func(T) R) []R {
	result := make([]R, len(slice))
	for i, item := range slice {
		result[i] = transform(item)
	}
	return result
}

func ReverseSlice[T any](slice []T) []T {
	result := make([]T, len(slice))
	for i, item := range slice {
		result[len(slice)-1-i] = item
	}
	return result
}
