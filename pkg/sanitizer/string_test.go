package sanitizer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

func TestTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes leading and trailing spaces",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "removes tabs and newlines",
			input:    "\t\nhello\n\t",
			expected: "hello",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles whitespace-only string",
			input:    "   \t\n  ",
			expected: "",
		},
		{
			name:     "preserves internal whitespace",
			input:    "  hello  world  ",
			expected: "hello  world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.Trim(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts uppercase to lowercase",
			input:    "HELLO WORLD",
			expected: "hello world",
		},
		{
			name:     "handles mixed case",
			input:    "Hello World",
			expected: "hello world",
		},
		{
			name:     "preserves lowercase",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles numbers and symbols",
			input:    "Hello123!@#",
			expected: "hello123!@#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ToLower(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToUpper(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts lowercase to uppercase",
			input:    "hello world",
			expected: "HELLO WORLD",
		},
		{
			name:     "handles mixed case",
			input:    "Hello World",
			expected: "HELLO WORLD",
		},
		{
			name:     "preserves uppercase",
			input:    "HELLO WORLD",
			expected: "HELLO WORLD",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles numbers and symbols",
			input:    "hello123!@#",
			expected: "HELLO123!@#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ToUpper(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts to title case",
			input:    "hello world",
			expected: "HELLO WORLD",
		},
		{
			name:     "handles mixed case",
			input:    "Hello World",
			expected: "HELLO WORLD",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ToTitle(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrimToLower(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trims and converts to lowercase",
			input:    "  HELLO WORLD  ",
			expected: "hello world",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles whitespace-only string",
			input:    "   \t\n  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.TrimToLower(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrimToUpper(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trims and converts to uppercase",
			input:    "  hello world  ",
			expected: "HELLO WORLD",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles whitespace-only string",
			input:    "   \t\n  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.TrimToUpper(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "truncates string longer than max",
			input:    "hello world",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "preserves string shorter than max",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "preserves string equal to max",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "handles zero length",
			input:    "hello",
			maxLen:   0,
			expected: "",
		},
		{
			name:     "handles negative length",
			input:    "hello",
			maxLen:   -1,
			expected: "",
		},
		{
			name:     "handles empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
		{
			name:     "handles unicode characters",
			input:    "héllo wörld",
			maxLen:   6,
			expected: "héllo ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.MaxLength(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveExtraWhitespace(t *testing.T) {
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
			name:     "handles tabs and newlines",
			input:    "hello\t\t\nworld",
			expected: "hello world",
		},
		{
			name:     "trims leading and trailing whitespace",
			input:    "  hello  world  ",
			expected: "hello world",
		},
		{
			name:     "handles mixed whitespace",
			input:    " \t hello \n\n  world \t ",
			expected: "hello world",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles whitespace-only string",
			input:    "   \t\n  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.RemoveExtraWhitespace(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveControlChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes control characters",
			input:    "hello\x00\x01world",
			expected: "helloworld",
		},
		{
			name:     "preserves newlines, tabs, and carriage returns",
			input:    "hello\nworld\ttest\r",
			expected: "hello\nworld\ttest\r",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "preserves normal text",
			input:    "hello world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.RemoveControlChars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes HTML tags",
			input:    "<p>Hello <strong>world</strong></p>",
			expected: "Hello world",
		},
		{
			name:     "handles self-closing tags",
			input:    "Hello<br/>world",
			expected: "Helloworld",
		},
		{
			name:     "unescapes HTML entities",
			input:    "&lt;p&gt;Hello &amp; goodbye&lt;/p&gt;",
			expected: "<p>Hello & goodbye</p>",
		},
		{
			name:     "handles mixed content",
			input:    "<div>Hello &quot;world&quot;</div>",
			expected: "Hello \"world\"",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles string without HTML",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.StripHTML(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		chars    string
		expected string
	}{
		{
			name:     "removes specified characters",
			input:    "hello world",
			chars:    "lo",
			expected: "he wrd",
		},
		{
			name:     "handles empty chars",
			input:    "hello world",
			chars:    "",
			expected: "hello world",
		},
		{
			name:     "handles empty input",
			input:    "",
			chars:    "abc",
			expected: "",
		},
		{
			name:     "removes multiple occurrences",
			input:    "aaabbbccc",
			chars:    "ac",
			expected: "bbb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.RemoveChars(tt.input, tt.chars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		old      string
		new      string
		expected string
	}{
		{
			name:     "replaces specified characters",
			input:    "hello world",
			old:      "lo",
			new:      "X",
			expected: "heXXX wXrXd",
		},
		{
			name:     "handles empty old chars",
			input:    "hello world",
			old:      "",
			new:      "X",
			expected: "hello world",
		},
		{
			name:     "handles empty input",
			input:    "",
			old:      "abc",
			new:      "X",
			expected: "",
		},
		{
			name:     "replaces with empty string",
			input:    "hello world",
			old:      "l",
			new:      "",
			expected: "heo word",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ReplaceChars(tt.input, tt.old, tt.new)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKeepAlphanumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "keeps letters, digits, and spaces",
			input:    "hello123 world!@#",
			expected: "hello123 world",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles symbols only",
			input:    "!@#$%",
			expected: "",
		},
		{
			name:     "preserves unicode letters",
			input:    "héllo wörld123",
			expected: "héllo wörld123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.KeepAlphanumeric(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKeepAlpha(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "keeps letters and spaces only",
			input:    "hello123 world!@#",
			expected: "hello world",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles numbers and symbols only",
			input:    "123!@#",
			expected: "",
		},
		{
			name:     "preserves unicode letters",
			input:    "héllo wörld123",
			expected: "héllo wörld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.KeepAlpha(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKeepDigits(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "keeps digits only",
			input:    "hello123 world!@#456",
			expected: "123456",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles letters and symbols only",
			input:    "hello!@#",
			expected: "",
		},
		{
			name:     "handles unicode digits",
			input:    "abc123def456",
			expected: "123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.KeepDigits(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSingleLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts newlines to spaces",
			input:    "hello\nworld",
			expected: "hello world",
		},
		{
			name:     "converts carriage returns to spaces",
			input:    "hello\rworld",
			expected: "hello world",
		},
		{
			name:     "handles mixed line breaks",
			input:    "hello\n\rworld\n",
			expected: "hello world",
		},
		{
			name:     "normalizes whitespace",
			input:    "hello\n  \n  world",
			expected: "hello world",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles single line already",
			input:    "hello world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SingleLine(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts spaces and underscores",
			input:    "Hello World_test",
			expected: "hello-world-test",
		},
		{
			name:     "handles multiple separators",
			input:    "  Hello---World__Again  ",
			expected: "hello-world-again",
		},
		{
			name:     "trims hyphens",
			input:    "--Hello--",
			expected: "hello",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ToKebabCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts spaces and hyphens",
			input:    "Hello-World test",
			expected: "hello_world_test",
		},
		{
			name:     "handles multiple separators",
			input:    "  Hello---World__Again  ",
			expected: "hello_world_again",
		},
		{
			name:     "trims underscores",
			input:    "__Hello__",
			expected: "hello",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ToSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts spaces and underscores",
			input:    "Hello World_test",
			expected: "helloWorldTest",
		},
		{
			name:     "handles multiple separators",
			input:    "  hello---world__again  ",
			expected: "helloWorldAgain",
		},
		{
			name:     "trims separators",
			input:    "--Hello--",
			expected: "hello",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ToCamelCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
