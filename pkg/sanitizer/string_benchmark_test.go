package sanitizer_test

import (
	"strings"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

var testStrings = []string{
	"hello world",
	"Hello   World   Test   Data",
	"test@example.com",
	"  trim  this  string  ",
	"UPPER CASE STRING",
	"lower case string",
	"kebab-case-string-test",
	"snake_case_string_test",
	"camelCaseStringTest",
	"<p>HTML content</p><script>alert('test')</script>",
	"This    has     extra    whitespace",
	"Control\x00chars\x1ftest",
	strings.Repeat("a", 1000),
}

func BenchmarkTrim(b *testing.B) {
	for _, s := range testStrings {
		b.Run(s[:min(20, len(s))], func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.Trim(s)
			}
		})
	}
}

func BenchmarkToLower(b *testing.B) {
	input := "HELLO WORLD TEST STRING"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.ToLower(input)
	}
}

func BenchmarkToUpper(b *testing.B) {
	input := "hello world test string"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.ToUpper(input)
	}
}

func BenchmarkToKebabCase(b *testing.B) {
	inputs := []string{
		"Hello World Test",
		"CamelCaseString",
		"snake_case_string",
		"Mixed-123-String!!!",
	}

	for _, input := range inputs {
		b.Run(input, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.ToKebabCase(input)
			}
		})
	}
}

func BenchmarkToSnakeCase(b *testing.B) {
	inputs := []string{
		"Hello World Test",
		"CamelCaseString",
		"kebab-case-string",
		"Mixed 123 String!!!",
	}

	for _, input := range inputs {
		b.Run(input, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.ToSnakeCase(input)
			}
		})
	}
}

func BenchmarkToCamelCase(b *testing.B) {
	inputs := []string{
		"hello world test",
		"kebab-case-string",
		"snake_case_string",
		"Mixed 123 String",
	}

	for _, input := range inputs {
		b.Run(input, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.ToCamelCase(input)
			}
		})
	}
}

func BenchmarkMaxLength(b *testing.B) {
	input := strings.Repeat("a", 1000)
	lengths := []int{10, 50, 100, 500}

	for _, length := range lengths {
		b.Run(string(rune(length)), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.MaxLength(input, length)
			}
		})
	}
}

func BenchmarkRemoveExtraWhitespace(b *testing.B) {
	inputs := []string{
		"hello    world",
		"lots     of     extra     spaces",
		"  leading and trailing  ",
		strings.Repeat("a ", 100),
	}

	for _, input := range inputs {
		b.Run(input[:min(20, len(input))], func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.RemoveExtraWhitespace(input)
			}
		})
	}
}

func BenchmarkRemoveControlChars(b *testing.B) {
	input := "hello\x00world\x1ftest\nkeep\tthis"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.RemoveControlChars(input)
	}
}

func BenchmarkStripHTML(b *testing.B) {
	inputs := []string{
		"<p>Simple paragraph</p>",
		"<div><p>Nested <b>tags</b></p></div>",
		"Text with &amp; &lt; &gt; entities",
		"<script>alert('xss')</script><p>Content</p>",
		strings.Repeat("<p>tag</p>", 100),
	}

	for _, input := range inputs {
		b.Run(input[:min(20, len(input))], func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.StripHTML(input)
			}
		})
	}
}

func BenchmarkKeepAlphanumeric(b *testing.B) {
	input := "Hello123!@#$%^&*()World456"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.KeepAlphanumeric(input)
	}
}

func BenchmarkKeepAlpha(b *testing.B) {
	input := "Hello123World456!@#$"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.KeepAlpha(input)
	}
}

func BenchmarkKeepDigits(b *testing.B) {
	input := "Hello123World456!@#$"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.KeepDigits(input)
	}
}

func BenchmarkSingleLine(b *testing.B) {
	input := "This\nis\na\nmulti\nline\nstring\rwith\rdifferent\rline\rendings"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.SingleLine(input)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
