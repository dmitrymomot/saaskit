package sanitizer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

func TestFilterEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "filters empty strings",
			input:    []string{"hello", "", "world", "   ", "test"},
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "handles all empty",
			input:    []string{"", "   ", "\t\n"},
			expected: []string{},
		},
		{
			name:     "handles no empty strings",
			input:    []string{"hello", "world", "test"},
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "handles empty slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.FilterEmpty(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeduplicate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "removes duplicate strings",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "preserves order of first occurrence",
			input:    []string{"first", "second", "first", "third"},
			expected: []string{"first", "second", "third"},
		},
		{
			name:     "handles no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "handles empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "handles all same",
			input:    []string{"same", "same", "same"},
			expected: []string{"same"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.Deduplicate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeduplicateStringsIgnoreCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "removes case-insensitive duplicates",
			input:    []string{"Hello", "WORLD", "hello", "World"},
			expected: []string{"Hello", "WORLD"},
		},
		{
			name:     "preserves original case of first occurrence",
			input:    []string{"Test", "test", "TEST"},
			expected: []string{"Test"},
		},
		{
			name:     "handles mixed case variations",
			input:    []string{"Apple", "BANANA", "apple", "Banana", "APPLE"},
			expected: []string{"Apple", "BANANA"},
		},
		{
			name:     "handles empty slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.DeduplicateStringsIgnoreCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLimitSliceLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     []string
		maxLength int
		expected  []string
	}{
		{
			name:      "truncates long slice",
			input:     []string{"a", "b", "c", "d", "e"},
			maxLength: 3,
			expected:  []string{"a", "b", "c"},
		},
		{
			name:      "preserves short slice",
			input:     []string{"a", "b"},
			maxLength: 5,
			expected:  []string{"a", "b"},
		},
		{
			name:      "handles zero length",
			input:     []string{"a", "b", "c"},
			maxLength: 0,
			expected:  []string{},
		},
		{
			name:      "handles negative length",
			input:     []string{"a", "b", "c"},
			maxLength: -1,
			expected:  []string{},
		},
		{
			name:      "handles empty slice",
			input:     []string{},
			maxLength: 3,
			expected:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.LimitSliceLength(tt.input, tt.maxLength)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "sorts strings alphabetically",
			input:    []string{"zebra", "apple", "banana"},
			expected: []string{"apple", "banana", "zebra"},
		},
		{
			name:     "handles already sorted",
			input:    []string{"apple", "banana", "zebra"},
			expected: []string{"apple", "banana", "zebra"},
		},
		{
			name:     "handles single item",
			input:    []string{"single"},
			expected: []string{"single"},
		},
		{
			name:     "handles empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "case sensitive sort",
			input:    []string{"Zebra", "apple", "Banana"},
			expected: []string{"Banana", "Zebra", "apple"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.SortStrings(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortStringsIgnoreCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "sorts ignoring case",
			input:    []string{"Zebra", "apple", "Banana"},
			expected: []string{"apple", "Banana", "Zebra"},
		},
		{
			name:     "preserves original case",
			input:    []string{"HELLO", "world", "Test"},
			expected: []string{"HELLO", "Test", "world"},
		},
		{
			name:     "handles empty slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.SortStringsIgnoreCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterSliceByPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		pattern  string
		expected []string
	}{
		{
			name:     "filters by pattern",
			input:    []string{"apple", "banana", "grape", "pineapple"},
			pattern:  "apple",
			expected: []string{"banana", "grape"},
		},
		{
			name:     "case insensitive filtering",
			input:    []string{"Apple", "BANANA", "grape"},
			pattern:  "apple",
			expected: []string{"BANANA", "grape"},
		},
		{
			name:     "no matches",
			input:    []string{"apple", "banana", "grape"},
			pattern:  "orange",
			expected: []string{"apple", "banana", "grape"},
		},
		{
			name:     "empty pattern",
			input:    []string{"apple", "banana"},
			pattern:  "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.FilterSliceByPattern(tt.input, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrimStringSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "trims whitespace from all strings",
			input:    []string{"  hello  ", "\tworld\n", " test "},
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "handles already trimmed",
			input:    []string{"hello", "world"},
			expected: []string{"hello", "world"},
		},
		{
			name:     "handles empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "handles empty strings",
			input:    []string{"", "   ", "\t\n"},
			expected: []string{"", "", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.TrimStringSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToLowerStringSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "converts to lowercase",
			input:    []string{"HELLO", "World", "TEST"},
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "handles already lowercase",
			input:    []string{"hello", "world"},
			expected: []string{"hello", "world"},
		},
		{
			name:     "handles mixed case",
			input:    []string{"HeLLo", "WoRLd"},
			expected: []string{"hello", "world"},
		},
		{
			name:     "handles empty slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.ToLowerStringSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanStringSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "comprehensive cleaning",
			input:    []string{"  hello  ", "", "world", "  hello  ", "test"},
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "handles all empty",
			input:    []string{"", "   ", "\t"},
			expected: []string{},
		},
		{
			name:     "handles duplicates after trimming",
			input:    []string{"hello", "  hello  ", "world"},
			expected: []string{"hello", "world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.CleanStringSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeMapKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name: "sanitizes keys to lowercase",
			input: map[string]string{
				"NAME":   "John",
				"EMAIL":  "john@example.com",
				"  ID  ": "123",
			},
			expected: map[string]string{
				"name":  "John",
				"email": "john@example.com",
				"id":    "123",
			},
		},
		{
			name:     "handles empty map",
			input:    map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "removes keys that become empty",
			input: map[string]string{
				"   ": "value",
				"key": "value2",
			},
			expected: map[string]string{
				"key": "value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.SanitizeMapKeys(tt.input, func(s string) string {
				return sanitizer.Apply(s, sanitizer.Trim, sanitizer.ToLower)
			})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeMapValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name: "sanitizes values",
			input: map[string]string{
				"name":  "  John  ",
				"email": "JOHN@EXAMPLE.COM",
			},
			expected: map[string]string{
				"name":  "john",
				"email": "john@example.com",
			},
		},
		{
			name:     "handles empty map",
			input:    map[string]string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.SanitizeMapValues(tt.input, func(s string) string {
				return sanitizer.Apply(s, sanitizer.Trim, sanitizer.ToLower)
			})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterMapByKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]string
		pattern  string
		expected map[string]string
	}{
		{
			name: "filters by key pattern",
			input: map[string]string{
				"user_name":  "john",
				"user_email": "john@example.com",
				"admin_role": "admin",
			},
			pattern: "user",
			expected: map[string]string{
				"admin_role": "admin",
			},
		},
		{
			name: "case insensitive filtering",
			input: map[string]string{
				"USER_name": "john",
				"admin":     "admin",
			},
			pattern: "user",
			expected: map[string]string{
				"admin": "admin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.FilterMapByKeys(tt.input, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterEmptyMapValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name: "filters empty values",
			input: map[string]string{
				"name":  "john",
				"email": "",
				"phone": "   ",
				"city":  "NYC",
			},
			expected: map[string]string{
				"name": "john",
				"city": "NYC",
			},
		},
		{
			name:     "handles empty map",
			input:    map[string]string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.FilterEmptyMapValues(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanStringMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name: "comprehensive map cleaning",
			input: map[string]string{
				"  NAME  ": "  John Doe  ",
				"EMAIL":    "john@example.com",
				"PHONE":    "",
				"  ":       "empty key",
			},
			expected: map[string]string{
				"name":  "John Doe",
				"email": "john@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.CleanStringMap(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLimitMapSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]string
		maxSize  int
		expected int // expected length
	}{
		{
			name: "limits map size",
			input: map[string]string{
				"a": "1", "b": "2", "c": "3", "d": "4", "e": "5",
			},
			maxSize:  3,
			expected: 3,
		},
		{
			name: "preserves smaller map",
			input: map[string]string{
				"a": "1", "b": "2",
			},
			maxSize:  5,
			expected: 2,
		},
		{
			name:     "zero size returns empty",
			input:    map[string]string{"a": "1"},
			maxSize:  0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.LimitMapSize(tt.input, tt.maxSize)
			assert.Equal(t, tt.expected, len(result))
		})
	}
}

func TestExtractMapKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]string
		expected int // expected length
	}{
		{
			name: "extracts all keys",
			input: map[string]string{
				"a": "1", "b": "2", "c": "3",
			},
			expected: 3,
		},
		{
			name:     "handles empty map",
			input:    map[string]string{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.ExtractMapKeys(tt.input)
			assert.Equal(t, tt.expected, len(result))

			// Verify all keys are present
			for key := range tt.input {
				assert.Contains(t, result, key)
			}
		})
	}
}

func TestMergeStringMaps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		maps     []map[string]string
		expected map[string]string
	}{
		{
			name: "merges multiple maps",
			maps: []map[string]string{
				{"a": "1", "b": "2"},
				{"b": "3", "c": "4"},
				{"c": "5", "d": "6"},
			},
			expected: map[string]string{
				"a": "1", "b": "3", "c": "5", "d": "6",
			},
		},
		{
			name:     "handles empty input",
			maps:     []map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "handles single map",
			maps: []map[string]string{
				{"a": "1", "b": "2"},
			},
			expected: map[string]string{
				"a": "1", "b": "2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.MergeStringMaps(tt.maps...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSliceToMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected map[int]string
	}{
		{
			name:  "converts slice to map",
			input: []string{"a", "b", "c"},
			expected: map[int]string{
				0: "a", 1: "b", 2: "c",
			},
		},
		{
			name:     "handles empty slice",
			input:    []string{},
			expected: map[int]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.SliceToMap(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterSlice(t *testing.T) {
	t.Parallel()

	t.Run("filters with predicate", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 2, 3, 4, 5, 6}
		result := sanitizer.FilterSlice(input, func(n int) bool {
			return n%2 == 0 // even numbers only
		})
		expected := []int{2, 4, 6}
		assert.Equal(t, expected, result)
	})

	t.Run("filters strings by length", func(t *testing.T) {
		t.Parallel()

		input := []string{"a", "hello", "hi", "world"}
		result := sanitizer.FilterSlice(input, func(s string) bool {
			return len(s) > 2
		})
		expected := []string{"hello", "world"}
		assert.Equal(t, expected, result)
	})
}

func TestTransformSlice(t *testing.T) {
	t.Parallel()

	t.Run("transforms int to string", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 2, 3}
		result := sanitizer.TransformSlice(input, func(n int) string {
			switch n {
			case 1:
				return "one"
			case 2:
				return "two"
			default:
				return "three"
			}
		})
		expected := []string{"one", "two", "three"}
		assert.Equal(t, expected, result)
	})

	t.Run("transforms strings to uppercase", func(t *testing.T) {
		t.Parallel()

		input := []string{"hello", "world"}
		result := sanitizer.TransformSlice(input, sanitizer.ToUpper)
		expected := []string{"HELLO", "WORLD"}
		assert.Equal(t, expected, result)
	})
}

func TestReverseSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "reverses slice",
			input:    []string{"a", "b", "c", "d"},
			expected: []string{"d", "c", "b", "a"},
		},
		{
			name:     "handles single element",
			input:    []string{"single"},
			expected: []string{"single"},
		},
		{
			name:     "handles empty slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.ReverseSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCollectionsApplyPattern(t *testing.T) {
	t.Parallel()

	t.Run("apply pattern with collection functions", func(t *testing.T) {
		t.Parallel()

		// Test slice processing pipeline
		messySlice := []string{"  hello  ", "", "WORLD", "hello", "test", "   "}
		result := sanitizer.Apply(messySlice,
			sanitizer.TrimStringSlice,
			sanitizer.FilterEmpty,
			sanitizer.ToLowerStringSlice,
			sanitizer.DeduplicateStrings,
			sanitizer.SortStrings,
		)
		expected := []string{"hello", "test", "world"}
		assert.Equal(t, expected, result)
	})

	t.Run("compose collection transformations", func(t *testing.T) {
		t.Parallel()

		// Create a reusable slice cleaner
		sliceCleaner := sanitizer.Compose(
			sanitizer.TrimStringSlice,
			sanitizer.FilterEmpty,
			sanitizer.DeduplicateStrings,
			func(slice []string) []string {
				return sanitizer.LimitSliceLength(slice, 5)
			},
		)

		input := []string{"  a  ", "b", "", "a", "c", "d", "e", "f", "g"}
		result := sliceCleaner(input)
		expected := []string{"a", "b", "c", "d", "e"}
		assert.Equal(t, expected, result)
	})
}

func TestRealWorldCollectionUsage(t *testing.T) {
	t.Parallel()

	t.Run("user tags processing", func(t *testing.T) {
		t.Parallel()

		// Create tag processor for user-submitted tags
		tagProcessor := sanitizer.Compose(
			sanitizer.TrimStringSlice,
			sanitizer.ToLowerStringSlice,
			sanitizer.FilterEmpty,
			sanitizer.DeduplicateStrings,
			func(slice []string) []string {
				return sanitizer.LimitSliceLength(slice, 10)
			},
			sanitizer.SortStrings,
		)

		userTags := []string{"  JavaScript  ", "golang", "", "PYTHON", "javascript", "React", "   ", "Vue", "Angular", "Node.js", "Docker", "Kubernetes"}
		cleanTags := tagProcessor(userTags)

		expected := []string{"angular", "docker", "golang", "javascript", "kubernetes", "node.js", "python", "react", "vue"}
		assert.Equal(t, expected, cleanTags)
		assert.True(t, len(cleanTags) <= 10)
	})

	t.Run("form data sanitization", func(t *testing.T) {
		t.Parallel()

		// Create form data sanitizer
		formSanitizer := sanitizer.Compose(
			sanitizer.CleanStringMap,
			func(m map[string]string) map[string]string {
				return sanitizer.LimitMapSize(m, 20)
			},
		)

		formData := map[string]string{
			"  NAME  ": "  John Doe  ",
			"EMAIL":    "john@example.com",
			"PHONE":    "",
			"  ":       "should be removed",
			"COMPANY":  "Acme Corp",
		}

		cleanData := formSanitizer(formData)
		expected := map[string]string{
			"name":    "John Doe",
			"email":   "john@example.com",
			"company": "Acme Corp",
		}
		assert.Equal(t, expected, cleanData)
	})

	t.Run("search results deduplication", func(t *testing.T) {
		t.Parallel()

		// Create search results processor
		resultsProcessor := sanitizer.Compose(
			func(slice []string) []string {
				return sanitizer.FilterSlice(slice, func(s string) bool {
					return len(s) > 2 // Filter out very short results
				})
			},
			sanitizer.DeduplicateStringsIgnoreCase,
			func(slice []string) []string {
				return sanitizer.LimitSliceLength(slice, 50)
			},
		)

		searchResults := []string{
			"Go Programming",
			"go programming",
			"Golang Tutorial",
			"Go",
			"GO PROGRAMMING",
			"Python Programming",
			"Java Tutorial",
		}

		cleanResults := resultsProcessor(searchResults)
		expected := []string{"Go Programming", "Golang Tutorial", "Python Programming", "Java Tutorial"}
		assert.Equal(t, expected, cleanResults)
	})

	t.Run("configuration merging", func(t *testing.T) {
		t.Parallel()

		// Merge configuration from multiple sources
		defaultConfig := map[string]string{
			"timeout": "30s",
			"retries": "3",
			"debug":   "false",
		}

		userConfig := map[string]string{
			"timeout": "60s",
			"host":    "localhost",
		}

		envConfig := map[string]string{
			"debug": "true",
			"port":  "8080",
		}

		finalConfig := sanitizer.MergeStringMaps(defaultConfig, userConfig, envConfig)

		assert.Equal(t, "60s", finalConfig["timeout"])    // user override
		assert.Equal(t, "true", finalConfig["debug"])     // env override
		assert.Equal(t, "3", finalConfig["retries"])      // default preserved
		assert.Equal(t, "localhost", finalConfig["host"]) // user addition
		assert.Equal(t, "8080", finalConfig["port"])      // env addition
	})
}
