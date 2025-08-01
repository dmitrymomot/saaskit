package validator

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode"
)

var (
	uppercaseRegex   = regexp.MustCompile(`[A-Z]`)
	lowercaseRegex   = regexp.MustCompile(`[a-z]`)
	digitRegex       = regexp.MustCompile(`[0-9]`)
	specialCharRegex = regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~` + "`" + `]`)

	// Common weak passwords - curated list of frequently compromised passwords
	commonPasswords = map[string]bool{
		"password":      true,
		"123456":        true,
		"password123":   true,
		"admin":         true,
		"qwerty":        true,
		"abc123":        true,
		"letmein":       true,
		"welcome":       true,
		"monkey":        true,
		"1234567890":    true,
		"dragon":        true,
		"sunshine":      true,
		"iloveyou":      true,
		"princess":      true,
		"football":      true,
		"charlie":       true,
		"aa123456":      true,
		"donald":        true,
		"password1":     true,
		"qwerty123":     true,
		"12345678":      true,
		"123456789":     true,
		"1234":          true,
		"12345":         true,
		"123123":        true,
		"111111":        true,
		"000000":        true,
		"qwertyuiop":    true,
		"asdfghjkl":     true,
		"zxcvbnm":       true,
		"qwerty12":      true,
		"qwerty1":       true,
		"password12":    true,
		"password!":     true,
		"Password":      true,
		"Password1":     true,
		"Password123":   true,
		"admin123":      true,
		"administrator": true,
		"root":          true,
		"toor":          true,
		"guest":         true,
		"test":          true,
		"testing":       true,
		"user":          true,
		"login":         true,
		"pass":          true,
		"master":        true,
		"secret":        true,
		"trustno1":      true,
		"baseball":      true,
		"basketball":    true,
		"soccer":        true,
		"hockey":        true,
		"tennis":        true,
		"golf":          true,
		"michael":       true,
		"jennifer":      true,
		"jessica":       true,
		"ashley":        true,
		"sarah":         true,
		"amanda":        true,
		"joshua":        true,
		"matthew":       true,
		"daniel":        true,
		"david":         true,
		"christopher":   true,
		"andrew":        true,
		"superman":      true,
		"batman":        true,
		"spiderman":     true,
		"pokemon":       true,
		"nintendo":      true,
		"windows":       true,
		"computer":      true,
		"internet":      true,
		"google":        true,
		"facebook":      true,
		"twitter":       true,
		"instagram":     true,
		"linkedin":      true,
		"amazon":        true,
		"apple":         true,
		"microsoft":     true,
		"samsung":       true,
		"iphone":        true,
		"android":       true,
		"freedom":       true,
		"america":       true,
		"eagle":         true,
		"flower":        true,
		"spring":        true,
		"summer":        true,
		"winter":        true,
		"autumn":        true,
		"shadow":        true,
		"midnight":      true,
		"silver":        true,
		"golden":        true,
		"diamond":       true,
		"rainbow":       true,
		"chocolate":     true,
		"vanilla":       true,
		"banana":        true,
		"orange":        true,
		"purple":        true,
		"yellow":        true,
		"jordan":        true,
		"hunter":        true,
		"jackson":       true,
		"madison":       true,
		"taylor":        true,
		"hannah":        true,
		"samantha":      true,
		"tyler":         true,
		"nicole":        true,
		"brittany":      true,
		"12341234":      true,
		"1q2w3e4r":      true,
		"1qaz2wsx":      true,
		"zaq12wsx":      true,
		"qazwsx":        true,
		"qazxsw":        true,
		"654321":        true,
		"987654321":     true,
		"abcdef":        true,
		"abcd1234":      true,
		"a1b2c3":        true,
		"123qwe":        true,
		"qwe123":        true,
		"asd123":        true,
		"123asd":        true,
		"zxc123":        true,
		"123zxc":        true,
	}
)

type PasswordStrengthConfig struct {
	MinLength        int
	MaxLength        int
	RequireUppercase bool
	RequireLowercase bool
	RequireDigits    bool
	RequireSpecial   bool
	MinCharClasses   int // Minimum number of different character classes required
}

// DefaultPasswordStrength returns NIST-recommended password policy: 8-128 chars, 3+ character classes.
func DefaultPasswordStrength() PasswordStrengthConfig {
	return PasswordStrengthConfig{
		MinLength:        8,
		MaxLength:        128,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigits:    true,
		RequireSpecial:   true,
		MinCharClasses:   3,
	}
}

func StrongPassword(field, value string, config PasswordStrengthConfig) Rule {
	return Rule{
		Check: func() bool {
			if len(value) < config.MinLength || len(value) > config.MaxLength {
				return false
			}

			charClasses := 0
			hasUpper := uppercaseRegex.MatchString(value)
			hasLower := lowercaseRegex.MatchString(value)
			hasDigit := digitRegex.MatchString(value)
			hasSpecial := specialCharRegex.MatchString(value)

			if hasUpper {
				charClasses++
			}
			if hasLower {
				charClasses++
			}
			if hasDigit {
				charClasses++
			}
			if hasSpecial {
				charClasses++
			}

			// Check specific requirements
			if config.RequireUppercase && !hasUpper {
				return false
			}
			if config.RequireLowercase && !hasLower {
				return false
			}
			if config.RequireDigits && !hasDigit {
				return false
			}
			if config.RequireSpecial && !hasSpecial {
				return false
			}

			return charClasses >= config.MinCharClasses
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("password must be %d-%d characters with required character types", config.MinLength, config.MaxLength),
			TranslationKey: "validation.password_strength",
			TranslationValues: map[string]any{
				"field":             field,
				"min_length":        config.MinLength,
				"max_length":        config.MaxLength,
				"require_uppercase": config.RequireUppercase,
				"require_lowercase": config.RequireLowercase,
				"require_digits":    config.RequireDigits,
				"require_special":   config.RequireSpecial,
				"min_char_classes":  config.MinCharClasses,
			},
		},
	}
}

func PasswordUppercase(field, value string) Rule {
	return Rule{
		Check: func() bool {
			return uppercaseRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "password must contain at least one uppercase letter",
			TranslationKey: "validation.password_uppercase",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func PasswordLowercase(field, value string) Rule {
	return Rule{
		Check: func() bool {
			return lowercaseRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "password must contain at least one lowercase letter",
			TranslationKey: "validation.password_lowercase",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func PasswordDigit(field, value string) Rule {
	return Rule{
		Check: func() bool {
			return digitRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "password must contain at least one digit",
			TranslationKey: "validation.password_digit",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func PasswordSpecialChar(field, value string) Rule {
	return Rule{
		Check: func() bool {
			return specialCharRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "password must contain at least one special character",
			TranslationKey: "validation.password_special",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func NotCommonPassword(field, value string) Rule {
	return Rule{
		Check: func() bool {
			return !commonPasswords[strings.ToLower(value)]
		},
		Error: ValidationError{
			Field:          field,
			Message:        "password is too common, please choose a different one",
			TranslationKey: "validation.password_common",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// PasswordEntropy validates password randomness using Shannon entropy.
// 50+ bits indicates strong randomness, 40-49 is moderate, <40 is weak.
func PasswordEntropy(field, value string, minEntropy float64) Rule {
	return Rule{
		Check: func() bool {
			entropy := calculatePasswordEntropy(value)
			return entropy >= minEntropy
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("password entropy too low, minimum %.1f bits required", minEntropy),
			TranslationKey: "validation.password_entropy",
			TranslationValues: map[string]any{
				"field":       field,
				"min_entropy": minEntropy,
			},
		},
	}
}

func NoRepeatingChars(field, value string, maxRepeats int) Rule {
	return Rule{
		Check: func() bool {
			if len(value) == 0 {
				return true
			}

			currentChar := rune(0)
			count := 0
			maxCount := 0

			for _, char := range value {
				if char == currentChar {
					count++
				} else {
					if count > maxCount {
						maxCount = count
					}
					currentChar = char
					count = 1
				}
			}

			if count > maxCount {
				maxCount = count
			}

			return maxCount <= maxRepeats
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("password cannot have more than %d repeating characters", maxRepeats),
			TranslationKey: "validation.password_repeating",
			TranslationValues: map[string]any{
				"field":       field,
				"max_repeats": maxRepeats,
			},
		},
	}
}

// NoSequentialChars prevents patterns like "abc" or "123" that reduce effective entropy.
func NoSequentialChars(field, value string, maxSequential int) Rule {
	return Rule{
		Check: func() bool {
			if len(value) <= maxSequential {
				return true
			}

			runes := []rune(value)
			sequentialCount := 1

			for i := 1; i < len(runes); i++ {
				if int(runes[i]) == int(runes[i-1])+1 || int(runes[i]) == int(runes[i-1])-1 {
					sequentialCount++
					if sequentialCount > maxSequential {
						return false
					}
				} else {
					sequentialCount = 1
				}
			}

			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("password cannot have more than %d sequential characters", maxSequential),
			TranslationKey: "validation.password_sequential",
			TranslationValues: map[string]any{
				"field":          field,
				"max_sequential": maxSequential,
			},
		},
	}
}

// calculatePasswordEntropy estimates password strength using Shannon entropy formula.
// Accounts for character set diversity and actual unique characters used.
func calculatePasswordEntropy(password string) float64 {
	if len(password) == 0 {
		return 0
	}

	uniqueChars := make(map[rune]bool)
	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		uniqueChars[char] = true

		if unicode.IsLower(char) {
			hasLower = true
		} else if unicode.IsUpper(char) {
			hasUpper = true
		} else if unicode.IsDigit(char) {
			hasDigit = true
		} else {
			hasSpecial = true
		}
	}

	// Estimate theoretical character set size
	charsetSize := 0
	if hasLower {
		charsetSize += 26
	}
	if hasUpper {
		charsetSize += 26
	}
	if hasDigit {
		charsetSize += 10
	}
	if hasSpecial {
		charsetSize += 32 // Approximation for common special chars
	}

	if charsetSize == 0 {
		return 0
	}

	// Use actual unique chars but cap at theoretical max
	effectiveCharsetSize := float64(len(uniqueChars))
	if effectiveCharsetSize > float64(charsetSize) {
		effectiveCharsetSize = float64(charsetSize)
	}

	// Shannon entropy: length * log2(charset_size)
	entropy := float64(len(password)) * math.Log2(effectiveCharsetSize)
	return entropy
}
