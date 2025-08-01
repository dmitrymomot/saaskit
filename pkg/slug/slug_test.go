package slug_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/slug"
)

func TestMake(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     []slug.Option
		expected string
	}{
		{
			name:     "simple text",
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "with punctuation",
			input:    "Hello, World!",
			expected: "hello-world",
		},
		{
			name:     "with numbers",
			input:    "Product 123",
			expected: "product-123",
		},
		{
			name:     "multiple spaces",
			input:    "Too    Many     Spaces",
			expected: "too-many-spaces",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  Trim Me  ",
			expected: "trim-me",
		},
		{
			name:     "special characters",
			input:    "Price: $99.99",
			expected: "price-99-99",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only special characters",
			input:    "!@#$%^&*()",
			expected: "",
		},
		{
			name:     "unicode diacritics",
			input:    "Café résumé naïve",
			expected: "cafe-resume-naive",
		},
		{
			name:     "mixed case with lowercase false",
			input:    "Hello World",
			opts:     []slug.Option{slug.Lowercase(false)},
			expected: "Hello-World",
		},
		{
			name:     "custom separator",
			input:    "Hello World",
			opts:     []slug.Option{slug.Separator("_")},
			expected: "hello_world",
		},
		{
			name:     "max length",
			input:    "This is a very long title that should be truncated",
			opts:     []slug.Option{slug.MaxLength(20)},
			expected: "this-is-a-very-long",
		},
		{
			name:     "max length with separator",
			input:    "Cut off cleanly",
			opts:     []slug.Option{slug.MaxLength(7)},
			expected: "cut-off",
		},
		{
			name:     "strip specific characters",
			input:    "Remove (these) [chars]",
			opts:     []slug.Option{slug.StripChars("()[]")},
			expected: "remove-these-chars",
		},
		{
			name:  "custom replacements",
			input: "Fish & Chips @ Home",
			opts: []slug.Option{
				slug.CustomReplace(map[string]string{
					"&": "and",
					"@": "at",
				}),
			},
			expected: "fish-and-chips-at-home",
		},
		{
			name:     "consecutive separators",
			input:    "Too---Many---Dashes",
			expected: "too-many-dashes",
		},
		{
			name:     "german characters",
			input:    "Über Größe straße",
			expected: "uber-grose-strase",
		},
		{
			name:     "french characters",
			input:    "Château façade élève",
			expected: "chateau-facade-eleve",
		},
		{
			name:     "spanish characters",
			input:    "Niño español año",
			expected: "nino-espanol-ano",
		},
		{
			name:     "polish characters",
			input:    "Zażółć gęślą jaźń",
			expected: "zazolc-gesla-jazn",
		},
		{
			name:     "mixed unicode and ascii",
			input:    "Côte d'Ivoire 2024",
			expected: "cote-d-ivoire-2024",
		},
		{
			name:  "all options combined",
			input: "COMPLEX & Test @ 2024!!!",
			opts: []slug.Option{
				slug.Separator("_"),
				slug.Lowercase(false),
				slug.MaxLength(15),
				slug.StripChars("!"),
				slug.CustomReplace(map[string]string{
					"&": "AND",
					"@": "AT",
				}),
			},
			expected: "COMPLEX_AND_Tes",
		},
		{
			name:     "trailing separator should be removed",
			input:    "Ends with dash-",
			expected: "ends-with-dash",
		},
		{
			name:     "multiple trailing separators",
			input:    "Multiple---",
			expected: "multiple",
		},
		{
			name:     "only numbers",
			input:    "123456789",
			expected: "123456789",
		},
		{
			name:     "mixed numbers and letters",
			input:    "abc123def456",
			expected: "abc123def456",
		},
		{
			name:     "url with protocol",
			input:    "https://example.com",
			expected: "https-example-com",
		},
		{
			name:     "email address",
			input:    "user@example.com",
			expected: "user-example-com",
		},
		{
			name:     "path like string",
			input:    "path/to/file.txt",
			expected: "path-to-file-txt",
		},
		{
			name:     "emoji should be stripped",
			input:    "Hello 😀 World 🌍",
			expected: "hello-world",
		},
		{
			name:     "tabs and newlines",
			input:    "Line1\nLine2\tTabbed",
			expected: "line1-line2-tabbed",
		},
		{
			name:     "zero max length",
			input:    "Should not truncate",
			opts:     []slug.Option{slug.MaxLength(0)},
			expected: "should-not-truncate",
		},
		{
			name:     "empty separator",
			input:    "No Separator",
			opts:     []slug.Option{slug.Separator("")},
			expected: "noseparator",
		},
		{
			name:     "multi-character separator",
			input:    "Multi Sep Test",
			opts:     []slug.Option{slug.Separator("---")},
			expected: "multi---sep---test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slug.Make(tt.input, tt.opts...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeDiacritic(t *testing.T) {
	// Test specific diacritic conversions
	inputs := []struct {
		char     string
		expected string
	}{
		{"à", "a"}, {"á", "a"}, {"â", "a"}, {"ã", "a"}, {"ä", "a"}, {"å", "a"},
		{"À", "a"}, {"Á", "a"}, {"Â", "a"}, {"Ã", "a"}, {"Ä", "a"}, {"Å", "a"},
		{"è", "e"}, {"é", "e"}, {"ê", "e"}, {"ë", "e"},
		{"È", "e"}, {"É", "e"}, {"Ê", "e"}, {"Ë", "e"},
		{"ì", "i"}, {"í", "i"}, {"î", "i"}, {"ï", "i"},
		{"Ì", "i"}, {"Í", "i"}, {"Î", "i"}, {"Ï", "i"},
		{"ò", "o"}, {"ó", "o"}, {"ô", "o"}, {"õ", "o"}, {"ö", "o"}, {"ø", "o"},
		{"Ò", "o"}, {"Ó", "o"}, {"Ô", "o"}, {"Õ", "o"}, {"Ö", "o"}, {"Ø", "o"},
		{"ù", "u"}, {"ú", "u"}, {"û", "u"}, {"ü", "u"},
		{"Ù", "u"}, {"Ú", "u"}, {"Û", "u"}, {"Ü", "u"},
		{"ñ", "n"}, {"Ñ", "n"},
		{"ç", "c"}, {"Ç", "c"},
		{"ß", "s"},
		{"æ", "a"}, {"Æ", "a"},
		{"œ", "o"}, {"Œ", "o"},
	}

	for _, tt := range inputs {
		t.Run(tt.char, func(t *testing.T) {
			result := slug.Make(tt.char)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkMake(b *testing.B) {
	testCases := []struct {
		name  string
		input string
		opts  []slug.Option
	}{
		{
			name:  "simple",
			input: "Hello World",
		},
		{
			name:  "with_diacritics",
			input: "Café résumé naïve",
		},
		{
			name:  "long_text",
			input: "This is a very long title that contains many words and should test the performance of the slug generation",
		},
		{
			name:  "with_options",
			input: "Complex & Test @ 2024",
			opts: []slug.Option{
				slug.MaxLength(20),
				slug.CustomReplace(map[string]string{"&": "and", "@": "at"}),
			},
		},
		{
			name:  "unicode_heavy",
			input: "Ñoño español año château façade über größe",
		},
		{
			name:  "special_chars_heavy",
			input: "!@#$%^&*()_+{}|:\"<>?[]\\;',./",
		},
		{
			name:  "with_suffix",
			input: "Product Name",
			opts:  []slug.Option{slug.WithSuffix(6)},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = slug.Make(tc.input, tc.opts...)
			}
		})
	}
}

func BenchmarkMakeParallel(b *testing.B) {
	input := "This is a sample text with some special characters: !@#$%"
	opts := []slug.Option{slug.MaxLength(50)}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = slug.Make(input, opts...)
		}
	})
}

func TestGenerateSuffixErrorHandling(t *testing.T) {
	// This test is designed to improve coverage by testing the error path
	// In real usage, rand.Read rarely fails, but we need to test the fallback

	// Test that generateSuffix produces valid output even in edge cases
	tests := []struct {
		name      string
		length    int
		lowercase bool
	}{
		{
			name:      "zero length",
			length:    0,
			lowercase: true,
		},
		{
			name:      "small length lowercase",
			length:    1,
			lowercase: true,
		},
		{
			name:      "small length uppercase allowed",
			length:    1,
			lowercase: false,
		},
		{
			name:      "large length",
			length:    100,
			lowercase: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since we can't easily mock rand.Read failure, we'll test that
			// the function always produces valid output
			for range 10 {
				result := slug.Make("test", slug.WithSuffix(tt.length))
				if tt.length > 0 {
					parts := strings.Split(result, "-")
					suffix := parts[len(parts)-1]
					assert.Len(t, suffix, tt.length)
					if tt.lowercase {
						assert.Regexp(t, "^[a-z0-9]*$", suffix)
					} else {
						assert.Regexp(t, "^[a-zA-Z0-9]*$", suffix)
					}
				}
			}
		})
	}
}

func TestMaxLengthEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     []slug.Option
		validate func(t *testing.T, result string)
	}{
		{
			name:  "max length smaller than suffix",
			input: "Test",
			opts:  []slug.Option{slug.WithSuffix(10), slug.MaxLength(5)},
			validate: func(t *testing.T, result string) {
				assert.LessOrEqual(t, len(result), 5)
				// Should be just truncated suffix
				assert.Regexp(t, "^[a-z0-9]{5}$", result)
			},
		},
		{
			name:  "max length exactly suffix length",
			input: "Test",
			opts:  []slug.Option{slug.WithSuffix(8), slug.MaxLength(8)},
			validate: func(t *testing.T, result string) {
				assert.Equal(t, 8, len(result))
				assert.Regexp(t, "^[a-z0-9]{8}$", result)
			},
		},
		{
			name:  "max length with multi-byte separator",
			input: "Test Case",
			opts:  []slug.Option{slug.WithSuffix(4), slug.MaxLength(15), slug.Separator("---")},
			validate: func(t *testing.T, result string) {
				assert.LessOrEqual(t, len(result), 15)
				assert.Contains(t, result, "---")
			},
		},
		{
			name:  "very small max length with suffix",
			input: "Long Title Here",
			opts:  []slug.Option{slug.WithSuffix(3), slug.MaxLength(5)},
			validate: func(t *testing.T, result string) {
				assert.LessOrEqual(t, len(result), 5)
				// Should be "l-abc" or similar
				parts := strings.Split(result, "-")
				if len(parts) > 1 {
					assert.Len(t, parts[len(parts)-1], 3)
				}
			},
		},
		{
			name:  "max length cuts in middle of rune",
			input: "Test™Case", // ™ is multi-byte
			opts:  []slug.Option{slug.MaxLength(6)},
			validate: func(t *testing.T, result string) {
				// "Test™Case" becomes "test-case" but truncated to 6 chars = "test-c"
				assert.Equal(t, "test-c", result)
			},
		},
		{
			name:  "empty input with suffix and max length",
			input: "",
			opts:  []slug.Option{slug.WithSuffix(10), slug.MaxLength(5)},
			validate: func(t *testing.T, result string) {
				assert.Equal(t, 5, len(result))
				assert.Regexp(t, "^[a-z0-9]{5}$", result)
			},
		},
		{
			name:  "suffix with no room after max length truncation",
			input: "VeryLongTitleThatNeedsToBeShortened",
			opts:  []slug.Option{slug.WithSuffix(6), slug.MaxLength(8)},
			validate: func(t *testing.T, result string) {
				// Should be "v-abc123" (1 char + separator + 6 char suffix = 8)
				assert.Equal(t, 8, len(result))
				parts := strings.Split(result, "-")
				assert.Len(t, parts[len(parts)-1], 6)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slug.Make(tt.input, tt.opts...)
			tt.validate(t, result)
		})
	}
}

func TestMakeWithSuffix(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		opts      []slug.Option
		checkFunc func(t *testing.T, result string)
	}{
		{
			name:  "basic suffix",
			input: "Hello World",
			opts:  []slug.Option{slug.WithSuffix(6)},
			checkFunc: func(t *testing.T, result string) {
				parts := strings.Split(result, "-")
				assert.Equal(t, "hello", parts[0])
				assert.Equal(t, "world", parts[1])
				assert.Len(t, parts[2], 6) // suffix should be 6 chars
				// Check suffix is alphanumeric lowercase
				assert.Regexp(t, "^[a-z0-9]{6}$", parts[2])
			},
		},
		{
			name:  "suffix with uppercase disabled",
			input: "Test",
			opts:  []slug.Option{slug.WithSuffix(8), slug.Lowercase(false)},
			checkFunc: func(t *testing.T, result string) {
				parts := strings.Split(result, "-")
				assert.Equal(t, "Test", parts[0])
				assert.Len(t, parts[1], 8)
				// Check suffix can contain uppercase
				assert.Regexp(t, "^[a-zA-Z0-9]{8}$", parts[1])
			},
		},
		{
			name:  "suffix with custom separator",
			input: "Product",
			opts:  []slug.Option{slug.WithSuffix(4), slug.Separator("_")},
			checkFunc: func(t *testing.T, result string) {
				parts := strings.Split(result, "_")
				assert.Equal(t, "product", parts[0])
				assert.Len(t, parts[1], 4)
			},
		},
		{
			name:  "suffix with max length",
			input: "Very Long Title Here",
			opts:  []slug.Option{slug.WithSuffix(6), slug.MaxLength(20)},
			checkFunc: func(t *testing.T, result string) {
				assert.LessOrEqual(t, len(result), 20)
				assert.Contains(t, result, "-") // Should have separator
				parts := strings.Split(result, "-")
				lastPart := parts[len(parts)-1]
				assert.Len(t, lastPart, 6) // suffix should still be 6 chars
			},
		},
		{
			name:  "suffix longer than max length",
			input: "Test",
			opts:  []slug.Option{slug.WithSuffix(10), slug.MaxLength(8)},
			checkFunc: func(t *testing.T, result string) {
				// Should just be the suffix truncated
				assert.Len(t, result, 8)
				assert.Regexp(t, "^[a-z0-9]{8}$", result)
			},
		},
		{
			name:  "empty input with suffix",
			input: "",
			opts:  []slug.Option{slug.WithSuffix(5)},
			checkFunc: func(t *testing.T, result string) {
				assert.Len(t, result, 5)
				assert.Regexp(t, "^[a-z0-9]{5}$", result)
			},
		},
		{
			name:  "zero length suffix",
			input: "Normal Slug",
			opts:  []slug.Option{slug.WithSuffix(0)},
			checkFunc: func(t *testing.T, result string) {
				assert.Equal(t, "normal-slug", result)
			},
		},
		{
			name:  "suffix preserves uniqueness",
			input: "Same Title",
			opts:  []slug.Option{slug.WithSuffix(6)},
			checkFunc: func(t *testing.T, result string) {
				// Generate another one and check they're different
				result2 := slug.Make("Same Title", slug.WithSuffix(6))
				assert.NotEqual(t, result, result2)
				// But the base should be the same
				parts1 := strings.Split(result, "-")
				parts2 := strings.Split(result2, "-")
				assert.Equal(t, parts1[0], parts2[0])
				assert.Equal(t, parts1[1], parts2[1])
				assert.NotEqual(t, parts1[2], parts2[2]) // suffixes should differ
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slug.Make(tt.input, tt.opts...)
			tt.checkFunc(t, result)
		})
	}
}
