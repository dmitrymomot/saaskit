package validator

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ValidUUID validates that a string is a valid UUID format.
func ValidUUID(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}

			// UUID must be exactly 36 characters with hyphens in correct positions
			if len(value) != 36 {
				return false
			}

			// Check hyphen positions: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
			if value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
				return false
			}

			_, err := uuid.Parse(value)
			return err == nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid UUID",
			TranslationKey: "validation.uuid",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// NonNilUUID validates that a UUID is not uuid.Nil (all zeros).
func NonNilUUID(field string, value uuid.UUID) Rule {
	return Rule{
		Check: func() bool {
			return value != uuid.Nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "UUID cannot be nil",
			TranslationKey: "validation.uuid_not_nil",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// NonNilUUIDString validates that a UUID string is not the nil UUID representation.
func NonNilUUIDString(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}

			// First validate it's a proper UUID format
			if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
				return false
			}

			parsedUUID, err := uuid.Parse(value)
			if err != nil {
				return false
			}
			return parsedUUID != uuid.Nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "UUID cannot be nil",
			TranslationKey: "validation.uuid_not_nil",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidUUIDVersion validates that a UUID is of a specific version.
func ValidUUIDVersion(field string, value uuid.UUID, version int) Rule {
	return Rule{
		Check: func() bool {
			return value.Version() == uuid.Version(version)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be a UUID version %d", version),
			TranslationKey: "validation.uuid_version",
			TranslationValues: map[string]any{
				"field":   field,
				"version": version,
			},
		},
	}
}

// ValidUUIDVersionString validates that a UUID string is of a specific version.
func ValidUUIDVersionString(field, value string, version int) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}

			// First validate it's a proper UUID format
			if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
				return false
			}

			parsedUUID, err := uuid.Parse(value)
			if err != nil {
				return false
			}
			return parsedUUID.Version() == uuid.Version(version)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be a UUID version %d", version),
			TranslationKey: "validation.uuid_version",
			TranslationValues: map[string]any{
				"field":   field,
				"version": version,
			},
		},
	}
}

// ValidUUIDv1 validates that a UUID is version 1 (time-based).
func ValidUUIDv1(field string, value uuid.UUID) Rule {
	return ValidUUIDVersion(field, value, 1)
}

// ValidUUIDv1String validates that a UUID string is version 1 (time-based).
func ValidUUIDv1String(field, value string) Rule {
	return ValidUUIDVersionString(field, value, 1)
}

// ValidUUIDv3 validates that a UUID is version 3 (name-based using MD5).
func ValidUUIDv3(field string, value uuid.UUID) Rule {
	return ValidUUIDVersion(field, value, 3)
}

// ValidUUIDv3String validates that a UUID string is version 3 (name-based using MD5).
func ValidUUIDv3String(field, value string) Rule {
	return ValidUUIDVersionString(field, value, 3)
}

// ValidUUIDv4 validates that a UUID is version 4 (random).
func ValidUUIDv4(field string, value uuid.UUID) Rule {
	return ValidUUIDVersion(field, value, 4)
}

// ValidUUIDv4String validates that a UUID string is version 4 (random).
func ValidUUIDv4String(field, value string) Rule {
	return ValidUUIDVersionString(field, value, 4)
}

// ValidUUIDv5 validates that a UUID is version 5 (name-based using SHA-1).
func ValidUUIDv5(field string, value uuid.UUID) Rule {
	return ValidUUIDVersion(field, value, 5)
}

// ValidUUIDv5String validates that a UUID string is version 5 (name-based using SHA-1).
func ValidUUIDv5String(field, value string) Rule {
	return ValidUUIDVersionString(field, value, 5)
}

// RequiredUUID validates that a UUID is not the zero value and not nil.
func RequiredUUID(field string, value uuid.UUID) Rule {
	return Rule{
		Check: func() bool {
			return value != uuid.UUID{} && value != uuid.Nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "UUID is required",
			TranslationKey: "validation.required",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// RequiredUUIDString validates that a UUID string is not empty and represents a valid, non-nil UUID.
func RequiredUUIDString(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}

			// First validate it's a proper UUID format
			if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
				return false
			}

			parsedUUID, err := uuid.Parse(value)
			if err != nil {
				return false
			}
			return parsedUUID != uuid.Nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "UUID is required",
			TranslationKey: "validation.required",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}
