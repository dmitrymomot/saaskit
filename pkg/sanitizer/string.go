package sanitizer

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

// Trim removes leading and trailing whitespace from a string.
func Trim(s string) string {
	return strings.TrimSpace(s)
}

// ToLower converts a string to lowercase.
func ToLower(s string) string {
	return strings.ToLower(s)
}

// ToUpper converts a string to uppercase.
func ToUpper(s string) string {
	return strings.ToUpper(s)
}

// ToTitle converts a string to title case.
func ToTitle(s string) string {
	return strings.ToTitle(s)
}

// ToKebabCase converts a string to kebab-case by replacing non-alphanumeric
// characters with hyphens and normalizing multiple hyphens.
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

// ToSnakeCase converts a string to snake_case by replacing non-alphanumeric
// characters with underscores and normalizing multiple underscores.
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

// ToCamelCase converts a string to camelCase. Non-alphanumeric characters start
// new words, with the first word lowercased and subsequent words capitalized.
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

// TrimToLower removes leading and trailing whitespace and converts to lowercase.
func TrimToLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// TrimToUpper removes leading and trailing whitespace and converts to uppercase.
func TrimToUpper(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// MaxLength truncates a string to the specified maximum length.
// If the string is longer than maxLen, it will be truncated.
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

// RemoveExtraWhitespace normalizes whitespace by replacing multiple consecutive
// whitespace characters with a single space and trimming.
func RemoveExtraWhitespace(s string) string {
	// Replace multiple whitespace with single space
	re := regexp.MustCompile(`\s+`)
	normalized := re.ReplaceAllString(s, " ")
	return strings.TrimSpace(normalized)
}

// RemoveControlChars removes control characters from a string,
// keeping only printable characters and common whitespace.
func RemoveControlChars(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1 // Remove the character
		}
		return r
	}, s)
}

// StripHTML removes HTML tags and unescapes HTML entities.
func StripHTML(s string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	stripped := re.ReplaceAllString(s, "")

	// Unescape HTML entities
	return html.UnescapeString(stripped)
}

// RemoveChars removes all occurrences of the specified characters from a string.
func RemoveChars(s string, chars string) string {
	for _, char := range chars {
		s = strings.ReplaceAll(s, string(char), "")
	}
	return s
}

// ReplaceChars replaces all occurrences of characters in 'old' with 'new'.
func ReplaceChars(s string, old string, new string) string {
	for _, char := range old {
		s = strings.ReplaceAll(s, string(char), new)
	}
	return s
}

// KeepAlphanumeric keeps only alphanumeric characters and spaces.
func KeepAlphanumeric(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, s)
}

// KeepAlpha keeps only alphabetic characters and spaces.
func KeepAlpha(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, s)
}

// KeepDigits keeps only numeric digits.
func KeepDigits(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
}

// SingleLine converts a multi-line string to a single line by replacing
// line breaks with spaces and normalizing whitespace.
func SingleLine(s string) string {
	// Replace line breaks with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")

	// Normalize whitespace
	return RemoveExtraWhitespace(s)
}
