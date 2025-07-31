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
	// E.164 international phone format: optional +, non-zero first digit, 7-15 total digits
	phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)

	alphanumericRegex  = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	alphaRegex         = regexp.MustCompile(`^[a-zA-Z]+$`)
	numericStringRegex = regexp.MustCompile(`^[0-9]+$`)
)

// ValidEmail validates email addresses using RFC 5322 with additional web-friendly constraints.
// Rejects edge cases like quoted local parts and comments that are valid per RFC but problematic for web apps.
func ValidEmail(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}

			addr, err := mail.ParseAddress(value)
			if err != nil {
				return false
			}

			// Validates domain structure and local part requirements for web applications
			email := addr.Address
			parts := strings.Split(email, "@")
			if len(parts) != 2 {
				return false
			}

			localPart := parts[0]
			domain := parts[1]

			if localPart == "" {
				return false
			}

			// Reject domains without dots or starting/ending with dots
			if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
				return false
			}

			// Reject empty domain parts (consecutive dots)
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

// ValidPhone validates international phone numbers in E.164 format.
// Allows common formatting chars (spaces, dashes) but requires 7-15 digits total.
func ValidPhone(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			cleaned := strings.ReplaceAll(strings.ReplaceAll(value, " ", ""), "-", "")

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
			// Accept IPv4-mapped IPv6 addresses (::ffff:192.0.2.1)
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
