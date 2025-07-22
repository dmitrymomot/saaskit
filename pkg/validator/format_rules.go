package validator

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

var (
	// Phone number regex - international format with optional country code
	phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)

	// Alphanumeric regex
	alphanumericRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	// Alpha regex
	alphaRegex = regexp.MustCompile(`^[a-zA-Z]+$`)

	// Numeric string regex
	numericStringRegex = regexp.MustCompile(`^[0-9]+$`)
)

// ValidEmail validates that a string is a valid email address using RFC 5322.
func ValidEmail(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}

			// Parse with Go's mail parser first
			addr, err := mail.ParseAddress(value)
			if err != nil {
				return false
			}

			// Additional validation for typical web use
			email := addr.Address
			parts := strings.Split(email, "@")
			if len(parts) != 2 {
				return false
			}

			localPart := parts[0]
			domain := parts[1]

			// Local part cannot be empty
			if localPart == "" {
				return false
			}

			// Domain must contain at least one dot and cannot start/end with dot
			if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
				return false
			}

			// Domain parts cannot be empty
			for part := range strings.SplitSeq(domain, ".") {
				if part == "" {
					return false
				}
			}

			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid email address",
			TranslationKey: "validation.email",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidURL validates that a string is a valid URL.
func ValidURL(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}

			u, err := url.ParseRequestURI(value)
			if err != nil {
				return false
			}

			// Must have a scheme and host
			if u.Scheme == "" || u.Host == "" {
				return false
			}

			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid URL",
			TranslationKey: "validation.url",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidURLWithScheme validates that a string is a valid URL with a specific scheme.
func ValidURLWithScheme(field, value string, schemes []string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			u, err := url.ParseRequestURI(value)
			if err != nil {
				return false
			}
			return slices.Contains(schemes, u.Scheme)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be a valid URL with scheme: %s", strings.Join(schemes, ", ")),
			TranslationKey: "validation.url_scheme",
			TranslationValues: map[string]any{
				"field":   field,
				"schemes": schemes,
			},
		},
	}
}

// ValidPhone validates that a string is a valid international phone number.
// Accepts formats like +1234567890, +123456789012345 (E.164 format).
func ValidPhone(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			// Remove spaces and dashes for validation
			cleaned := strings.ReplaceAll(strings.ReplaceAll(value, " ", ""), "-", "")

			// Must be at least 7 digits (minimum valid phone number)
			if len(cleaned) < 7 {
				return false
			}

			return phoneRegex.MatchString(cleaned)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid phone number in international format",
			TranslationKey: "validation.phone",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidIPv4 validates that a string is a valid IPv4 address.
func ValidIPv4(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			ip := net.ParseIP(value)
			return ip != nil && ip.To4() != nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid IPv4 address",
			TranslationKey: "validation.ipv4",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidIPv6 validates that a string is a valid IPv6 address.
func ValidIPv6(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			ip := net.ParseIP(value)
			if ip == nil {
				return false
			}
			// IPv6 addresses can include IPv4-mapped addresses
			return ip.To4() == nil || strings.Contains(value, ":")
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid IPv6 address",
			TranslationKey: "validation.ipv6",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidIP validates that a string is a valid IP address (IPv4 or IPv6).
func ValidIP(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			return net.ParseIP(value) != nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid IP address",
			TranslationKey: "validation.ip",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidMAC validates that a string is a valid MAC address.
// Supports formats: AA:BB:CC:DD:EE:FF, AA-BB-CC-DD-EE-FF.
func ValidMAC(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			_, err := net.ParseMAC(value)
			return err == nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid MAC address",
			TranslationKey: "validation.mac",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidAlphanumeric validates that a string contains only letters and numbers.
func ValidAlphanumeric(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			return alphanumericRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must contain only letters and numbers",
			TranslationKey: "validation.alphanumeric",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidAlpha validates that a string contains only letters.
func ValidAlpha(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			return alphaRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must contain only letters",
			TranslationKey: "validation.alpha",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidNumericString validates that a string contains only digits.
func ValidNumericString(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			return numericStringRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must contain only digits",
			TranslationKey: "validation.numeric_string",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}
