package validator

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	slugRegex      = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	usernameRegex  = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	handleRegex    = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	skuRegex       = regexp.MustCompile(`^[A-Z0-9-]+$`)
	hexStringRegex = regexp.MustCompile(`^[0-9A-Fa-f]+$`)
	base64Regex    = regexp.MustCompile(`^[A-Za-z0-9+/]*={0,2}$`)
	subdomainRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]*$`)
)

// ValidSlug validates URL-safe slugs, preventing edge cases like leading/trailing hyphens.
func ValidSlug(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			return slugRegex.MatchString(value) && !strings.HasPrefix(value, "-") && !strings.HasSuffix(value, "-")
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid slug (lowercase letters, numbers, and hyphens only)",
			TranslationKey: "validation.slug",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func ValidUsername(field, value string, minLen int, maxLen int) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			if len(value) < minLen || len(value) > maxLen {
				return false
			}
			return usernameRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("username must be %d-%d characters long and contain only letters, numbers, underscores, and hyphens", minLen, maxLen),
			TranslationKey: "validation.username",
			TranslationValues: map[string]any{
				"field":   field,
				"min_len": minLen,
				"max_len": maxLen,
			},
		},
	}
}

// ValidHandle validates that a string is a valid handle (starts with letter, then letters/numbers/underscore/hyphen).
func ValidHandle(field, value string, minLen int, maxLen int) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			if len(value) < minLen || len(value) > maxLen {
				return false
			}
			return handleRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("handle must be %d-%d characters, start with a letter, and contain only letters, numbers, underscores, and hyphens", minLen, maxLen),
			TranslationKey: "validation.handle",
			TranslationValues: map[string]any{
				"field":   field,
				"min_len": minLen,
				"max_len": maxLen,
			},
		},
	}
}

// ValidSKU validates that a string is a valid SKU (Stock Keeping Unit).
func ValidSKU(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			// SKU should be 3-50 characters, uppercase letters, numbers, and hyphens
			if len(value) < 3 || len(value) > 50 {
				return false
			}
			return skuRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "SKU must be 3-50 characters and contain only uppercase letters, numbers, and hyphens",
			TranslationKey: "validation.sku",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidProductCode validates that a string is a valid product code.
func ValidProductCode(field, value string, pattern string) Rule {
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
			Message:        "invalid product code format",
			TranslationKey: "validation.product_code",
			TranslationValues: map[string]any{
				"field":   field,
				"pattern": pattern,
			},
		},
	}
}

// ValidHexString validates that a string contains only hexadecimal characters.
func ValidHexString(field, value string, exactLength int) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			if exactLength > 0 && len(value) != exactLength {
				return false
			}
			return hexStringRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be a valid hexadecimal string of length %d", exactLength),
			TranslationKey: "validation.hex_string",
			TranslationValues: map[string]any{
				"field":  field,
				"length": exactLength,
			},
		},
	}
}

// ValidBase64 validates that a string is valid base64 encoding.
func ValidBase64(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			// Base64 string length must be multiple of 4
			if len(value)%4 != 0 {
				return false
			}
			return base64Regex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid base64 encoded string",
			TranslationKey: "validation.base64",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidCustomID validates that a string matches a custom identifier pattern.
func ValidCustomID(field, value string, pattern string, description string) Rule {
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
			Message:        fmt.Sprintf("invalid %s format", description),
			TranslationKey: "validation.custom_id",
			TranslationValues: map[string]any{
				"field":       field,
				"pattern":     pattern,
				"description": description,
			},
		},
	}
}

// ValidDomainName validates that a string is a valid domain name.
func ValidDomainName(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			// Domain name should be 1-253 characters
			if len(value) > 253 {
				return false
			}

			// Split into labels
			labels := strings.Split(value, ".")
			if len(labels) < 2 {
				return false
			}

			// Validate each label
			for i, label := range labels {
				// Each label should be 1-63 characters
				if len(label) == 0 || len(label) > 63 {
					return false
				}

				// Labels cannot start or end with hyphen
				if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
					return false
				}

				// Labels must contain only alphanumeric and hyphens
				for _, char := range label {
					//nolint:staticcheck // More readable than De Morgan's law
					if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
						(char >= '0' && char <= '9') || char == '-') {
						return false
					}
				}

				// TLD (last label) must be at least 2 characters and only letters
				if i == len(labels)-1 {
					if len(label) < 2 {
						return false
					}
					for _, char := range label {
						//nolint:staticcheck // More readable than De Morgan's law
						if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')) {
							return false
						}
					}
				}
			}

			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid domain name",
			TranslationKey: "validation.domain_name",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidSubdomain validates that a string is a valid subdomain.
func ValidSubdomain(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			// Subdomain should be 1-63 characters
			if len(value) > 63 {
				return false
			}

			// Cannot start or end with hyphen
			if strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
				return false
			}

			return subdomainRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid subdomain (1-63 characters, letters, numbers, and hyphens)",
			TranslationKey: "validation.subdomain",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidAPIKey validates that a string looks like a valid API key.
func ValidAPIKey(field, value string, minLength int, maxLength int) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			if len(value) < minLength || len(value) > maxLength {
				return false
			}
			// API keys are typically alphanumeric with some special characters
			return regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("API key must be %d-%d characters and contain only letters, numbers, underscores, and hyphens", minLength, maxLength),
			TranslationKey: "validation.api_key",
			TranslationValues: map[string]any{
				"field":      field,
				"min_length": minLength,
				"max_length": maxLength,
			},
		},
	}
}

// ValidTicketNumber validates that a string is a valid ticket/reference number.
func ValidTicketNumber(field, value string, prefix string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			if prefix != "" && !strings.HasPrefix(value, prefix) {
				return false
			}
			// After prefix, should be alphanumeric
			remaining := value
			if prefix != "" {
				remaining = strings.TrimPrefix(value, prefix)
			}
			return regexp.MustCompile(`^[A-Z0-9]+$`).MatchString(remaining)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("invalid ticket number format (expected format: %s + alphanumeric)", prefix),
			TranslationKey: "validation.ticket_number",
			TranslationValues: map[string]any{
				"field":  field,
				"prefix": prefix,
			},
		},
	}
}

// ValidVersion validates that a string is a valid semantic version.
func ValidVersion(field, value string) Rule {
	// Semantic versioning pattern: MAJOR.MINOR.PATCH with optional pre-release and build metadata
	versionRegex := regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			return versionRegex.MatchString(value)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid semantic version (e.g., 1.2.3, 1.0.0-alpha.1)",
			TranslationKey: "validation.version",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}
