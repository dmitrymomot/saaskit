package sanitizer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trims whitespace and converts to lowercase",
			input:    "  USER@EXAMPLE.COM  ",
			expected: "user@example.com",
		},
		{
			name:     "removes consecutive dots in local part",
			input:    "user..name@example.com",
			expected: "user.name@example.com",
		},
		{
			name:     "removes leading and trailing dots in local part",
			input:    ".user.name.@example.com",
			expected: "user.name@example.com",
		},
		{
			name:     "handles normal email",
			input:    "user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "handles invalid email format",
			input:    "invalid-email",
			expected: "invalid-email",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.NormalizeEmail(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractEmailDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "extracts domain from valid email",
			input:    "user@EXAMPLE.COM",
			expected: "example.com",
		},
		{
			name:     "handles email with spaces",
			input:    "  user@domain.org  ",
			expected: "domain.org",
		},
		{
			name:     "returns empty for invalid email",
			input:    "invalid-email",
			expected: "",
		},
		{
			name:     "returns empty for empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ExtractEmailDomain(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "masks normal email",
			input:    "user@example.com",
			expected: "u***@example.com",
		},
		{
			name:     "masks single character local part",
			input:    "a@example.com",
			expected: "*@example.com",
		},
		{
			name:     "handles email with spaces",
			input:    "  testuser@domain.org  ",
			expected: "t*******@domain.org",
		},
		{
			name:     "handles invalid email format",
			input:    "invalid-email",
			expected: "invalid-email",
		},
		{
			name:     "handles empty local part",
			input:    "@example.com",
			expected: "@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.MaskEmail(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes formatting from phone number",
			input:    "(123) 456-7890",
			expected: "1234567890",
		},
		{
			name:     "removes spaces and dashes",
			input:    "123 456 7890",
			expected: "1234567890",
		},
		{
			name:     "handles phone with extensions",
			input:    "123-456-7890 ext 123",
			expected: "1234567890123",
		},
		{
			name:     "handles international format",
			input:    "+1 (123) 456-7890",
			expected: "11234567890",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.NormalizePhone(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPhoneUS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "formats 10-digit number",
			input:    "1234567890",
			expected: "(123) 456-7890",
		},
		{
			name:     "formats number with existing formatting",
			input:    "123-456-7890",
			expected: "(123) 456-7890",
		},
		{
			name:     "returns original if not 10 digits",
			input:    "123456789",
			expected: "123456789",
		},
		{
			name:     "returns original if too many digits",
			input:    "12345678901",
			expected: "12345678901",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.FormatPhoneUS(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "masks phone showing last 4 digits",
			input:    "1234567890",
			expected: "******7890",
		},
		{
			name:     "masks formatted phone",
			input:    "(123) 456-7890",
			expected: "******7890",
		},
		{
			name:     "handles short phone",
			input:    "123",
			expected: "***",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.MaskPhone(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "adds https to domain",
			input:    "example.com",
			expected: "https://example.com",
		},
		{
			name:     "preserves existing protocol",
			input:    "http://example.com",
			expected: "http://example.com",
		},
		{
			name:     "normalizes host to lowercase",
			input:    "https://EXAMPLE.COM/PATH",
			expected: "https://example.com/PATH",
		},
		{
			name:     "removes trailing slash from root",
			input:    "https://example.com/",
			expected: "https://example.com",
		},
		{
			name:     "trims whitespace",
			input:    "  example.com/path  ",
			expected: "https://example.com/path",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.NormalizeURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "extracts domain from URL with protocol",
			input:    "https://example.com/path",
			expected: "example.com",
		},
		{
			name:     "extracts domain from URL without protocol",
			input:    "example.com/path",
			expected: "example.com",
		},
		{
			name:     "converts domain to lowercase",
			input:    "EXAMPLE.COM",
			expected: "example.com",
		},
		{
			name:     "extracts domain with port",
			input:    "https://example.com:8080/path",
			expected: "example.com:8080",
		},
		{
			name:     "handles invalid URL",
			input:    "://invalid",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ExtractDomain(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveQueryParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes query parameters",
			input:    "https://example.com/path?param1=value1&param2=value2",
			expected: "https://example.com/path",
		},
		{
			name:     "handles URL without query params",
			input:    "https://example.com/path",
			expected: "https://example.com/path",
		},
		{
			name:     "handles URL with fragment",
			input:    "https://example.com/path?param=value#section",
			expected: "https://example.com/path#section",
		},
		{
			name:     "handles invalid URL",
			input:    "://invalid",
			expected: "://invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.RemoveQueryParams(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveFragment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes fragment",
			input:    "https://example.com/path#section",
			expected: "https://example.com/path",
		},
		{
			name:     "handles URL without fragment",
			input:    "https://example.com/path",
			expected: "https://example.com/path",
		},
		{
			name:     "handles URL with query params and fragment",
			input:    "https://example.com/path?param=value#section",
			expected: "https://example.com/path?param=value",
		},
		{
			name:     "handles invalid URL",
			input:    "://invalid",
			expected: "://invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.RemoveFragment(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeCreditCard(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes spaces and dashes",
			input:    "1234 5678 9012 3456",
			expected: "1234567890123456",
		},
		{
			name:     "removes various formatting",
			input:    "1234-5678-9012-3456",
			expected: "1234567890123456",
		},
		{
			name:     "handles already clean number",
			input:    "1234567890123456",
			expected: "1234567890123456",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.NormalizeCreditCard(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskCreditCard(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "masks credit card showing last 4 digits",
			input:    "1234567890123456",
			expected: "************3456",
		},
		{
			name:     "masks formatted credit card",
			input:    "1234 5678 9012 3456",
			expected: "************3456",
		},
		{
			name:     "handles short number",
			input:    "123",
			expected: "***",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.MaskCreditCard(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatCreditCard(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "formats 16-digit card with spaces",
			input:    "1234567890123456",
			expected: "1234 5678 9012 3456",
		},
		{
			name:     "formats 15-digit card (Amex)",
			input:    "123456789012345",
			expected: "1234 5678 9012 345",
		},
		{
			name:     "returns original if too short",
			input:    "123456789012",
			expected: "123456789012",
		},
		{
			name:     "returns original if too long",
			input:    "12345678901234567890",
			expected: "12345678901234567890",
		},
		{
			name:     "handles already formatted card",
			input:    "1234-5678-9012-3456",
			expected: "1234 5678 9012 3456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.FormatCreditCard(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeSSN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes dashes from SSN",
			input:    "123-45-6789",
			expected: "123456789",
		},
		{
			name:     "removes spaces from SSN",
			input:    "123 45 6789",
			expected: "123456789",
		},
		{
			name:     "handles clean SSN",
			input:    "123456789",
			expected: "123456789",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.NormalizeSSN(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskSSN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "masks SSN showing last 4 digits",
			input:    "123456789",
			expected: "*****6789",
		},
		{
			name:     "masks formatted SSN",
			input:    "123-45-6789",
			expected: "*****6789",
		},
		{
			name:     "handles short number",
			input:    "123",
			expected: "***",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.MaskSSN(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatSSN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "formats 9-digit SSN",
			input:    "123456789",
			expected: "123-45-6789",
		},
		{
			name:     "formats already formatted SSN",
			input:    "123-45-6789",
			expected: "123-45-6789",
		},
		{
			name:     "returns original if not 9 digits",
			input:    "12345678",
			expected: "12345678",
		},
		{
			name:     "returns original if too many digits",
			input:    "1234567890",
			expected: "1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.FormatSSN(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizePostalCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes spaces and converts to uppercase",
			input:    "k1a 0a6",
			expected: "K1A0A6",
		},
		{
			name:     "trims whitespace",
			input:    "  12345  ",
			expected: "12345",
		},
		{
			name:     "handles already normalized",
			input:    "K1A0A6",
			expected: "K1A0A6",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.NormalizePostalCode(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPostalCodeUS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "formats 5-digit ZIP",
			input:    "12345",
			expected: "12345",
		},
		{
			name:     "formats 9-digit ZIP+4",
			input:    "123456789",
			expected: "12345-6789",
		},
		{
			name:     "formats ZIP with existing formatting",
			input:    "12345-6789",
			expected: "12345-6789",
		},
		{
			name:     "returns original if wrong length",
			input:    "1234",
			expected: "1234",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.FormatPostalCodeUS(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPostalCodeCA(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "formats Canadian postal code",
			input:    "k1a0a6",
			expected: "K1A 0A6",
		},
		{
			name:     "formats already formatted postal code",
			input:    "K1A 0A6",
			expected: "K1A 0A6",
		},
		{
			name:     "returns original if wrong length",
			input:    "K1A0A",
			expected: "K1A0A",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.FormatPostalCodeCA(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		visibleChars int
		expected     string
	}{
		{
			name:         "masks middle of string",
			input:        "sensitive",
			visibleChars: 2,
			expected:     "se*****ve",
		},
		{
			name:         "masks with 1 visible char",
			input:        "password",
			visibleChars: 1,
			expected:     "p******d",
		},
		{
			name:         "handles short string",
			input:        "hi",
			visibleChars: 2,
			expected:     "**",
		},
		{
			name:         "handles negative visible chars",
			input:        "test",
			visibleChars: -1,
			expected:     "t**t",
		},
		{
			name:         "handles zero visible chars",
			input:        "test",
			visibleChars: 0,
			expected:     "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.MaskString(tt.input, tt.visibleChars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveNonAlphanumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes special characters",
			input:    "hello@world!123",
			expected: "helloworld123",
		},
		{
			name:     "removes spaces and punctuation",
			input:    "test string, with punctuation.",
			expected: "teststringwithpunctuation",
		},
		{
			name:     "handles alphanumeric only",
			input:    "abc123",
			expected: "abc123",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.RemoveNonAlphanumeric(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replaces multiple spaces with single space",
			input:    "hello    world",
			expected: "hello world",
		},
		{
			name:     "replaces tabs and newlines with spaces",
			input:    "hello\t\nworld",
			expected: "hello world",
		},
		{
			name:     "trims leading and trailing whitespace",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "handles mixed whitespace",
			input:    "\t  hello   \n\n  world  \t",
			expected: "hello world",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.NormalizeWhitespace(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "extracts numbers from mixed content",
			input:    "abc123def456",
			expected: "123456",
		},
		{
			name:     "extracts separated numbers",
			input:    "test 123 and 456 more",
			expected: "123456",
		},
		{
			name:     "handles no numbers",
			input:    "no numbers here",
			expected: "",
		},
		{
			name:     "handles only numbers",
			input:    "123456",
			expected: "123456",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ExtractNumbers(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replaces unsafe characters",
			input:    "file<name>with:unsafe/chars",
			expected: "file_name_with_unsafe_chars",
		},
		{
			name:     "trims spaces and dots",
			input:    "  .filename.  ",
			expected: "filename",
		},
		{
			name:     "handles safe filename",
			input:    "normal_filename.txt",
			expected: "normal_filename.txt",
		},
		{
			name:     "handles empty result",
			input:    "...",
			expected: "file",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatApplyPattern(t *testing.T) {
	t.Run("apply pattern with format functions", func(t *testing.T) {
		// Test email sanitization pipeline
		dirtyEmail := "  USER..NAME@EXAMPLE.COM  "
		cleanEmail := sanitizer.Apply(dirtyEmail,
			sanitizer.NormalizeEmail,
		)
		assert.Equal(t, "user.name@example.com", cleanEmail)
	})

	t.Run("compose format transformations", func(t *testing.T) {
		// Create a phone sanitizer
		phoneSanitizer := sanitizer.Compose(
			sanitizer.NormalizePhone,
			sanitizer.FormatPhoneUS,
		)

		result := phoneSanitizer("123 456 7890")
		assert.Equal(t, "(123) 456-7890", result)
	})

	t.Run("compose URL cleaning pipeline", func(t *testing.T) {
		// Create URL cleaner
		urlCleaner := sanitizer.Compose(
			sanitizer.NormalizeURL,
			sanitizer.RemoveQueryParams,
			sanitizer.RemoveFragment,
		)

		result := urlCleaner("  EXAMPLE.COM/path?param=value#section  ")
		assert.Equal(t, "https://example.com/path", result)
	})
}

func TestRealWorldFormatUsage(t *testing.T) {
	t.Run("user profile sanitization", func(t *testing.T) {
		// Create sanitizers for different profile fields
		emailSanitizer := sanitizer.Compose(
			sanitizer.NormalizeEmail,
		)

		phoneSanitizer := sanitizer.Compose(
			sanitizer.NormalizePhone,
			sanitizer.FormatPhoneUS,
		)

		// Test profile data
		rawEmail := "  USER..EMAIL@DOMAIN.COM  "
		rawPhone := "123 456 7890"

		cleanEmail := emailSanitizer(rawEmail)
		cleanPhone := phoneSanitizer(rawPhone)

		assert.Equal(t, "user.email@domain.com", cleanEmail)
		assert.Equal(t, "(123) 456-7890", cleanPhone)
	})

	t.Run("payment form sanitization", func(t *testing.T) {
		// Create credit card sanitizer
		cardSanitizer := sanitizer.Compose(
			sanitizer.NormalizeCreditCard,
		)

		rawCard := "1234 5678 9012 3456"
		cleanCard := cardSanitizer(rawCard)

		assert.Equal(t, "1234567890123456", cleanCard)
	})

	t.Run("address form sanitization", func(t *testing.T) {
		// Create postal code sanitizer
		zipSanitizer := sanitizer.Compose(
			sanitizer.FormatPostalCodeUS,
		)

		rawZip := "123456789"
		cleanZip := zipSanitizer(rawZip)

		assert.Equal(t, "12345-6789", cleanZip)
	})
}
