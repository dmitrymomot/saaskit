package sanitizer

import (
	"net/url"
	"regexp"
	"strings"
)

// NormalizeEmail sanitizes an email address by trimming whitespace,
// converting to lowercase, and removing extra dots in the local part.
func NormalizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)

	// Split email into local and domain parts
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email // Return as-is if not a valid email format
	}

	local := parts[0]
	domain := parts[1]

	// Remove consecutive dots in local part
	re := regexp.MustCompile(`\.+`)
	local = re.ReplaceAllString(local, ".")

	// Remove leading/trailing dots in local part
	local = strings.Trim(local, ".")

	return local + "@" + domain
}

// ExtractEmailDomain extracts the domain part from an email address.
// Returns empty string if email format is invalid.
func ExtractEmailDomain(email string) string {
	email = strings.TrimSpace(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// MaskEmail masks an email address for privacy, showing only first character
// of local part and full domain.
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

// NormalizePhone removes all non-digit characters from a phone number.
func NormalizePhone(phone string) string {
	re := regexp.MustCompile(`\D`)
	return re.ReplaceAllString(phone, "")
}

// FormatPhoneUS formats a phone number as a US phone number (XXX) XXX-XXXX.
// Input should be 10 digits. Returns original string if not 10 digits.
func FormatPhoneUS(phone string) string {
	digits := NormalizePhone(phone)
	if len(digits) != 10 {
		return phone
	}

	return "(" + digits[0:3] + ") " + digits[3:6] + "-" + digits[6:10]
}

// MaskPhone masks a phone number, showing only the last 4 digits.
func MaskPhone(phone string) string {
	digits := NormalizePhone(phone)
	if len(digits) < 4 {
		return strings.Repeat("*", len(digits))
	}

	masked := strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
	return masked
}

// ExtractPhoneDigits is an alias for NormalizePhone for clarity.
func ExtractPhoneDigits(phone string) string {
	return NormalizePhone(phone)
}

// NormalizeURL sanitizes a URL by ensuring it has a protocol,
// trimming whitespace, and normalizing the format.
func NormalizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	// Add protocol if missing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	// Parse and reconstruct URL to normalize
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // Return original if can't parse
	}

	// Normalize host to lowercase
	parsedURL.Host = strings.ToLower(parsedURL.Host)

	// Remove trailing slash from path if it's just "/"
	if parsedURL.Path == "/" {
		parsedURL.Path = ""
	}

	return parsedURL.String()
}

// ExtractDomain extracts the domain from a URL.
func ExtractDomain(rawURL string) string {
	// Add protocol if missing for parsing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	host := strings.ToLower(parsedURL.Host)

	// Return empty if host is invalid (empty, just colon, etc.)
	if host == "" || host == ":" {
		return ""
	}

	return host
}

// RemoveQueryParams removes query parameters from a URL.
func RemoveQueryParams(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	parsedURL.RawQuery = ""
	return parsedURL.String()
}

// RemoveFragment removes the fragment (hash) from a URL.
func RemoveFragment(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	parsedURL.Fragment = ""
	return parsedURL.String()
}

// NormalizeCreditCard removes all non-digit characters from a credit card number.
func NormalizeCreditCard(cardNumber string) string {
	re := regexp.MustCompile(`\D`)
	return re.ReplaceAllString(cardNumber, "")
}

// MaskCreditCard masks a credit card number, showing only the last 4 digits.
func MaskCreditCard(cardNumber string) string {
	digits := NormalizeCreditCard(cardNumber)
	if len(digits) < 4 {
		return strings.Repeat("*", len(digits))
	}

	masked := strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
	return masked
}

// FormatCreditCard formats a credit card number with spaces every 4 digits.
// Returns original string if not 13-19 digits (common card lengths).
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

// NormalizeSSN removes all non-digit characters from a Social Security Number.
func NormalizeSSN(ssn string) string {
	re := regexp.MustCompile(`\D`)
	return re.ReplaceAllString(ssn, "")
}

// MaskSSN masks a Social Security Number, showing only the last 4 digits.
func MaskSSN(ssn string) string {
	digits := NormalizeSSN(ssn)
	if len(digits) < 4 {
		return strings.Repeat("*", len(digits))
	}

	masked := strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
	return masked
}

// FormatSSN formats a Social Security Number as XXX-XX-XXXX.
// Input should be 9 digits. Returns original string if not 9 digits.
func FormatSSN(ssn string) string {
	digits := NormalizeSSN(ssn)
	if len(digits) != 9 {
		return ssn
	}

	return digits[0:3] + "-" + digits[3:5] + "-" + digits[5:9]
}

// NormalizePostalCode normalizes a postal code by removing spaces and converting to uppercase.
func NormalizePostalCode(postalCode string) string {
	code := strings.TrimSpace(postalCode)
	code = strings.ReplaceAll(code, " ", "")
	return strings.ToUpper(code)
}

// FormatPostalCodeUS formats a US ZIP code. Supports 5-digit and 9-digit formats.
func FormatPostalCodeUS(postalCode string) string {
	digits := regexp.MustCompile(`\D`).ReplaceAllString(postalCode, "")

	switch len(digits) {
	case 5:
		return digits
	case 9:
		return digits[0:5] + "-" + digits[5:9]
	default:
		return postalCode
	}
}

// FormatPostalCodeCA formats a Canadian postal code as A1A 1A1.
func FormatPostalCodeCA(postalCode string) string {
	// Remove spaces and convert to uppercase
	code := NormalizePostalCode(postalCode)

	// Canadian postal codes are 6 characters: letter-digit-letter digit-letter-digit
	if len(code) != 6 {
		return postalCode
	}

	return code[0:3] + " " + code[3:6]
}

// MaskString masks a string showing only the first and last characters,
// with the middle replaced by asterisks.
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

// RemoveNonAlphanumeric removes all characters that are not letters or digits.
func RemoveNonAlphanumeric(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]`)
	return re.ReplaceAllString(s, "")
}

// NormalizeWhitespace replaces any whitespace characters with single spaces and trims.
func NormalizeWhitespace(s string) string {
	re := regexp.MustCompile(`\s+`)
	normalized := re.ReplaceAllString(s, " ")
	return strings.TrimSpace(normalized)
}

// ExtractNumbers extracts all numeric sequences from a string.
func ExtractNumbers(s string) string {
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(s, -1)
	return strings.Join(matches, "")
}

// SanitizeFilename sanitizes a filename by removing or replacing unsafe characters.
func SanitizeFilename(filename string) string {
	// Replace unsafe characters with underscores
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	safe := re.ReplaceAllString(filename, "_")

	// Remove leading/trailing spaces and dots
	safe = strings.Trim(safe, " .")

	// Limit length (filesystem limits)
	if len(safe) > 255 {
		safe = safe[:255]
	}

	// Ensure it's not empty
	if safe == "" {
		safe = "file"
	}

	return safe
}
