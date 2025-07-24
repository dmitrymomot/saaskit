package sanitizer

import (
	"maps"
	"sort"
	"strings"
)

// FilterEmpty removes empty strings from a slice.
func FilterEmpty(slice []string) []string {
	result := make([]string, 0)
	for _, item := range slice {
		if strings.TrimSpace(item) != "" {
			result = append(result, item)
		}
	}
	return result
}

// Deduplicate removes duplicate items from a slice while preserving order.
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

// DeduplicateStrings removes duplicate strings (case-sensitive).
func DeduplicateStrings(slice []string) []string {
	return Deduplicate(slice)
}

// DeduplicateStringsIgnoreCase removes duplicate strings (case-insensitive).
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

// LimitSliceLength truncates a slice to the specified maximum length.
func LimitSliceLength[T any](slice []T, maxLength int) []T {
	if maxLength <= 0 {
		return []T{}
	}

	if len(slice) <= maxLength {
		return slice
	}

	return slice[:maxLength]
}

// SortStrings sorts a string slice in ascending order.
func SortStrings(slice []string) []string {
	result := make([]string, len(slice))
	copy(result, slice)
	sort.Strings(result)
	return result
}

// SortStringsIgnoreCase sorts a string slice in ascending order (case-insensitive).
func SortStringsIgnoreCase(slice []string) []string {
	result := make([]string, len(slice))
	copy(result, slice)

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})

	return result
}

// FilterSliceByPattern removes strings that match the given pattern.
func FilterSliceByPattern(slice []string, pattern string) []string {
	result := make([]string, 0)
	for _, item := range slice {
		if !strings.Contains(strings.ToLower(item), strings.ToLower(pattern)) {
			result = append(result, item)
		}
	}
	return result
}

// TrimStringSlice trims whitespace from all strings in a slice.
func TrimStringSlice(slice []string) []string {
	result := make([]string, len(slice))
	for i, item := range slice {
		result[i] = strings.TrimSpace(item)
	}
	return result
}

// ToLowerStringSlice converts all strings in a slice to lowercase.
func ToLowerStringSlice(slice []string) []string {
	result := make([]string, len(slice))
	for i, item := range slice {
		result[i] = strings.ToLower(item)
	}
	return result
}

// CleanStringSlice applies comprehensive cleaning to a string slice.
func CleanStringSlice(slice []string) []string {
	return Apply(slice,
		TrimStringSlice,
		FilterEmpty,
		DeduplicateStrings,
	)
}

// SanitizeMapKeys applies a sanitization function to all keys in a map.
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

// SanitizeMapValues applies a sanitization function to all values in a map.
func SanitizeMapValues[K comparable](m map[K]string, sanitizer func(string) string) map[K]string {
	result := make(map[K]string)
	for k, v := range m {
		result[k] = sanitizer(v)
	}
	return result
}

// FilterMapByKeys removes map entries where keys match the filter pattern.
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

// FilterMapByValues removes map entries where string values match the filter pattern.
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

// FilterEmptyMapValues removes map entries with empty string values.
func FilterEmptyMapValues[K comparable](m map[K]string) map[K]string {
	result := make(map[K]string)
	for k, v := range m {
		if strings.TrimSpace(v) != "" {
			result[k] = v
		}
	}
	return result
}

// CleanStringMap applies comprehensive cleaning to a string map.
func CleanStringMap(m map[string]string) map[string]string {
	// Clean keys
	cleaned := SanitizeMapKeys(m, func(s string) string {
		return Apply(s, Trim, ToLower)
	})

	// Clean values
	cleaned = SanitizeMapValues(cleaned, func(s string) string {
		return Apply(s, Trim)
	})

	// Remove empty values
	cleaned = FilterEmptyMapValues(cleaned)

	return cleaned
}

// LimitMapSize limits the number of entries in a map.
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

// ExtractMapKeys returns a slice of all keys from a map.
func ExtractMapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ExtractMapValues returns a slice of all values from a map.
func ExtractMapValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// MergeStringMaps merges multiple string maps, with later maps overriding earlier ones.
func MergeStringMaps(ms ...map[string]string) map[string]string {
	result := make(map[string]string)

	for _, m := range ms {
		maps.Copy(result, m)
	}

	return result
}

// SliceToMap converts a slice to a map with indices as keys.
func SliceToMap[T any](slice []T) map[int]T {
	result := make(map[int]T)
	for i, item := range slice {
		result[i] = item
	}
	return result
}

// MapToSlice converts map values to a slice (order not guaranteed).
func MapToSlice[K comparable, V any](m map[K]V) []V {
	return ExtractMapValues(m)
}

// FilterSlice filters slice elements using a predicate function.
func FilterSlice[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0)
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// TransformSlice applies a transformation function to each element in a slice.
func TransformSlice[T any, R any](slice []T, transform func(T) R) []R {
	result := make([]R, len(slice))
	for i, item := range slice {
		result[i] = transform(item)
	}
	return result
}

// ReverseSlice reverses the order of elements in a slice.
func ReverseSlice[T any](slice []T) []T {
	result := make([]T, len(slice))
	for i, item := range slice {
		result[len(slice)-1-i] = item
	}
	return result
}
