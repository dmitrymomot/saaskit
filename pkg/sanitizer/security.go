package sanitizer

import (
	"html"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// EscapeHTML escapes HTML special characters to prevent XSS attacks.
func EscapeHTML(s string) string {
	return html.EscapeString(s)
}

// UnescapeHTML unescapes HTML entities.
func UnescapeHTML(s string) string {
	return html.UnescapeString(s)
}

// StripScriptTags removes all <script> tags and their content.
func StripScriptTags(s string) string {
	re := regexp.MustCompile(`(?i)<script\b[^>]*>.*?</script>`)
	return re.ReplaceAllString(s, "")
}

// RemoveJavaScriptEvents removes JavaScript event handlers from HTML attributes.
func RemoveJavaScriptEvents(s string) string {
	// Remove on* event handlers (onclick, onload, etc.)
	re := regexp.MustCompile(`(?i)\s*on\w+\s*=\s*("[^"]*"|'[^']*')`)
	result := re.ReplaceAllString(s, "")

	// Remove javascript: protocols
	re = regexp.MustCompile(`(?i)javascript\s*:`)
	return re.ReplaceAllString(result, "")
}

// SanitizeHTMLAttributes removes potentially dangerous HTML attributes.
func SanitizeHTMLAttributes(s string) string {
	// Remove dangerous attributes
	dangerous := []string{
		`(?i)\s*onclick\s*=\s*["'][^"']*["']`,
		`(?i)\s*onload\s*=\s*["'][^"']*["']`,
		`(?i)\s*onerror\s*=\s*["'][^"']*["']`,
		`(?i)\s*onmouseover\s*=\s*["'][^"']*["']`,
		`(?i)\s*onfocus\s*=\s*["'][^"']*["']`,
		`(?i)\s*onblur\s*=\s*["'][^"']*["']`,
		`(?i)\s*style\s*=\s*["'][^"']*expression[^"']*["']`,
		`(?i)\s*href\s*=\s*["']javascript:[^"']*["']`,
		`(?i)\s*src\s*=\s*["']javascript:[^"']*["']`,
	}

	result := s
	for _, pattern := range dangerous {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")
	}

	return result
}

// PreventXSS applies comprehensive XSS prevention measures.
func PreventXSS(s string) string {
	result := s
	result = StripScriptTags(result)
	result = RemoveJavaScriptEvents(result)
	result = SanitizeHTMLAttributes(result)
	result = EscapeHTML(result)
	return result
}

// EscapeSQLString escapes single quotes in SQL strings to prevent injection.
func EscapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// RemoveSQLKeywords removes common SQL keywords that could be used for injection.
func RemoveSQLKeywords(s string) string {
	keywords := []string{
		`(?i)\bSELECT\b`, `(?i)\bINSERT\b`, `(?i)\bUPDATE\b`, `(?i)\bDELETE\b`,
		`(?i)\bDROP\b`, `(?i)\bTABLE\b`, `(?i)\bCREATE\b`, `(?i)\bALTER\b`, `(?i)\bTRUNCATE\b`,
		`(?i)\bEXEC\b`, `(?i)\bEXECUTE\b`, `(?i)\bUNION\b`, `(?i)\bJOIN\b`,
		`(?i)\bWHERE\b`, `(?i)\bHAVING\b`, `(?i)\bORDER\s+BY\b`, `(?i)\bGROUP\s+BY\b`,
		`(?i)\bINTO\b`, `(?i)\bVALUES\b`, `(?i)\bFROM\b`, `(?i)\bSET\b`,
		`(?i)\bSCRIPT\b`, `(?i)\bDATA\b`, `(?i)\bSCHEMA\b`,
	}

	result := s
	for _, keyword := range keywords {
		re := regexp.MustCompile(keyword)
		result = re.ReplaceAllString(result, "")
	}

	return result
}

// SanitizeSQLIdentifier ensures SQL identifiers (table names, column names) are safe.
func SanitizeSQLIdentifier(s string) string {
	// Keep only alphanumeric and underscore
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	result := re.ReplaceAllString(s, "")

	// Ensure it doesn't start with a number
	if len(result) > 0 && unicode.IsDigit(rune(result[0])) {
		result = "_" + result
	}

	// Limit length
	if len(result) > 64 {
		result = result[:64]
	}

	return result
}

// PreventPathTraversal removes path traversal attempts (../ and ..\).
func PreventPathTraversal(path string) string {
	// Remove any ../ or ..\
	re := regexp.MustCompile(`\.\.[\\/]`)
	result := re.ReplaceAllString(path, "")

	// Remove any remaining .. at the end
	result = strings.ReplaceAll(result, "..", "")

	return result
}

// SanitizePath cleans and normalizes file paths to prevent directory traversal.
func SanitizePath(path string) string {
	// Clean the path
	cleaned := filepath.Clean(path)

	// Remove any path traversal attempts
	cleaned = PreventPathTraversal(cleaned)

	// Remove any drive letters on Windows (C:, D:, etc.)
	re := regexp.MustCompile(`^[a-zA-Z]:`)
	cleaned = re.ReplaceAllString(cleaned, "")

	// Ensure it doesn't start with / or \ (after drive letter removal)
	cleaned = strings.TrimPrefix(cleaned, "/")
	cleaned = strings.TrimPrefix(cleaned, "\\")

	// Normalize path separators to forward slashes
	cleaned = filepath.ToSlash(cleaned)

	return cleaned
}

// NormalizePath normalizes a file path and prevents traversal attacks.
func NormalizePath(path string) string {
	// Normalize path separators
	normalized := filepath.ToSlash(path)

	// Apply sanitization
	normalized = SanitizePath(normalized)

	return normalized
}

// SanitizeShellArgument makes a string safe for use as a shell argument.
func SanitizeShellArgument(arg string) string {
	// Remove shell metacharacters
	dangerous := []string{
		"|", "&", ";", "$", "`", "\\", "\"", "'", " ", "\t", "\n", "\r",
		"*", "?", "[", "]", "(", ")", "{", "}", "<", ">", "^", "!",
	}

	result := arg
	for _, char := range dangerous {
		result = strings.ReplaceAll(result, char, "")
	}

	return result
}

// RemoveShellMetacharacters removes shell metacharacters that could be used for injection.
func RemoveShellMetacharacters(s string) string {
	// Remove characters that have special meaning in shells
	re := regexp.MustCompile(`[|&;$\x60\\<>^!\*\?\[\]\(\)\{\}]`)
	return re.ReplaceAllString(s, "")
}

// RemoveNullBytes removes null bytes that could cause issues in C-based systems.
func RemoveNullBytes(s string) string {
	return strings.ReplaceAll(s, "\x00", "")
}

// RemoveControlSequences removes ANSI escape sequences and other control characters.
func RemoveControlSequences(s string) string {
	// Remove ANSI escape sequences
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	result := re.ReplaceAllString(s, "")

	// Remove other control characters except common ones
	result = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, result)

	return result
}

// LimitLength truncates input to prevent DoS attacks through large inputs.
func LimitLength(s string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= maxLength {
		return s
	}

	return string(runes[:maxLength])
}

// SanitizeUserInput applies comprehensive sanitization for user input.
func SanitizeUserInput(s string) string {
	result := s
	result = RemoveNullBytes(result)
	result = RemoveControlSequences(result)
	result = strings.TrimSpace(result)
	result = LimitLength(result, 10000) // Reasonable default limit
	return result
}

// PreventLDAPInjection removes LDAP injection characters.
func PreventLDAPInjection(s string) string {
	// Remove LDAP special characters
	dangerous := []string{"(", ")", "*", "\\", "/", "\x00"}

	result := s
	for _, char := range dangerous {
		result = strings.ReplaceAll(result, char, "")
	}

	return result
}

// SanitizeEmail removes dangerous characters from email addresses while preserving valid format.
func SanitizeEmail(email string) string {
	// Remove null bytes and control characters
	result := RemoveNullBytes(email)
	result = RemoveControlSequences(result)

	// Remove potential XSS attempts
	result = strings.ReplaceAll(result, "<", "")
	result = strings.ReplaceAll(result, ">", "")
	result = strings.ReplaceAll(result, "\"", "")
	result = strings.ReplaceAll(result, "'", "")

	return strings.TrimSpace(result)
}

// SanitizeURL removes dangerous elements from URLs while preserving valid structure.
func SanitizeURL(url string) string {
	result := url

	// Remove dangerous protocols
	dangerous := []string{
		"javascript:", "data:", "vbscript:", "file:", "ftp:",
	}

	lower := strings.ToLower(result)
	for _, protocol := range dangerous {
		if strings.HasPrefix(lower, protocol) {
			return ""
		}
	}

	// Remove potential XSS
	result = RemoveJavaScriptEvents(result)
	result = strings.ReplaceAll(result, "<", "")
	result = strings.ReplaceAll(result, ">", "")

	return strings.TrimSpace(result)
}

// PreventHeaderInjection removes characters that could be used for HTTP header injection.
func PreventHeaderInjection(s string) string {
	// Remove line breaks that could split headers
	result := strings.ReplaceAll(s, "\r", "")
	result = strings.ReplaceAll(result, "\n", "")

	// Remove null bytes
	result = RemoveNullBytes(result)

	return result
}

// SanitizeSecureFilename makes a filename safe by removing dangerous characters.
func SanitizeSecureFilename(filename string) string {
	// Remove path separators and dangerous characters
	dangerous := []string{
		"/", "\\", ":", "*", "?", "\"", "<", ">", "|",
		"\x00", "\r", "\n", "\t",
	}

	result := filename
	for _, char := range dangerous {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Remove leading/trailing spaces and dots
	result = strings.Trim(result, " .")

	// Limit length
	result = LimitLength(result, 255)

	// Ensure it's not empty
	if result == "" {
		result = "file"
	}

	return result
}
