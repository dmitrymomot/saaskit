package sanitizer_test

import (
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

func BenchmarkApply(b *testing.B) {
	input := "  Hello   World  "
	transforms := []func(string) string{
		sanitizer.Trim,
		sanitizer.ToLower,
		sanitizer.RemoveExtraWhitespace,
	}

	b.Run("single", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_ = sanitizer.Apply(input, sanitizer.Trim)
		}
	})

	b.Run("multiple", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_ = sanitizer.Apply(input, transforms...)
		}
	})
}

func BenchmarkCompose(b *testing.B) {
	transforms := []func(string) string{
		sanitizer.Trim,
		sanitizer.ToLower,
		sanitizer.RemoveExtraWhitespace,
		sanitizer.ToKebabCase,
	}

	composed := sanitizer.Compose(transforms...)
	input := "  Hello   World  Test  "

	b.ResetTimer()
	for b.Loop() {
		_ = composed(input)
	}
}

func BenchmarkChain(b *testing.B) {
	chain := sanitizer.Compose(
		sanitizer.Trim,
		sanitizer.RemoveExtraWhitespace,
		sanitizer.ToLower,
		sanitizer.ToSnakeCase,
	)

	inputs := []string{
		"  Simple Test  ",
		"Complex    String   With   Spaces",
		"MixedCaseStringTest",
	}

	for _, input := range inputs {
		b.Run(input[:min(20, len(input))], func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = chain(input)
			}
		})
	}
}

func BenchmarkApplyToPtr(b *testing.B) {
	input := "  Test String  "

	b.ResetTimer()
	for b.Loop() {
		// Use Apply with pointer dereference
		result := sanitizer.Apply(input, sanitizer.Trim, sanitizer.ToLower)
		_ = result
	}
}

// Benchmark complex transformation pipeline
func BenchmarkComplexPipeline(b *testing.B) {
	// Email normalization pipeline
	emailPipeline := sanitizer.Compose(
		sanitizer.Trim,
		sanitizer.ToLower,
		sanitizer.NormalizeEmail,
	)

	// URL normalization pipeline
	urlPipeline := sanitizer.Compose(
		sanitizer.Trim,
		sanitizer.NormalizeURL,
		sanitizer.RemoveQueryParams,
	)

	// Phone normalization pipeline
	phonePipeline := sanitizer.Compose(
		sanitizer.NormalizePhone,
		sanitizer.FormatPhoneUS,
	)

	b.Run("email_pipeline", func(b *testing.B) {
		input := "  John.Doe@EXAMPLE.COM  "
		b.ResetTimer()
		for b.Loop() {
			_ = emailPipeline(input)
		}
	})

	b.Run("url_pipeline", func(b *testing.B) {
		input := "  HTTP://EXAMPLE.COM/path?query=1  "
		b.ResetTimer()
		for b.Loop() {
			_ = urlPipeline(input)
		}
	})

	b.Run("phone_pipeline", func(b *testing.B) {
		input := "(555) 123-4567"
		b.ResetTimer()
		for b.Loop() {
			_ = phonePipeline(input)
		}
	})
}
