package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestMatchesRegex(t *testing.T) {
	t.Run("valid regex matches", func(t *testing.T) {
		testCases := []struct {
			value       string
			pattern     string
			description string
		}{
			{"abc123", `^[a-z]+\d+$`, "lowercase letters followed by digits"},
			{"test@example.com", `^[^@]+@[^@]+\.[^@]+$`, "email format"},
			{"123-456-7890", `^\d{3}-\d{3}-\d{4}$`, "phone number"},
			{"ABC", `^[A-Z]+$`, "uppercase letters"},
		}

		for _, tc := range testCases {
			rule := validator.MatchesRegex("field", tc.value, tc.pattern, tc.description)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should match pattern: %s", tc.value)
		}
	})

	t.Run("invalid regex matches", func(t *testing.T) {
		testCases := []struct {
			value       string
			pattern     string
			description string
		}{
			{"", `^[a-z]+\d+$`, "lowercase letters followed by digits"},
			{"   ", `^[a-z]+\d+$`, "lowercase letters followed by digits"},
			{"ABC123", `^[a-z]+\d+$`, "lowercase letters followed by digits"},
			{"abc", `^[a-z]+\d+$`, "lowercase letters followed by digits"},
			{"123", `^[a-z]+\d+$`, "lowercase letters followed by digits"},
			{"invalid-email", `^[^@]+@[^@]+\.[^@]+$`, "email format"},
		}

		for _, tc := range testCases {
			rule := validator.MatchesRegex("field", tc.value, tc.pattern, tc.description)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should not match pattern: %s", tc.value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.regex_pattern", validationErr[0].TranslationKey)
		}
	})
}

func TestDoesNotMatchRegex(t *testing.T) {
	t.Run("valid non-matches", func(t *testing.T) {
		testCases := []struct {
			value       string
			pattern     string
			description string
		}{
			{"ABC123", `^[a-z]+\d+$`, "lowercase letters followed by digits"},
			{"abc", `^[a-z]+\d+$`, "lowercase letters followed by digits"},
			{"invalid-email", `^[^@]+@[^@]+\.[^@]+$`, "email format"},
			{"", `^[a-z]+\d+$`, "lowercase letters followed by digits"},
		}

		for _, tc := range testCases {
			rule := validator.DoesNotMatchRegex("field", tc.value, tc.pattern, tc.description)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should not match pattern: %s", tc.value)
		}
	})

	t.Run("invalid non-matches (actually matches)", func(t *testing.T) {
		testCases := []struct {
			value       string
			pattern     string
			description string
		}{
			{"abc123", `^[a-z]+\d+$`, "lowercase letters followed by digits"},
			{"test@example.com", `^[^@]+@[^@]+\.[^@]+$`, "email format"},
		}

		for _, tc := range testCases {
			rule := validator.DoesNotMatchRegex("field", tc.value, tc.pattern, tc.description)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected because it matches pattern: %s", tc.value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.regex_not_pattern", validationErr[0].TranslationKey)
		}
	})
}

func TestContainsPattern(t *testing.T) {
	t.Run("valid pattern contains", func(t *testing.T) {
		testCases := []struct {
			value       string
			pattern     string
			description string
		}{
			{"hello world 123", `\d+`, "numbers"},
			{"test@example.com", `@`, "at symbol"},
			{"abc123def", `\d`, "digit"},
			{"UPPERCASE", `[A-Z]`, "uppercase letter"},
		}

		for _, tc := range testCases {
			rule := validator.ContainsPattern("field", tc.value, tc.pattern, tc.description)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should contain pattern: %s", tc.value)
		}
	})

	t.Run("invalid pattern contains", func(t *testing.T) {
		testCases := []struct {
			value       string
			pattern     string
			description string
		}{
			{"hello world", `\d+`, "numbers"},
			{"testexample.com", `@`, "at symbol"},
			{"abcdef", `\d`, "digit"},
			{"lowercase", `[A-Z]`, "uppercase letter"},
		}

		for _, tc := range testCases {
			rule := validator.ContainsPattern("field", tc.value, tc.pattern, tc.description)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should not contain pattern: %s", tc.value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.contains_pattern", validationErr[0].TranslationKey)
		}
	})
}

func TestStartsWithPattern(t *testing.T) {
	t.Run("valid starts with pattern", func(t *testing.T) {
		testCases := []struct {
			value   string
			pattern string
		}{
			{"hello world", "hello"},
			{"123abc", `\d+`},
			{"ABC123", "[A-Z]+"},
			{"test@example.com", "test"},
		}

		for _, tc := range testCases {
			rule := validator.StartsWithPattern("field", tc.value, tc.pattern)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should start with pattern: %s", tc.value)
		}
	})

	t.Run("invalid starts with pattern", func(t *testing.T) {
		testCases := []struct {
			value   string
			pattern string
		}{
			{"world hello", "hello"},
			{"abc123", `\d+`},
			{"123ABC", "[A-Z]+"},
			{"@test.com", "test"},
		}

		for _, tc := range testCases {
			rule := validator.StartsWithPattern("field", tc.value, tc.pattern)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should not start with pattern: %s", tc.value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.starts_with_pattern", validationErr[0].TranslationKey)
		}
	})
}

func TestEndsWithPattern(t *testing.T) {
	t.Run("valid ends with pattern", func(t *testing.T) {
		testCases := []struct {
			value   string
			pattern string
		}{
			{"hello world", "world"},
			{"abc123", `\d+`},
			{"test123ABC", "[A-Z]+"},
			{"test@example.com", "\\.com"},
		}

		for _, tc := range testCases {
			rule := validator.EndsWithPattern("field", tc.value, tc.pattern)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should end with pattern: %s", tc.value)
		}
	})

	t.Run("invalid ends with pattern", func(t *testing.T) {
		testCases := []struct {
			value   string
			pattern string
		}{
			{"world hello", "world"},
			{"123abc", `\d+`},
			{"ABC123", "[A-Z]+"},
			{"test@example.org", "\\.com"},
		}

		for _, tc := range testCases {
			rule := validator.EndsWithPattern("field", tc.value, tc.pattern)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should not end with pattern: %s", tc.value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.ends_with_pattern", validationErr[0].TranslationKey)
		}
	})
}

func TestNoWhitespace(t *testing.T) {
	t.Run("valid no whitespace", func(t *testing.T) {
		validValues := []string{
			"helloworld",
			"abc123",
			"test@example.com",
			"no_spaces_here",
			"",
		}

		for _, value := range validValues {
			rule := validator.NoWhitespace("field", value)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should have no whitespace: %s", value)
		}
	})

	t.Run("invalid no whitespace", func(t *testing.T) {
		invalidValues := []string{
			"hello world",
			"test\tvalue",
			"new\nline",
			"carriage\rreturn",
			"   ",
			" ",
		}

		for _, value := range invalidValues {
			rule := validator.NoWhitespace("field", value)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected for whitespace: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.no_whitespace", validationErr[0].TranslationKey)
		}
	})
}

func TestOnlyWhitespace(t *testing.T) {
	t.Run("valid only whitespace", func(t *testing.T) {
		validValues := []string{
			"   ",
			" ",
			"\t",
			"\n",
			"\r",
			"\t\n\r ",
		}

		for _, value := range validValues {
			rule := validator.OnlyWhitespace("field", value)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should be only whitespace: %s", value)
		}
	})

	t.Run("invalid only whitespace", func(t *testing.T) {
		invalidValues := []string{
			"",
			"hello world",
			"a ",
			" b",
			"test\tvalue",
			"new\nline text",
		}

		for _, value := range invalidValues {
			rule := validator.OnlyWhitespace("field", value)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.only_whitespace", validationErr[0].TranslationKey)
		}
	})
}

func TestNoControlChars(t *testing.T) {
	t.Run("valid no control chars", func(t *testing.T) {
		validValues := []string{
			"hello world",
			"test@example.com",
			"abc123",
			"Normal text with spaces",
			"Text with\ttab",      // tab is allowed
			"Text with\nnewline",  // newline is allowed
			"Text with\rcarriage", // carriage return is allowed
			"",
		}

		for _, value := range validValues {
			rule := validator.NoControlChars("field", value)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should have no control chars: %s", value)
		}
	})

	t.Run("invalid no control chars", func(t *testing.T) {
		invalidValues := []string{
			"text\x00with null",    // null character
			"text\x01with control", // control character
			"text\x1Fwith control", // control character
		}

		for _, value := range invalidValues {
			rule := validator.NoControlChars("field", value)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected for control chars: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.no_control_chars", validationErr[0].TranslationKey)
		}
	})
}

func TestPrintableChars(t *testing.T) {
	t.Run("valid printable chars", func(t *testing.T) {
		validValues := []string{
			"hello world",
			"test@example.com",
			"abc123!@#$%^&*()",
			"Text with spaces and symbols",
			"",
		}

		for _, value := range validValues {
			rule := validator.PrintableChars("field", value)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should have only printable chars: %s", value)
		}
	})

	t.Run("invalid printable chars", func(t *testing.T) {
		invalidValues := []string{
			"text\x00with null",    // null character
			"text\x01with control", // control character
			"text\x7Fwith del",     // DEL character
		}

		for _, value := range invalidValues {
			rule := validator.PrintableChars("field", value)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected for non-printable chars: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.printable_chars", validationErr[0].TranslationKey)
		}
	})
}

func TestASCIIOnly(t *testing.T) {
	t.Run("valid ASCII only", func(t *testing.T) {
		validValues := []string{
			"hello world",
			"test@example.com",
			"abc123!@#$%^&*()",
			"Text with spaces and symbols",
			"",
		}

		for _, value := range validValues {
			rule := validator.ASCIIOnly("field", value)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should have only ASCII chars: %s", value)
		}
	})

	t.Run("invalid ASCII only", func(t *testing.T) {
		invalidValues := []string{
			"héllo world",      // accented character
			"test@exämple.com", // accented character
			"中文",               // Chinese characters
			"🚀",                // emoji
			"café",             // accented character
		}

		for _, value := range invalidValues {
			rule := validator.ASCIIOnly("field", value)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected for non-ASCII chars: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.ascii_only", validationErr[0].TranslationKey)
		}
	})
}

func TestNoSpecialChars(t *testing.T) {
	t.Run("valid no special chars", func(t *testing.T) {
		validValues := []string{
			"hello world",
			"abc123",
			"Text with spaces",
			"123456789",
			"ABCDEFG",
			"",
		}

		for _, value := range validValues {
			rule := validator.NoSpecialChars("field", value)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should have no special chars: %s", value)
		}
	})

	t.Run("invalid no special chars", func(t *testing.T) {
		invalidValues := []string{
			"test@example.com",
			"hello!",
			"text-with-hyphens",
			"text_with_underscores",
			"text.with.dots",
			"text#with#hash",
		}

		for _, value := range invalidValues {
			rule := validator.NoSpecialChars("field", value)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected for special chars: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.no_special_chars", validationErr[0].TranslationKey)
		}
	})
}

func TestContainsUppercase(t *testing.T) {
	t.Run("valid contains uppercase", func(t *testing.T) {
		validValues := []string{
			"Hello world",
			"TEST",
			"mixedCase",
			"A",
			"test123ABC",
		}

		for _, value := range validValues {
			rule := validator.ContainsUppercase("field", value)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should contain uppercase: %s", value)
		}
	})

	t.Run("invalid contains uppercase", func(t *testing.T) {
		invalidValues := []string{
			"",
			"hello world",
			"test123",
			"all lowercase",
			"123456",
		}

		for _, value := range invalidValues {
			rule := validator.ContainsUppercase("field", value)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected for no uppercase: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.contains_uppercase", validationErr[0].TranslationKey)
		}
	})
}

func TestContainsLowercase(t *testing.T) {
	t.Run("valid contains lowercase", func(t *testing.T) {
		validValues := []string{
			"Hello world",
			"test",
			"mixedCase",
			"a",
			"TEST123abc",
		}

		for _, value := range validValues {
			rule := validator.ContainsLowercase("field", value)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should contain lowercase: %s", value)
		}
	})

	t.Run("invalid contains lowercase", func(t *testing.T) {
		invalidValues := []string{
			"",
			"HELLO WORLD",
			"TEST123",
			"ALL UPPERCASE",
			"123456",
		}

		for _, value := range invalidValues {
			rule := validator.ContainsLowercase("field", value)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected for no lowercase: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.contains_lowercase", validationErr[0].TranslationKey)
		}
	})
}

func TestContainsDigit(t *testing.T) {
	t.Run("valid contains digit", func(t *testing.T) {
		validValues := []string{
			"hello1world",
			"test123",
			"abc9def",
			"1",
			"PASSWORD123",
		}

		for _, value := range validValues {
			rule := validator.ContainsDigit("field", value)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should contain digit: %s", value)
		}
	})

	t.Run("invalid contains digit", func(t *testing.T) {
		invalidValues := []string{
			"",
			"hello world",
			"TEST",
			"all letters",
			"!@#$%^&*()",
		}

		for _, value := range invalidValues {
			rule := validator.ContainsDigit("field", value)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected for no digit: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.contains_digit", validationErr[0].TranslationKey)
		}
	})
}

func TestBalancedParentheses(t *testing.T) {
	t.Run("valid balanced parentheses", func(t *testing.T) {
		validValues := []string{
			"",
			"hello world",
			"()",
			"(hello)",
			"(hello) (world)",
			"((nested))",
			"text (with) balanced (parentheses)",
			"(((())))",
		}

		for _, value := range validValues {
			rule := validator.BalancedParentheses("field", value)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should have balanced parentheses: %s", value)
		}
	})

	t.Run("invalid balanced parentheses", func(t *testing.T) {
		invalidValues := []string{
			"(",
			")",
			"(hello",
			"world)",
			"(hello))",
			"((hello)",
			")hello(",
			"(()",
			"())",
		}

		for _, value := range invalidValues {
			rule := validator.BalancedParentheses("field", value)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected for unbalanced parentheses: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.balanced_parentheses", validationErr[0].TranslationKey)
		}
	})
}

func TestWordCount(t *testing.T) {
	t.Run("valid word count", func(t *testing.T) {
		testCases := []struct {
			value string
			min   int
			max   int
		}{
			{"hello world", 2, 2},
			{"one two three", 1, 5},
			{"single", 1, 1},
			{"", 0, 0},
			{"one two three four five", 3, 10},
		}

		for _, tc := range testCases {
			rule := validator.WordCount("field", tc.value, tc.min, tc.max)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should have valid word count: %s", tc.value)
		}
	})

	t.Run("invalid word count", func(t *testing.T) {
		testCases := []struct {
			value string
			min   int
			max   int
		}{
			{"hello", 2, 5},                        // too few words
			{"hello world test extra words", 1, 3}, // too many words
			{"", 1, 5},                             // empty string, min > 0
			{"single word", 3, 5},                  // too few words
		}

		for _, tc := range testCases {
			rule := validator.WordCount("field", tc.value, tc.min, tc.max)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected for word count: %s", tc.value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.word_count", validationErr[0].TranslationKey)
		}
	})
}

func TestLineCount(t *testing.T) {
	t.Run("valid line count", func(t *testing.T) {
		testCases := []struct {
			value string
			min   int
			max   int
		}{
			{"hello\nworld", 2, 2},
			{"single line", 1, 1},
			{"line1\nline2\nline3", 1, 5},
			{"", 1, 1}, // empty string is 1 line
			{"line1\nline2\nline3\nline4\nline5", 3, 10},
		}

		for _, tc := range testCases {
			rule := validator.LineCount("field", tc.value, tc.min, tc.max)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should have valid line count: %s", tc.value)
		}
	})

	t.Run("invalid line count", func(t *testing.T) {
		testCases := []struct {
			value string
			min   int
			max   int
		}{
			{"single line", 2, 5},                       // too few lines
			{"line1\nline2\nline3\nline4\nline5", 1, 3}, // too many lines
			{"", 2, 5},             // empty string, min > 1
			{"line1\nline2", 3, 5}, // too few lines
		}

		for _, tc := range testCases {
			rule := validator.LineCount("field", tc.value, tc.min, tc.max)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected for line count: %s", tc.value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.line_count", validationErr[0].TranslationKey)
		}
	})
}

func TestPatternValidationCombination(t *testing.T) {
	t.Run("comprehensive pattern validation", func(t *testing.T) {
		value := "Hello123World"

		err := validator.Apply(
			validator.NoWhitespace("field", value),
			validator.ContainsUppercase("field", value),
			validator.ContainsLowercase("field", value),
			validator.ContainsDigit("field", value),
			validator.ASCIIOnly("field", value),
			validator.PrintableChars("field", value),
		)

		assert.NoError(t, err, "Valid pattern data should pass all validations")
	})

	t.Run("invalid pattern data fails multiple validations", func(t *testing.T) {
		value := "hello world" // has whitespace, no uppercase, no digit

		err := validator.Apply(
			validator.NoWhitespace("field", value),
			validator.ContainsUppercase("field", value),
			validator.ContainsDigit("field", value),
		)

		assert.Error(t, err, "Invalid pattern data should fail validations")

		validationErr := validator.ExtractValidationErrors(err)
		require.NotNil(t, validationErr)
		assert.True(t, len(validationErr) > 1, "Should have multiple validation errors")
	})
}

func TestValidOTP(t *testing.T) {
	t.Run("valid OTP codes", func(t *testing.T) {
		testCases := []struct {
			name   string
			value  string
			length int
		}{
			{"6-digit OTP with different digits", "123456", 6},
			{"6-digit OTP with all zeros", "000000", 6},
			{"6-digit OTP with all nines", "999999", 6},
			{"6-digit OTP with leading zeros", "001234", 6},
			{"4-digit PIN", "1234", 4},
			{"8-digit security code", "12345678", 8},
			{"single digit", "5", 1},
			{"10-digit code", "1234567890", 10},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := validator.ValidOTP("code", tc.value, tc.length)
				err := validator.Apply(rule)
				assert.NoError(t, err, "Valid OTP should pass validation: %s", tc.value)
			})
		}
	})

	t.Run("invalid OTP codes - wrong length", func(t *testing.T) {
		testCases := []struct {
			name   string
			value  string
			length int
		}{
			{"too short - 5 digits for 6-digit OTP", "12345", 6},
			{"too long - 7 digits for 6-digit OTP", "1234567", 6},
			{"empty string for 6-digit OTP", "", 6},
			{"too short - 3 digits for 4-digit PIN", "123", 4},
			{"too long - 5 digits for 4-digit PIN", "12345", 4},
			{"single digit for 6-digit OTP", "1", 6},
			{"way too long", "123456789012345", 6},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := validator.ValidOTP("code", tc.value, tc.length)
				err := validator.Apply(rule)
				assert.Error(t, err, "Invalid length OTP should fail validation: %s (length %d, expected %d)", tc.value, len(tc.value), tc.length)

				validationErr := validator.ExtractValidationErrors(err)
				require.NotNil(t, validationErr)
				assert.Len(t, validationErr, 1)
				assert.Equal(t, "code", validationErr[0].Field)
				assert.Equal(t, "validation.otp_code", validationErr[0].TranslationKey)
				assert.Equal(t, tc.length, validationErr[0].TranslationValues["length"])
			})
		}
	})

	t.Run("invalid OTP codes - non-numeric characters", func(t *testing.T) {
		testCases := []struct {
			name   string
			value  string
			length int
		}{
			{"letters only", "abcdef", 6},
			{"mixed alphanumeric", "12a456", 6},
			{"with spaces", "12 456", 6},
			{"with special characters", "123!56", 6},
			{"with dashes", "123-56", 6},
			{"with dots", "123.56", 6},
			{"uppercase letters", "ABC123", 6},
			{"lowercase letters", "abc123", 6},
			{"symbols only", "!@#$%^", 6},
			{"with newline", "12345\n", 6},
			{"with tab", "12345\t", 6},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := validator.ValidOTP("code", tc.value, tc.length)
				err := validator.Apply(rule)
				assert.Error(t, err, "Non-numeric OTP should fail validation: %s", tc.value)

				validationErr := validator.ExtractValidationErrors(err)
				require.NotNil(t, validationErr)
				assert.Len(t, validationErr, 1)
				assert.Equal(t, "code", validationErr[0].Field)
				assert.Equal(t, "validation.otp_code", validationErr[0].TranslationKey)
			})
		}
	})

	t.Run("invalid length parameter", func(t *testing.T) {
		testCases := []struct {
			name   string
			value  string
			length int
		}{
			{"zero length", "123456", 0},
			{"negative length", "123456", -1},
			{"very negative length", "123456", -100},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := validator.ValidOTP("code", tc.value, tc.length)
				err := validator.Apply(rule)
				assert.Error(t, err, "Invalid length parameter should fail validation")

				validationErr := validator.ExtractValidationErrors(err)
				require.NotNil(t, validationErr)
				assert.Len(t, validationErr, 1)
				assert.Equal(t, "code", validationErr[0].Field)
				assert.Equal(t, "validation.otp_code", validationErr[0].TranslationKey)
			})
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("unicode digits should fail", func(t *testing.T) {
			// Unicode digits that are not ASCII 0-9
			rule := validator.ValidOTP("code", "①②③④⑤⑥", 6)
			err := validator.Apply(rule)
			assert.Error(t, err, "Unicode digits should fail validation")
		})

		t.Run("long valid OTP", func(t *testing.T) {
			longOTP := "12345678901234567890"
			rule := validator.ValidOTP("code", longOTP, 20)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Long valid OTP should pass")
		})

		t.Run("translation values", func(t *testing.T) {
			rule := validator.ValidOTP("security_code", "abc", 8)
			err := validator.Apply(rule)
			assert.Error(t, err)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Len(t, validationErr, 1)

			assert.Equal(t, "security_code", validationErr[0].Field)
			assert.Equal(t, "validation.otp_code", validationErr[0].TranslationKey)
			assert.Equal(t, "security_code", validationErr[0].TranslationValues["field"])
			assert.Equal(t, 8, validationErr[0].TranslationValues["length"])
			assert.Contains(t, validationErr[0].Message, "8-digit OTP code")
		})
	})
}
