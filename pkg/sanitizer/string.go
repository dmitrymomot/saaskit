package sanitizer

import (
	"html"
	"strings"
	"unicode"
)

func Trim(s string) string {
	return strings.TrimSpace(s)
}

func ToLower(s string) string {
	return strings.ToLower(s)
}

func ToUpper(s string) string {
	return strings.ToUpper(s)
}

func ToTitle(s string) string {
	return strings.ToTitle(s)
}

// ToKebabCase prevents consecutive dashes and ensures clean URL-safe identifiers.
func ToKebabCase(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}

	result := strings.Trim(b.String(), "-")
	return result
}

// ToSnakeCase prevents consecutive underscores for clean database column names.
func ToSnakeCase(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	var b strings.Builder
	prevUnderscore := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevUnderscore = false
			continue
		}
		if !prevUnderscore {
			b.WriteRune('_')
			prevUnderscore = true
		}
	}

	result := strings.Trim(b.String(), "_")
	return result
}

// ToCamelCase follows JavaScript convention: first word lowercase, subsequent words capitalized.
func ToCamelCase(s string) string {
	s = strings.TrimSpace(s)

	var b strings.Builder
	newWord := false
	first := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if first {
				b.WriteRune(unicode.ToLower(r))
				first = false
				newWord = false
				continue
			}
			if newWord {
				b.WriteRune(unicode.ToUpper(r))
				newWord = false
			} else {
				b.WriteRune(unicode.ToLower(r))
			}
			continue
		}
		if !first {
			newWord = true
		}
	}

	return b.String()
}

func TrimToLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func TrimToUpper(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// MaxLength handles Unicode properly and prevents buffer overflows from malicious input.
func MaxLength(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	return string(runes[:maxLen])
}

// RemoveExtraWhitespace prevents layout issues and normalizes user input formatting.
func RemoveExtraWhitespace(s string) string {
	normalized := whitespaceRegex.ReplaceAllString(s, " ")
	return strings.TrimSpace(normalized)
}

// RemoveControlChars prevents injection attacks while preserving common whitespace.
func RemoveControlChars(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, s)
}

// StripHTML prevents XSS by removing tags and decoding entities for safe text extraction.
func StripHTML(s string) string {
	stripped := htmlTagRegex.ReplaceAllString(s, "")

	// Decode entities like &amp; to &
	return html.UnescapeString(stripped)
}

func RemoveChars(s string, chars string) string {
	for _, char := range chars {
		s = strings.ReplaceAll(s, string(char), "")
	}
	return s
}

func ReplaceChars(s string, old string, new string) string {
	for _, char := range old {
		s = strings.ReplaceAll(s, string(char), new)
	}
	return s
}

// KeepAlphanumeric preserves spaces for readability while removing special characters.
func KeepAlphanumeric(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, s)
}

func KeepAlpha(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, s)
}

func KeepDigits(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
}

// SingleLine useful for form fields and log messages that need to be on one line.
func SingleLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")

	return RemoveExtraWhitespace(s)
}
