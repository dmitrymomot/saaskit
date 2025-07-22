package sanitizer_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escapes basic HTML characters",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "escapes quotes and ampersands",
			input:    `"test" & 'value'`,
			expected: "&#34;test&#34; &amp; &#39;value&#39;",
		},
		{
			name:     "handles normal text",
			input:    "normal text",
			expected: "normal text",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.EscapeHTML(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnescapeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unescapes HTML entities",
			input:    "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
			expected: "<script>alert('xss')</script>",
		},
		{
			name:     "unescapes quotes and ampersands",
			input:    "&#34;test&#34; &amp; &#39;value&#39;",
			expected: `"test" & 'value'`,
		},
		{
			name:     "handles normal text",
			input:    "normal text",
			expected: "normal text",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.UnescapeHTML(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripScriptTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes script tags",
			input:    "<script>alert('xss')</script>Hello",
			expected: "Hello",
		},
		{
			name:     "removes script tags with attributes",
			input:    `<script type="text/javascript">alert('xss')</script>`,
			expected: "",
		},
		{
			name:     "removes multiple script tags",
			input:    "Before<script>bad()</script>Middle<script>worse()</script>After",
			expected: "BeforeMiddleAfter",
		},
		{
			name:     "handles case insensitive",
			input:    "<SCRIPT>alert('xss')</SCRIPT>",
			expected: "",
		},
		{
			name:     "handles no script tags",
			input:    "<p>Normal content</p>",
			expected: "<p>Normal content</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.StripScriptTags(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveJavaScriptEvents(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes onclick events",
			input:    `<div onclick="alert('xss')">content</div>`,
			expected: `<div>content</div>`,
		},
		{
			name:     "removes multiple event handlers",
			input:    `<div onclick="bad()" onload="worse()">content</div>`,
			expected: `<div>content</div>`,
		},
		{
			name:     "removes javascript: protocols",
			input:    `<a href="javascript:alert('xss')">link</a>`,
			expected: `<a href="alert('xss')">link</a>`,
		},
		{
			name:     "handles case insensitive",
			input:    `<div ONCLICK="alert('xss')">content</div>`,
			expected: `<div>content</div>`,
		},
		{
			name:     "handles normal attributes",
			input:    `<div class="normal">content</div>`,
			expected: `<div class="normal">content</div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.RemoveJavaScriptEvents(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreventXSS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "prevents script injection",
			input:    `<script>alert('xss')</script>Hello`,
			expected: "Hello",
		},
		{
			name:     "prevents event handler injection",
			input:    `<div onclick="alert('xss')">content</div>`,
			expected: "&lt;div&gt;content&lt;/div&gt;",
		},
		{
			name:     "comprehensive XSS prevention",
			input:    `<script>alert('xss')</script><div onclick="bad()">content</div>`,
			expected: "&lt;div&gt;content&lt;/div&gt;",
		},
		{
			name:     "handles normal content",
			input:    "Normal safe content",
			expected: "Normal safe content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.PreventXSS(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeSQLString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escapes single quotes",
			input:    "O'Reilly",
			expected: "O''Reilly",
		},
		{
			name:     "escapes multiple quotes",
			input:    "It's a 'test' string",
			expected: "It''s a ''test'' string",
		},
		{
			name:     "handles no quotes",
			input:    "normal string",
			expected: "normal string",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.EscapeSQLString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveSQLKeywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes SELECT keyword",
			input:    "SELECT * FROM users",
			expected: " *  users",
		},
		{
			name:     "removes multiple keywords",
			input:    "SELECT name FROM users WHERE id = 1",
			expected: " name  users  id = 1",
		},
		{
			name:     "handles case insensitive",
			input:    "select * from users",
			expected: " *  users",
		},
		{
			name:     "handles normal text",
			input:    "normal user input",
			expected: "normal user input",
		},
		{
			name:     "removes injection attempts",
			input:    "'; DROP TABLE users; --",
			expected: "';   users; --",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.RemoveSQLKeywords(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSQLIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "keeps valid identifier",
			input:    "user_name",
			expected: "user_name",
		},
		{
			name:     "removes invalid characters",
			input:    "user-name!@#",
			expected: "username",
		},
		{
			name:     "adds underscore if starts with number",
			input:    "123table",
			expected: "_123table",
		},
		{
			name:     "truncates long names",
			input:    "very_long_table_name_that_exceeds_the_maximum_length_allowed_for_sql_identifiers",
			expected: "very_long_table_name_that_exceeds_the_maximum_length_allowed_for",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SanitizeSQLIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreventPathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes path traversal attempts",
			input:    "../../../etc/passwd",
			expected: "etc/passwd",
		},
		{
			name:     "removes windows path traversal",
			input:    "..\\..\\windows\\system32",
			expected: "windows\\system32",
		},
		{
			name:     "removes mixed separators",
			input:    "../folder\\../file.txt",
			expected: "folder\\file.txt",
		},
		{
			name:     "handles normal path",
			input:    "documents/file.txt",
			expected: "documents/file.txt",
		},
		{
			name:     "removes standalone dots",
			input:    "file..name.txt",
			expected: "filename.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.PreventPathTraversal(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sanitizes path traversal",
			input:    "../../../etc/passwd",
			expected: "etc/passwd",
		},
		{
			name:     "removes leading slashes",
			input:    "/absolute/path",
			expected: "absolute/path",
		},
		{
			name:     "removes drive letters",
			input:    "C:\\Windows\\System32",
			expected: "Windows\\System32",
		},
		{
			name:     "handles normal relative path",
			input:    "documents/file.txt",
			expected: "documents/file.txt",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SanitizePath(tt.input)
			// Clean empty result to match filepath.Clean behavior
			if result == "." && tt.expected == "" {
				result = ""
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeShellArgument(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes shell metacharacters",
			input:    "file|name&command",
			expected: "filenamecommand",
		},
		{
			name:     "removes dangerous characters",
			input:    "test;rm -rf *",
			expected: "testrm-rf",
		},
		{
			name:     "removes quotes and spaces",
			input:    `"quoted argument"`,
			expected: "quotedargument",
		},
		{
			name:     "handles safe argument",
			input:    "normalfilename.txt",
			expected: "normalfilename.txt",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SanitizeShellArgument(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveNullBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes null bytes",
			input:    "test\x00string",
			expected: "teststring",
		},
		{
			name:     "removes multiple null bytes",
			input:    "a\x00b\x00c",
			expected: "abc",
		},
		{
			name:     "handles no null bytes",
			input:    "normal string",
			expected: "normal string",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.RemoveNullBytes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveControlSequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes ANSI escape sequences",
			input:    "\x1b[31mRed text\x1b[0m",
			expected: "Red text",
		},
		{
			name:     "preserves newlines and tabs",
			input:    "line1\nline2\ttabbed",
			expected: "line1\nline2\ttabbed",
		},
		{
			name:     "removes other control characters",
			input:    "test\x01\x02string",
			expected: "teststring",
		},
		{
			name:     "handles normal text",
			input:    "normal text",
			expected: "normal text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.RemoveControlSequences(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLimitLength(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "truncates long string",
			input:     "very long string",
			maxLength: 5,
			expected:  "very ",
		},
		{
			name:      "preserves short string",
			input:     "short",
			maxLength: 10,
			expected:  "short",
		},
		{
			name:      "handles zero length",
			input:     "test",
			maxLength: 0,
			expected:  "",
		},
		{
			name:      "handles negative length",
			input:     "test",
			maxLength: -1,
			expected:  "",
		},
		{
			name:      "handles unicode properly",
			input:     "héllo",
			maxLength: 3,
			expected:  "hél",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.LimitLength(tt.input, tt.maxLength)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeUserInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "comprehensive sanitization",
			input:    "  \x00test\x01string\x1b[31m  ",
			expected: "teststring",
		},
		{
			name:     "handles normal input",
			input:    "  normal user input  ",
			expected: "normal user input",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "limits very long input",
			input:    strings.Repeat("a", 15000),
			expected: strings.Repeat("a", 10000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SanitizeUserInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreventLDAPInjection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes LDAP special characters",
			input:    "user(*)name",
			expected: "username",
		},
		{
			name:     "removes injection attempts",
			input:    "admin)(uid=*",
			expected: "adminuid=",
		},
		{
			name:     "handles normal input",
			input:    "normaluser",
			expected: "normaluser",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.PreventLDAPInjection(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes dangerous characters",
			input:    `user<script>@example.com`,
			expected: "userscript@example.com",
		},
		{
			name:     "removes quotes",
			input:    `"user"@example.com`,
			expected: "user@example.com",
		},
		{
			name:     "handles normal email",
			input:    "user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "trims whitespace",
			input:    "  user@example.com  ",
			expected: "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SanitizeEmail(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes dangerous protocol",
			input:    "javascript:alert('xss')",
			expected: "",
		},
		{
			name:     "allows safe protocols",
			input:    "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "removes XSS attempts",
			input:    "http://example.com<script>",
			expected: "http://example.comscript",
		},
		{
			name:     "handles normal URL",
			input:    "https://www.example.com/path",
			expected: "https://www.example.com/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SanitizeURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreventHeaderInjection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes line breaks",
			input:    "normal\r\nheader",
			expected: "normalheader",
		},
		{
			name:     "removes null bytes",
			input:    "header\x00value",
			expected: "headervalue",
		},
		{
			name:     "handles normal header",
			input:    "normal header value",
			expected: "normal header value",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.PreventHeaderInjection(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSecureFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replaces dangerous characters",
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
			name:     "limits length",
			input:    strings.Repeat("a", 300),
			expected: strings.Repeat("a", 255),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SanitizeSecureFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecurityApplyPattern(t *testing.T) {
	t.Run("apply pattern with security functions", func(t *testing.T) {
		// Test comprehensive security sanitization pipeline
		maliciousInput := "<script>alert('xss')</script>user\x00input"
		result := sanitizer.Apply(maliciousInput,
			sanitizer.RemoveNullBytes,
			sanitizer.StripScriptTags,
			sanitizer.EscapeHTML,
		)
		assert.Equal(t, "userinput", result)
	})

	t.Run("compose security transformations", func(t *testing.T) {
		// Create a comprehensive input sanitizer
		inputSanitizer := sanitizer.Compose(
			sanitizer.RemoveNullBytes,
			sanitizer.RemoveControlSequences,
			func(s string) string { return sanitizer.LimitLength(s, 100) },
			sanitizer.PreventXSS,
		)

		maliciousInput := "<script>alert('xss')</script>\x00\x01dangerous input"
		result := inputSanitizer(maliciousInput)
		assert.Equal(t, "dangerous input", result)
	})
}

func TestRealWorldSecurityUsage(t *testing.T) {
	t.Run("user comment sanitization", func(t *testing.T) {
		// Create comment sanitizer for user-generated content
		commentSanitizer := sanitizer.Compose(
			sanitizer.RemoveNullBytes,
			sanitizer.RemoveControlSequences,
			sanitizer.StripScriptTags,
			sanitizer.RemoveJavaScriptEvents,
			func(s string) string { return sanitizer.LimitLength(s, 1000) },
		)

		dangerousComment := `<script>steal_data()</script>Nice post! <div onclick="evil()">Click me</div>`
		cleanComment := commentSanitizer(dangerousComment)
		assert.Equal(t, `Nice post! <div>Click me</div>`, cleanComment)
	})

	t.Run("file upload sanitization", func(t *testing.T) {
		// Create file path sanitizer
		pathSanitizer := sanitizer.Compose(
			sanitizer.PreventPathTraversal,
			sanitizer.SanitizePath,
		)

		filenameSanitizer := sanitizer.Compose(
			sanitizer.RemoveNullBytes,
			sanitizer.SanitizeSecureFilename,
		)

		dangerousPath := "../../../etc/passwd"
		dangerousFilename := "file\x00name<script>.txt"

		cleanPath := pathSanitizer(dangerousPath)
		cleanFilename := filenameSanitizer(dangerousFilename)

		assert.Equal(t, "etc/passwd", cleanPath)
		assert.Equal(t, "filename_script_.txt", cleanFilename)
	})

	t.Run("database input sanitization", func(t *testing.T) {
		// Create SQL input sanitizer
		sqlSanitizer := sanitizer.Compose(
			sanitizer.RemoveNullBytes,
			sanitizer.RemoveSQLKeywords,
			sanitizer.EscapeSQLString,
		)

		maliciousInput := "'; DROP TABLE users; --"
		cleanInput := sqlSanitizer(maliciousInput)

		// Should remove SQL keywords and escape quotes
		assert.NotContains(t, cleanInput, "DROP")
		assert.NotContains(t, cleanInput, "TABLE")
		assert.Contains(t, cleanInput, "''") // Should escape the quotes
	})
}
