package sanitizer

import (
	"net/url"
	"strings"
)

// NormalizeEmail prevents common email input errors but preserves original for invalid formats.
// Consolidates consecutive dots which can cause delivery issues with some email providers.
func NormalizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	local := parts[0]
	domain := parts[1]

	// Consolidate consecutive dots to prevent delivery failures
	local = dotRegex.ReplaceAllString(local, ".")
	local = strings.Trim(local, ".")

	return local + "@" + domain
}

func ExtractEmailDomain(email string) string {
	email = strings.TrimSpace(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// MaskEmail preserves full domain for user recognition while hiding personal info.
func MaskEmail(email string) string {
	email = strings.TrimSpace(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	local := parts[0]
	domain := parts[1]

	if len(local) == 0 {
		return email
	}

	if len(local) == 1 {
		return "*@" + domain
	}

	masked := string(local[0]) + strings.Repeat("*", len(local)-1)
	return masked + "@" + domain
}

// NormalizePhone strips formatting to enable consistent database storage and comparison.
func NormalizePhone(phone string) string {
	return nonDigitRegex.ReplaceAllString(phone, "")
}

// FormatPhoneUS enforces NANP format; preserves original if not 10 digits to avoid data loss.
func FormatPhoneUS(phone string) string {
	digits := NormalizePhone(phone)
	if len(digits) != 10 {
		return phone
	}

	return "(" + digits[0:3] + ") " + digits[3:6] + "-" + digits[6:10]
}

// MaskPhone follows PCI compliance pattern of showing last 4 digits for user recognition.
func MaskPhone(phone string) string {
	digits := NormalizePhone(phone)
	if len(digits) < 4 {
		return strings.Repeat("*", len(digits))
	}

	masked := strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
	return masked
}

func ExtractPhoneDigits(phone string) string {
	return NormalizePhone(phone)
}

// NormalizeURL assumes HTTPS for security; preserves original on parse errors to avoid data loss.
// Removes trailing slash for consistent URL comparison and caching.
func NormalizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	// Default to HTTPS for security
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	parsedURL.Host = strings.ToLower(parsedURL.Host)

	// Remove trailing slash for consistent comparison
	if parsedURL.Path == "/" {
		parsedURL.Path = ""
	}

	return parsedURL.String()
}

// ExtractDomain assumes HTTPS protocol for parsing; validates host to prevent invalid domains.
func ExtractDomain(rawURL string) string {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	host := strings.ToLower(parsedURL.Host)

	// Validate host format
	if host == "" || host == ":" {
		return ""
	}

	return host
}

// RemoveQueryParams useful for URL comparison and preventing tracking parameter leakage.
func RemoveQueryParams(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	parsedURL.RawQuery = ""
	return parsedURL.String()
}

func RemoveFragment(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	parsedURL.Fragment = ""
	return parsedURL.String()
}

// NormalizeCreditCard strips formatting for PCI-compliant storage and validation.
func NormalizeCreditCard(cardNumber string) string {
	return nonDigitRegex.ReplaceAllString(cardNumber, "")
}

// MaskCreditCard follows PCI DSS requirement to show only last 4 digits.
func MaskCreditCard(cardNumber string) string {
	digits := NormalizeCreditCard(cardNumber)
	if len(digits) < 4 {
		return strings.Repeat("*", len(digits))
	}

	masked := strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
	return masked
}

// FormatCreditCard validates common card lengths (13-19 digits) before formatting for display.
func FormatCreditCard(cardNumber string) string {
	digits := NormalizeCreditCard(cardNumber)
	if len(digits) < 13 || len(digits) > 19 {
		return cardNumber
	}

	var formatted strings.Builder
	for i, digit := range digits {
		if i > 0 && i%4 == 0 {
			formatted.WriteString(" ")
		}
		formatted.WriteRune(digit)
	}

	return formatted.String()
}

// NormalizeSSN strips formatting for consistent storage and validation of sensitive data.
func NormalizeSSN(ssn string) string {
	return nonDigitRegex.ReplaceAllString(ssn, "")
}

// MaskSSN follows privacy regulations requiring masking of all but last 4 digits.
func MaskSSN(ssn string) string {
	digits := NormalizeSSN(ssn)
	if len(digits) < 4 {
		return strings.Repeat("*", len(digits))
	}

	masked := strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
	return masked
}

// FormatSSN enforces standard 9-digit format; preserves original if invalid to avoid data loss.
func FormatSSN(ssn string) string {
	digits := NormalizeSSN(ssn)
	if len(digits) != 9 {
		return ssn
	}

	return digits[0:3] + "-" + digits[3:5] + "-" + digits[5:9]
}

// NormalizePostalCode creates consistent format for database storage and comparison.
func NormalizePostalCode(postalCode string) string {
	code := strings.TrimSpace(postalCode)
	code = strings.ReplaceAll(code, " ", "")
	return strings.ToUpper(code)
}

// FormatPostalCodeUS handles both ZIP and ZIP+4 formats; preserves invalid input.
func FormatPostalCodeUS(postalCode string) string {
	digits := nonDigitRegex.ReplaceAllString(postalCode, "")

	switch len(digits) {
	case 5:
		return digits
	case 9:
		return digits[0:5] + "-" + digits[5:9]
	default:
		return postalCode
	}
}

// FormatPostalCodeCA enforces standard Canadian format (A1A 1A1); preserves invalid input.
func FormatPostalCodeCA(postalCode string) string {
	code := NormalizePostalCode(postalCode)

	// Canadian postal codes: letter-digit-letter digit-letter-digit
	if len(code) != 6 {
		return postalCode
	}

	return code[0:3] + " " + code[3:6]
}

// MaskString preserves start/end characters for user recognition while hiding sensitive middle.
// Handles Unicode properly and prevents over-masking short strings.
func MaskString(s string, visibleChars int) string {
	if visibleChars < 0 {
		visibleChars = 1
	}

	runes := []rune(s)
	length := len(runes)

	if length <= visibleChars*2 {
		return strings.Repeat("*", length)
	}

	visible := visibleChars
	if visible > length/2 {
		visible = length / 2
	}

	start := string(runes[0:visible])
	end := string(runes[length-visible:])
	middle := strings.Repeat("*", length-visible*2)

	return start + middle + end
}

func RemoveNonAlphanumeric(s string) string {
	return nonAlphanumericRegex.ReplaceAllString(s, "")
}

// NormalizeWhitespace prevents layout issues from multiple spaces, tabs, and newlines.
func NormalizeWhitespace(s string) string {
	normalized := whitespaceRegex.ReplaceAllString(s, " ")
	return strings.TrimSpace(normalized)
}

// ExtractNumbers concatenates all digit sequences, useful for ID extraction from mixed content.
func ExtractNumbers(s string) string {
	matches := digitRegex.FindAllString(s, -1)
	return strings.Join(matches, "")
}

// SanitizeFilename prevents filesystem vulnerabilities and ensures cross-platform compatibility.
// Enforces 255-byte limit and provides fallback for completely invalid names.
func SanitizeFilename(filename string) string {
	// Replace filesystem-unsafe characters
	safe := unsafeFilenameRegex.ReplaceAllString(filename, "_")

	// Remove problematic leading/trailing characters
	safe = strings.Trim(safe, " .")

	// Enforce filesystem length limits
	if len(safe) > 255 {
		safe = safe[:255]
	}

	// Prevent empty filenames
	if safe == "" {
		safe = "file"
	}

	return safe
}
