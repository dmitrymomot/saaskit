package sanitizer

import "regexp"

// Pre-compiled regular expressions for performance
var (
	// Email and general formatting
	dotRegex = regexp.MustCompile(`\.+`)

	// Phone and numeric extraction
	nonDigitRegex = regexp.MustCompile(`\D`)
	digitRegex    = regexp.MustCompile(`\d+`)

	// Whitespace normalization
	whitespaceRegex = regexp.MustCompile(`\s+`)

	// Alphanumeric filtering
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

	// HTML stripping
	htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

	// Filename sanitization
	unsafeFilenameRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
)
