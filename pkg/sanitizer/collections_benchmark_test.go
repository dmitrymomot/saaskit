package sanitizer_test

import (
	"strings"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

func BenchmarkSanitizeSlice(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		input := make([]string, size)
		for i := range size {
			input[i] = "  Test String  "
		}

		b.Run(string(rune(size)), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				// Use TransformSlice to apply sanitizers
				_ = sanitizer.TransformSlice(input, func(s string) string {
					return sanitizer.Apply(s, sanitizer.Trim, sanitizer.ToLower)
				})
			}
		})
	}
}

func BenchmarkSanitizeMap(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		input := make(map[string]string, size)
		for i := range size {
			key := strings.Repeat("k", i%10+1)
			input[key] = "  Value String  "
		}

		b.Run(string(rune(size)), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				// Use SanitizeMapValues
				_ = sanitizer.SanitizeMapValues(input, func(s string) string {
					return sanitizer.Apply(s, sanitizer.Trim, sanitizer.ToUpper)
				})
			}
		})
	}
}

func BenchmarkSanitizeMapKeys(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		input := make(map[string]string, size)
		for i := range size {
			key := "  Key " + string(rune(i)) + "  "
			input[key] = "value"
		}

		b.Run(string(rune(size)), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.SanitizeMapKeys(input, func(s string) string {
					return sanitizer.Apply(s, sanitizer.Trim, sanitizer.ToSnakeCase)
				})
			}
		})
	}
}

func BenchmarkFilterSlice(b *testing.B) {
	sizes := []int{10, 100, 1000}

	isNotEmpty := func(s string) bool {
		return strings.TrimSpace(s) != ""
	}

	for _, size := range sizes {
		input := make([]string, size)
		for i := range size {
			if i%3 == 0 {
				input[i] = ""
			} else {
				input[i] = "value"
			}
		}

		b.Run(string(rune(size)), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.FilterSlice(input, isNotEmpty)
			}
		})
	}
}

func BenchmarkFilterMapByValues(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		input := make(map[string]string, size)
		for i := range size {
			key := "key" + string(rune(i))
			if i%2 == 0 {
				input[key] = "test@example.com"
			} else {
				input[key] = "invalid"
			}
		}

		b.Run(string(rune(size)), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				// Filter by value pattern
				_ = sanitizer.FilterMapByValues(input, "invalid")
			}
		})
	}
}

func BenchmarkDeduplicateSlice(b *testing.B) {
	testCases := []struct {
		name  string
		input []string
	}{
		{"no_duplicates", []string{"a", "b", "c", "d", "e"}},
		{"all_duplicates", []string{"a", "a", "a", "a", "a"}},
		{"mixed", []string{"a", "b", "a", "c", "b", "d", "c", "e", "d"}},
		{"large", generateDuplicateSlice(1000, 100)},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.Deduplicate(tc.input)
			}
		})
	}
}

func BenchmarkNormalizeSlice(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		input := make([]string, size)
		for i := range size {
			input[i] = "  Test@EXAMPLE.com  "
		}

		b.Run(string(rune(size)), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				// Use TransformSlice with NormalizeEmail
				_ = sanitizer.TransformSlice(input, sanitizer.NormalizeEmail)
			}
		})
	}
}

func BenchmarkSanitizeStruct(b *testing.B) {
	type TestStruct struct {
		Name  string
		Email string
		Phone string
		URL   string
	}

	input := TestStruct{
		Name:  "  John Doe  ",
		Email: "  JOHN.DOE@EXAMPLE.COM  ",
		Phone: "(555) 123-4567",
		URL:   "  HTTP://EXAMPLE.COM/  ",
	}

	b.ResetTimer()
	for b.Loop() {
		// Manually apply transformations since SanitizeStruct doesn't exist
		result := TestStruct{
			Name:  sanitizer.Apply(input.Name, sanitizer.Trim, sanitizer.RemoveExtraWhitespace),
			Email: sanitizer.Apply(input.Email, sanitizer.Trim, sanitizer.NormalizeEmail),
			Phone: sanitizer.NormalizePhone(input.Phone),
			URL:   sanitizer.Apply(input.URL, sanitizer.Trim, sanitizer.NormalizeURL),
		}
		_ = result
	}
}

// Helper function to generate slice with duplicates
func generateDuplicateSlice(size, uniqueValues int) []string {
	result := make([]string, size)
	for i := range size {
		result[i] = string(rune('a' + (i % uniqueValues)))
	}
	return result
}
