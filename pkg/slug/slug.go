package slug

import (
	"crypto/rand"
	"strings"
	"unicode"
)

// Option configures the slug generation behavior.
type Option func(*config)

// config holds the configuration for slug generation.
type config struct {
	maxLength     int
	separator     string
	lowercase     bool
	stripChars    string
	customReplace map[string]string
	suffixLength  int
}

// defaultConfig returns the default configuration.
func defaultConfig() *config {
	return &config{
		maxLength:     0, // no limit
		separator:     "-",
		lowercase:     true,
		stripChars:    "",
		customReplace: nil,
		suffixLength:  0, // no suffix by default
	}
}

// MaxLength sets the maximum length of the generated slug.
// If the slug exceeds this length, it will be truncated.
func MaxLength(n int) Option {
	return func(c *config) {
		c.maxLength = n
	}
}

// Separator sets the separator character for the slug.
// Default is "-".
func Separator(s string) Option {
	return func(c *config) {
		c.separator = s
	}
}

// Lowercase controls whether the slug should be converted to lowercase.
// Default is true.
func Lowercase(enabled bool) Option {
	return func(c *config) {
		c.lowercase = enabled
	}
}

// StripChars sets additional characters to strip from the slug.
func StripChars(chars string) Option {
	return func(c *config) {
		c.stripChars = chars
	}
}

// CustomReplace sets custom string replacements to apply before slugification.
// For example: {"&": "and", "@": "at"}
func CustomReplace(replacements map[string]string) Option {
	return func(c *config) {
		c.customReplace = replacements
	}
}

// WithSuffix adds a random alphanumeric suffix to reduce collision possibility.
// The suffix is separated by the configured separator.
// Example: "hello-world-x7g3k2" (with length=6)
func WithSuffix(length int) Option {
	return func(c *config) {
		c.suffixLength = length
	}
}

// shouldBreakForLength checks if adding a separator would exceed the max length.
func shouldBreakForLength(cfg *config, currentRuneCount int) bool {
	return cfg.maxLength > 0 && currentRuneCount+len(cfg.separator) > cfg.maxLength
}

// Make creates a URL-safe slug from the input string.
// It normalizes the string by replacing spaces and special characters
// with the separator (default "-"), and optionally converts to lowercase.
func Make(s string, opts ...Option) string {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Apply custom replacements first
	if cfg.customReplace != nil {
		for old, new := range cfg.customReplace {
			s = strings.ReplaceAll(s, old, new)
		}
	}

	// Strip specified characters
	if cfg.stripChars != "" {
		for _, char := range cfg.stripChars {
			s = strings.ReplaceAll(s, string(char), "")
		}
	}

	// Pre-allocate builder with estimated capacity
	var b strings.Builder
	b.Grow(len(s))

	lastWasSep := true // Start as true to avoid leading separator
	runeCount := 0

	for _, r := range s {
		// Check max length (counts runes, not bytes)
		if cfg.maxLength > 0 && runeCount >= cfg.maxLength {
			break
		}

		if cfg.lowercase {
			r = unicode.ToLower(r)
		}

		// ASCII letters and digits pass through unchanged
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastWasSep = false
			runeCount++
			continue
		}

		// Try to normalize diacritics (é → e, ñ → n, etc.)
		if normalized, ok := normalizeDiacritic(r); ok {
			if cfg.lowercase {
				normalized = unicode.ToLower(normalized)
			}
			b.WriteRune(normalized)
			lastWasSep = false
			runeCount++
			continue
		}

		// Replace all other characters with separator, but avoid consecutive separators
		if !lastWasSep {
			if shouldBreakForLength(cfg, runeCount) {
				break
			}
			b.WriteString(cfg.separator)
			lastWasSep = true
			runeCount += len([]rune(cfg.separator))
		}
	}

	result := strings.TrimSuffix(b.String(), cfg.separator)

	// Add random suffix for collision avoidance if requested
	if cfg.suffixLength > 0 {
		actualSuffixLen := cfg.suffixLength
		if cfg.maxLength > 0 && cfg.suffixLength > cfg.maxLength {
			actualSuffixLen = cfg.maxLength
		}

		suffix := generateSuffix(actualSuffixLen, cfg.lowercase)

		// Ensure total length doesn't exceed maxLength
		if cfg.maxLength > 0 {
			totalLen := len([]rune(result)) + len([]rune(cfg.separator)) + actualSuffixLen
			if totalLen > cfg.maxLength {
				// Truncate main slug to make room for suffix
				mainSlugMaxLen := cfg.maxLength - len([]rune(cfg.separator)) - actualSuffixLen
				if mainSlugMaxLen > 0 {
					runes := []rune(result)
					if len(runes) > mainSlugMaxLen {
						result = string(runes[:mainSlugMaxLen])
					}
				} else {
					// No room for main slug, use suffix only
					result = ""
				}
			}
		}

		if result != "" {
			result = result + cfg.separator + suffix
		} else {
			result = suffix
		}
	}

	return result
}

// diacriticMap maps common Latin diacritics to ASCII equivalents.
// Covers major European languages but not exhaustive for all Unicode ranges.
var diacriticMap = map[rune]rune{
	// lowercase a
	'à': 'a', 'á': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'å': 'a', 'ā': 'a', 'ă': 'a', 'ą': 'a',
	// uppercase A
	'À': 'A', 'Á': 'A', 'Â': 'A', 'Ã': 'A', 'Ä': 'A', 'Å': 'A', 'Ā': 'A', 'Ă': 'A', 'Ą': 'A',
	// c/C
	'ç': 'c', 'ć': 'c', 'č': 'c',
	'Ç': 'C', 'Ć': 'C', 'Č': 'C',
	// d/D
	'đ': 'd', 'ď': 'd',
	'Đ': 'D', 'Ď': 'D',
	// e/E
	'è': 'e', 'é': 'e', 'ê': 'e', 'ë': 'e', 'ē': 'e', 'ė': 'e', 'ę': 'e', 'ě': 'e',
	'È': 'E', 'É': 'E', 'Ê': 'E', 'Ë': 'E', 'Ē': 'E', 'Ė': 'E', 'Ę': 'E', 'Ě': 'E',
	// i/I
	'ì': 'i', 'í': 'i', 'î': 'i', 'ï': 'i', 'ī': 'i', 'į': 'i',
	'Ì': 'I', 'Í': 'I', 'Î': 'I', 'Ï': 'I', 'Ī': 'I', 'Į': 'I',
	// l/L
	'ł': 'l',
	'Ł': 'L',
	// n/N
	'ñ': 'n', 'ń': 'n', 'ň': 'n',
	'Ñ': 'N', 'Ń': 'N', 'Ň': 'N',
	// o/O
	'ò': 'o', 'ó': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o', 'ø': 'o', 'ō': 'o',
	'Ò': 'O', 'Ó': 'O', 'Ô': 'O', 'Õ': 'O', 'Ö': 'O', 'Ø': 'O', 'Ō': 'O',
	// r/R
	'ř': 'r',
	'Ř': 'R',
	// s/S
	'ś': 's', 'š': 's', 'ș': 's',
	'Ś': 'S', 'Š': 'S', 'Ș': 'S',
	// t/T
	'ť': 't', 'ț': 't',
	'Ť': 'T', 'Ț': 'T',
	// u/U
	'ù': 'u', 'ú': 'u', 'û': 'u', 'ü': 'u', 'ū': 'u', 'ů': 'u', 'ų': 'u',
	'Ù': 'U', 'Ú': 'U', 'Û': 'U', 'Ü': 'U', 'Ū': 'U', 'Ů': 'U', 'Ų': 'U',
	// y/Y
	'ý': 'y', 'ÿ': 'y',
	'Ý': 'Y', 'Ÿ': 'Y',
	// z/Z
	'ź': 'z', 'ž': 'z', 'ż': 'z',
	'Ź': 'Z', 'Ž': 'Z', 'Ż': 'Z',
	// special characters
	'æ': 'a', // Could also be "ae"
	'Æ': 'A', // Could also be "AE"
	'œ': 'o', // Could also be "oe"
	'Œ': 'O', // Could also be "OE"
	'ß': 's', // Could also be "ss"
}

// normalizeDiacritic attempts to convert a Unicode diacritic to its ASCII equivalent.
// Returns true if normalization was applied, false if character should be handled elsewhere.
func normalizeDiacritic(r rune) (rune, bool) {
	if normalized, ok := diacriticMap[r]; ok {
		return normalized, true
	}
	return r, false
}

// generateSuffix creates a random alphanumeric suffix of the specified length.
func generateSuffix(length int, lowercase bool) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	const charsUpper = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	charset := chars
	if !lowercase {
		charset = charsUpper
	}

	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// Fallback to deterministic suffix on rand.Read failure
		for i := range b {
			b[i] = charset[i%len(charset)]
		}
		return string(b)
	}

	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}

	return string(b)
}
