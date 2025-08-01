package sanitizer_test

import (
	"strings"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

func BenchmarkNormalizeEmail(b *testing.B) {
	emails := []string{
		"test@example.com",
		"John.Doe@EXAMPLE.COM",
		"user...name@domain.com",
		"  email@test.com  ",
	}

	for _, email := range emails {
		b.Run(email, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.NormalizeEmail(email)
			}
		})
	}
}

func BenchmarkMaskEmail(b *testing.B) {
	email := "john.doe@example.com"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.MaskEmail(email)
	}
}

func BenchmarkNormalizePhone(b *testing.B) {
	phones := []string{
		"(555) 123-4567",
		"555.123.4567",
		"+1-555-123-4567",
		"5551234567",
	}

	for _, phone := range phones {
		b.Run(phone, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.NormalizePhone(phone)
			}
		})
	}
}

func BenchmarkFormatPhoneUS(b *testing.B) {
	phone := "5551234567"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.FormatPhoneUS(phone)
	}
}

func BenchmarkMaskPhone(b *testing.B) {
	phone := "5551234567"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.MaskPhone(phone)
	}
}

func BenchmarkNormalizeURL(b *testing.B) {
	urls := []string{
		"example.com",
		"http://example.com/",
		"HTTPS://EXAMPLE.COM/PATH",
		"  https://example.com/path?query=1  ",
	}

	for _, url := range urls {
		b.Run(url[:min(20, len(url))], func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.NormalizeURL(url)
			}
		})
	}
}

func BenchmarkExtractDomain(b *testing.B) {
	url := "https://www.example.com/path?query=1"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.ExtractDomain(url)
	}
}

func BenchmarkRemoveQueryParams(b *testing.B) {
	url := "https://example.com/path?param1=value1&param2=value2"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.RemoveQueryParams(url)
	}
}

func BenchmarkNormalizeCreditCard(b *testing.B) {
	cards := []string{
		"4111-1111-1111-1111",
		"4111 1111 1111 1111",
		"4111111111111111",
	}

	for _, card := range cards {
		b.Run(card, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.NormalizeCreditCard(card)
			}
		})
	}
}

func BenchmarkMaskCreditCard(b *testing.B) {
	card := "4111111111111111"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.MaskCreditCard(card)
	}
}

func BenchmarkFormatCreditCard(b *testing.B) {
	card := "4111111111111111"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.FormatCreditCard(card)
	}
}

func BenchmarkNormalizeSSN(b *testing.B) {
	ssns := []string{
		"123-45-6789",
		"123 45 6789",
		"123456789",
	}

	for _, ssn := range ssns {
		b.Run(ssn, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.NormalizeSSN(ssn)
			}
		})
	}
}

func BenchmarkMaskSSN(b *testing.B) {
	ssn := "123456789"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.MaskSSN(ssn)
	}
}

func BenchmarkFormatSSN(b *testing.B) {
	ssn := "123456789"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.FormatSSN(ssn)
	}
}

func BenchmarkNormalizePostalCode(b *testing.B) {
	codes := []string{
		"12345",
		"12345-6789",
		"K1A 0B1",
		"  k1a0b1  ",
	}

	for _, code := range codes {
		b.Run(code, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.NormalizePostalCode(code)
			}
		})
	}
}

func BenchmarkFormatPostalCodeUS(b *testing.B) {
	codes := []string{
		"12345",
		"123456789",
	}

	for _, code := range codes {
		b.Run(code, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.FormatPostalCodeUS(code)
			}
		})
	}
}

func BenchmarkMaskString(b *testing.B) {
	testCases := []struct {
		name         string
		input        string
		visibleChars int
	}{
		{"short", "test", 1},
		{"medium", "hello world", 2},
		{"long", strings.Repeat("a", 100), 5},
		{"unicode", "Hello 世界", 2},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.MaskString(tc.input, tc.visibleChars)
			}
		})
	}
}

func BenchmarkRemoveNonAlphanumeric(b *testing.B) {
	input := "Hello123!@#$%^&*()World456"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.RemoveNonAlphanumeric(input)
	}
}

func BenchmarkNormalizeWhitespace(b *testing.B) {
	input := "Hello   \t\n   World   \r\n   Test"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.NormalizeWhitespace(input)
	}
}

func BenchmarkExtractNumbers(b *testing.B) {
	input := "Order #12345 shipped on 2023-10-15 for $99.99"
	b.ResetTimer()
	for b.Loop() {
		_ = sanitizer.ExtractNumbers(input)
	}
}

func BenchmarkSanitizeFilename(b *testing.B) {
	filenames := []string{
		"normal_file.txt",
		"file<with>special:chars?.txt",
		"file/with\\path|separators.txt",
		strings.Repeat("a", 300) + ".txt",
		"   .hidden.file   ",
	}

	for _, filename := range filenames {
		b.Run(filename[:min(20, len(filename))], func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = sanitizer.SanitizeFilename(filename)
			}
		})
	}
}
