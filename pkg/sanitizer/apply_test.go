package sanitizer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

func TestApply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		transforms []func(string) string
		expected   string
	}{
		{
			name:       "applies single transform",
			input:      "  hello  ",
			transforms: []func(string) string{sanitizer.Trim},
			expected:   "hello",
		},
		{
			name:  "applies multiple transforms in sequence",
			input: "  HELLO WORLD  ",
			transforms: []func(string) string{
				sanitizer.Trim,
				sanitizer.ToLower,
			},
			expected: "hello world",
		},
		{
			name:  "applies complex transformation chain",
			input: "  Hello    World!@#  ",
			transforms: []func(string) string{
				sanitizer.Trim,
				sanitizer.RemoveExtraWhitespace,
				sanitizer.ToLower,
				func(s string) string { return sanitizer.MaxLength(s, 10) },
			},
			expected: "hello worl",
		},
		{
			name:       "handles empty transforms slice",
			input:      "hello world",
			transforms: []func(string) string{},
			expected:   "hello world",
		},
		{
			name:  "handles empty input",
			input: "",
			transforms: []func(string) string{
				sanitizer.Trim,
				sanitizer.ToLower,
			},
			expected: "",
		},
		{
			name:  "applies all string functions",
			input: "  HELLO    WORLD!@#123  ",
			transforms: []func(string) string{
				sanitizer.Trim,
				sanitizer.RemoveExtraWhitespace,
				sanitizer.KeepAlphanumeric,
				sanitizer.ToLower,
			},
			expected: "hello world123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.Apply(tt.input, tt.transforms...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompose(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		transforms []func(string) string
		input      string
		expected   string
	}{
		{
			name:       "composes single transform",
			transforms: []func(string) string{sanitizer.Trim},
			input:      "  hello  ",
			expected:   "hello",
		},
		{
			name: "composes multiple transforms",
			transforms: []func(string) string{
				sanitizer.Trim,
				sanitizer.ToLower,
				func(s string) string { return sanitizer.MaxLength(s, 10) },
			},
			input:    "  TEST@EXAMPLE.COM  ",
			expected: "test@examp",
		},
		{
			name:       "handles empty transforms",
			transforms: []func(string) string{},
			input:      "hello",
			expected:   "hello",
		},
		{
			name: "creates reusable transformation",
			transforms: []func(string) string{
				sanitizer.Trim,
				sanitizer.RemoveExtraWhitespace,
				sanitizer.ToLower,
			},
			input:    "  HELLO    WORLD  ",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			composedRule := sanitizer.Compose(tt.transforms...)
			result := composedRule(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComposeReusability(t *testing.T) {
	t.Parallel()

	t.Run("composed rule can be reused multiple times", func(t *testing.T) {
		t.Parallel()

		// Create a reusable email cleaning rule
		emailCleanRule := sanitizer.Compose(
			sanitizer.Trim,
			sanitizer.ToLower,
			func(s string) string { return sanitizer.MaxLength(s, 50) },
		)

		// Test with different inputs
		inputs := []string{
			"  USER@EXAMPLE.COM  ",
			"ADMIN@COMPANY.ORG",
			"  test.email+tag@very-long-domain-name.example.org  ",
		}

		expected := []string{
			"user@example.com",
			"admin@company.org",
			"test.email+tag@very-long-domain-name.example.org",
		}

		for i, input := range inputs {
			result := emailCleanRule(input)
			assert.Equal(t, expected[i], result, "Failed for input: %s", input)
		}
	})
}

func TestApplyWithCompose(t *testing.T) {
	t.Parallel()

	t.Run("apply can use composed rules", func(t *testing.T) {
		t.Parallel()

		// Create a composed rule
		nameCleanRule := sanitizer.Compose(
			sanitizer.Trim,
			sanitizer.RemoveExtraWhitespace,
			sanitizer.ToTitle,
		)

		// Use it in Apply
		result := sanitizer.Apply("  john    doe  ", nameCleanRule)
		assert.Equal(t, "JOHN DOE", result)
	})

	t.Run("mix composed rules with direct functions", func(t *testing.T) {
		t.Parallel()

		// Create a partial composed rule
		basicCleanRule := sanitizer.Compose(
			sanitizer.Trim,
			sanitizer.RemoveExtraWhitespace,
		)

		// Use it with additional direct functions
		result := sanitizer.Apply("  HELLO    WORLD  ",
			basicCleanRule,
			sanitizer.ToLower,
			func(s string) string { return sanitizer.MaxLength(s, 8) },
		)
		assert.Equal(t, "hello wo", result)
	})
}

func TestRealWorldUsagePatternsUsage(t *testing.T) {
	t.Parallel()

	t.Run("user registration data sanitization", func(t *testing.T) {
		t.Parallel()

		// Create reusable rules for different field types
		nameCleanRule := sanitizer.Compose(
			sanitizer.Trim,
			sanitizer.RemoveExtraWhitespace,
			sanitizer.ToTitle,
		)

		emailCleanRule := sanitizer.Compose(
			sanitizer.Trim,
			sanitizer.ToLower,
		)

		usernameCleanRule := sanitizer.Compose(
			sanitizer.KeepAlphanumeric,
			sanitizer.ToLower,
			func(s string) string { return sanitizer.MaxLength(s, 20) },
		)

		// Simulate cleaning user registration data
		rawName := "  john    DOE  "
		rawEmail := "  JOHN.DOE@EXAMPLE.COM  "
		rawUsername := "john_doe_123!@#"

		cleanName := nameCleanRule(rawName)
		cleanEmail := emailCleanRule(rawEmail)
		cleanUsername := usernameCleanRule(rawUsername)

		assert.Equal(t, "JOHN DOE", cleanName)
		assert.Equal(t, "john.doe@example.com", cleanEmail)
		assert.Equal(t, "johndoe123", cleanUsername)
	})

	t.Run("content sanitization pipeline", func(t *testing.T) {
		t.Parallel()

		// Create a complex content cleaning pipeline
		contentCleanRule := sanitizer.Compose(
			sanitizer.StripHTML,
			sanitizer.RemoveControlChars,
			sanitizer.RemoveExtraWhitespace,
			func(s string) string { return sanitizer.MaxLength(s, 200) },
		)

		dirtyContent := "<script>alert('xss')</script><p>Hello    \x00world</p>\n\nThis is a test."
		cleanContent := contentCleanRule(dirtyContent)

		expected := "alert('xss')Hello world This is a test."
		assert.Equal(t, expected, cleanContent)
	})
}

func TestEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("apply with no transforms preserves value", func(t *testing.T) {
		t.Parallel()

		input := "test value"
		result := sanitizer.Apply(input)
		assert.Equal(t, input, result)
	})

	t.Run("compose with no transforms creates identity function", func(t *testing.T) {
		t.Parallel()

		identityRule := sanitizer.Compose[string]()
		input := "test value"
		result := identityRule(input)
		assert.Equal(t, input, result)
	})

	t.Run("chained compositions work correctly", func(t *testing.T) {
		t.Parallel()

		rule1 := sanitizer.Compose(sanitizer.Trim)
		rule2 := sanitizer.Compose(sanitizer.ToLower)
		combinedRule := sanitizer.Compose(rule1, rule2)

		result := combinedRule("  HELLO  ")
		assert.Equal(t, "hello", result)
	})
}
