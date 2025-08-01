package validator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// MatchesRegex validates against custom patterns. Compiles regex on each call - cache externally for performance.
func MatchesRegex(field, value string, pattern string, description string) Rule {
	regex := regexp.MustCompile(pattern)
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			return regex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must match %s pattern", description),
			TranslationKey: "validation.regex_pattern",
			TranslationValues: map[string]any{
				"field":       field,
				"pattern":     pattern,
				"description": description,
			},
		},
	}
}

func DoesNotMatchRegex(field, value string, pattern string, description string) Rule {
	regex := regexp.MustCompile(pattern)
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return true
			}
			return !regex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must not match %s pattern", description),
			TranslationKey: "validation.regex_not_pattern",
			TranslationValues: map[string]any{
				"field":       field,
				"pattern":     pattern,
				"description": description,
			},
		},
	}
}

// ContainsPattern validates that a string contains a specific pattern.
func ContainsPattern(field, value string, pattern string, description string) Rule {
	regex := regexp.MustCompile(pattern)
	return Rule{
		Check: func() bool {
			return regex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must contain %s", description),
			TranslationKey: "validation.contains_pattern",
			TranslationValues: map[string]any{
				"field":       field,
				"pattern":     pattern,
				"description": description,
			},
		},
	}
}

// StartsWithPattern validates that a string starts with a specific pattern.
func StartsWithPattern(field, value string, pattern string) Rule {
	regex := regexp.MustCompile("^" + pattern)
	return Rule{
		Check: func() bool {
			return regex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must start with pattern: %s", pattern),
			TranslationKey: "validation.starts_with_pattern",
			TranslationValues: map[string]any{
				"field":   field,
				"pattern": pattern,
			},
		},
	}
}

// EndsWithPattern validates that a string ends with a specific pattern.
func EndsWithPattern(field, value string, pattern string) Rule {
	regex := regexp.MustCompile(pattern + "$")
	return Rule{
		Check: func() bool {
			return regex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must end with pattern: %s", pattern),
			TranslationKey: "validation.ends_with_pattern",
			TranslationValues: map[string]any{
				"field":   field,
				"pattern": pattern,
			},
		},
	}
}

// NoWhitespace validates that a string contains no whitespace characters.
func NoWhitespace(field, value string) Rule {
	return Rule{
		Check: func() bool {
			for _, char := range value {
				if unicode.IsSpace(char) {
					return false
				}
			}
			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must not contain whitespace characters",
			TranslationKey: "validation.no_whitespace",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// OnlyWhitespace validates that a string contains only whitespace characters.
func OnlyWhitespace(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if len(value) == 0 {
				return false
			}
			for _, char := range value {
				if !unicode.IsSpace(char) {
					return false
				}
			}
			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must contain only whitespace characters",
			TranslationKey: "validation.only_whitespace",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// NoControlChars validates that a string contains no control characters.
func NoControlChars(field, value string) Rule {
	return Rule{
		Check: func() bool {
			for _, char := range value {
				if unicode.IsControl(char) && char != '\t' && char != '\n' && char != '\r' {
					return false
				}
			}
			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must not contain control characters",
			TranslationKey: "validation.no_control_chars",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// PrintableChars validates that a string contains only printable characters.
func PrintableChars(field, value string) Rule {
	return Rule{
		Check: func() bool {
			for _, char := range value {
				if !unicode.IsPrint(char) && !unicode.IsSpace(char) {
					return false
				}
			}
			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must contain only printable characters",
			TranslationKey: "validation.printable_chars",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ASCIIOnly validates that a string contains only ASCII characters.
func ASCIIOnly(field, value string) Rule {
	return Rule{
		Check: func() bool {
			for _, char := range value {
				if char > 127 {
					return false
				}
			}
			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must contain only ASCII characters",
			TranslationKey: "validation.ascii_only",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// NoSpecialChars validates that a string contains no special characters (only letters, numbers, and spaces).
func NoSpecialChars(field, value string) Rule {
	return Rule{
		Check: func() bool {
			for _, char := range value {
				if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != ' ' {
					return false
				}
			}
			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must contain only letters, numbers, and spaces",
			TranslationKey: "validation.no_special_chars",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ContainsUppercase validates that a string contains at least one uppercase letter.
func ContainsUppercase(field, value string) Rule {
	return Rule{
		Check: func() bool {
			for _, char := range value {
				if unicode.IsUpper(char) {
					return true
				}
			}
			return false
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must contain at least one uppercase letter",
			TranslationKey: "validation.contains_uppercase",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ContainsLowercase validates that a string contains at least one lowercase letter.
func ContainsLowercase(field, value string) Rule {
	return Rule{
		Check: func() bool {
			for _, char := range value {
				if unicode.IsLower(char) {
					return true
				}
			}
			return false
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must contain at least one lowercase letter",
			TranslationKey: "validation.contains_lowercase",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ContainsDigit validates that a string contains at least one digit.
func ContainsDigit(field, value string) Rule {
	return Rule{
		Check: func() bool {
			for _, char := range value {
				if unicode.IsDigit(char) {
					return true
				}
			}
			return false
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must contain at least one digit",
			TranslationKey: "validation.contains_digit",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// BalancedParentheses validates that parentheses in a string are balanced.
func BalancedParentheses(field, value string) Rule {
	return Rule{
		Check: func() bool {
			count := 0
			for _, char := range value {
				switch char {
				case '(':
					count++
				case ')':
					count--
					if count < 0 {
						return false
					}
				}
			}
			return count == 0
		},
		Error: ValidationError{
			Field:          field,
			Message:        "parentheses must be balanced",
			TranslationKey: "validation.balanced_parentheses",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// WordCount validates that a string has a specific number of words.
func WordCount(field, value string, min int, max int) Rule {
	return Rule{
		Check: func() bool {
			words := strings.Fields(strings.TrimSpace(value))
			count := len(words)
			return count >= min && count <= max
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must contain between %d and %d words", min, max),
			TranslationKey: "validation.word_count",
			TranslationValues: map[string]any{
				"field": field,
				"min":   min,
				"max":   max,
			},
		},
	}
}

// LineCount validates that a string has a specific number of lines.
func LineCount(field, value string, min int, max int) Rule {
	return Rule{
		Check: func() bool {
			lines := strings.Split(value, "\n")
			count := len(lines)
			return count >= min && count <= max
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must contain between %d and %d lines", min, max),
			TranslationKey: "validation.line_count",
			TranslationValues: map[string]any{
				"field": field,
				"min":   min,
				"max":   max,
			},
		},
	}
}

// ValidOTP validates that a string is a valid OTP code with the specified length.
// The OTP must contain exactly the specified number of digits (0-9 only).
func ValidOTP(field, value string, length int) Rule {
	return Rule{
		Check: func() bool {
			if length <= 0 {
				return false
			}
			if len(value) != length {
				return false
			}
			for _, char := range value {
				if !unicode.IsDigit(char) {
					return false
				}
			}
			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be a %d-digit OTP code", length),
			TranslationKey: "validation.otp_code",
			TranslationValues: map[string]any{
				"field":  field,
				"length": length,
			},
		},
	}
}
