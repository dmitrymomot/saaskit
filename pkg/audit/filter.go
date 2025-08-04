package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// FilterAction defines the action to take on matched metadata fields
type FilterAction string

const (
	FilterActionRemove FilterAction = "remove"
	FilterActionHash   FilterAction = "hash"
	FilterActionMask   FilterAction = "mask"
)

// FilterRule defines how to filter a specific metadata field
type FilterRule struct {
	Action FilterAction
}

// MetadataFilter provides configurable filtering for sensitive data in audit events
type MetadataFilter struct {
	customFilters map[string]FilterRule
	allowedFields map[string]bool
	filterPII     bool
}

// Default PII fields that should be filtered automatically
var defaultPIIFields = map[string]FilterRule{
	"password":               {Action: FilterActionRemove},
	"pass":                   {Action: FilterActionRemove},
	"pwd":                    {Action: FilterActionRemove},
	"secret":                 {Action: FilterActionRemove},
	"token":                  {Action: FilterActionRemove},
	"api_key":                {Action: FilterActionRemove},
	"apikey":                 {Action: FilterActionRemove},
	"access_token":           {Action: FilterActionRemove},
	"refresh_token":          {Action: FilterActionRemove},
	"private_key":            {Action: FilterActionRemove},
	"privatekey":             {Action: FilterActionRemove},
	"ssn":                    {Action: FilterActionMask},
	"social_security_number": {Action: FilterActionMask},
	"credit_card":            {Action: FilterActionMask},
	"creditcard":             {Action: FilterActionMask},
	"card_number":            {Action: FilterActionMask},
	"cardnumber":             {Action: FilterActionMask},
	"cvv":                    {Action: FilterActionRemove},
	"cvc":                    {Action: FilterActionRemove},
	"email":                  {Action: FilterActionHash},
	"phone":                  {Action: FilterActionMask},
	"phone_number":           {Action: FilterActionMask},
	"phonenumber":            {Action: FilterActionMask},
	"date_of_birth":          {Action: FilterActionHash},
	"dateofbirth":            {Action: FilterActionHash},
	"dob":                    {Action: FilterActionHash},
}

// FilterOption configures MetadataFilter behavior
type FilterOption func(*MetadataFilter)

// NewMetadataFilter creates a new metadata filter with default PII filtering enabled
func NewMetadataFilter(opts ...FilterOption) *MetadataFilter {
	f := &MetadataFilter{
		customFilters: make(map[string]FilterRule),
		allowedFields: make(map[string]bool),
		filterPII:     true,
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithCustomField adds a custom field filter rule
func WithCustomField(field string, action FilterAction) FilterOption {
	return func(f *MetadataFilter) {
		f.customFilters[strings.ToLower(field)] = FilterRule{Action: action}
	}
}

// WithAllowedField explicitly allows a field to pass through without filtering
func WithAllowedField(field string) FilterOption {
	return func(f *MetadataFilter) {
		f.allowedFields[strings.ToLower(field)] = true
	}
}

// WithoutPIIDefaults disables default PII field filtering
func WithoutPIIDefaults() FilterOption {
	return func(f *MetadataFilter) {
		f.filterPII = false
	}
}

// Filter applies filtering rules to the provided metadata map
func (f *MetadataFilter) Filter(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}

	filtered := make(map[string]any)

	for key, value := range metadata {
		lowerKey := strings.ToLower(key)

		// Check if field is explicitly allowed
		if f.allowedFields[lowerKey] {
			filtered[key] = value
			continue
		}

		// Check custom filters first
		if rule, ok := f.customFilters[lowerKey]; ok {
			if result := f.applyRule(rule, value); result != nil {
				filtered[key] = result
			}
			continue
		}

		// Check wildcard patterns in custom filters
		if rule := f.matchWildcard(lowerKey, f.customFilters); rule != nil {
			if result := f.applyRule(*rule, value); result != nil {
				filtered[key] = result
			}
			continue
		}

		// Check default PII filters if enabled
		if f.filterPII {
			if rule, ok := defaultPIIFields[lowerKey]; ok {
				if result := f.applyRule(rule, value); result != nil {
					filtered[key] = result
				}
				continue
			}

			// Check wildcard patterns in default PII filters
			if rule := f.matchWildcard(lowerKey, defaultPIIFields); rule != nil {
				if result := f.applyRule(*rule, value); result != nil {
					filtered[key] = result
				}
				continue
			}
		}

		// No filter matched, include the field as-is
		filtered[key] = value
	}

	return filtered
}

// matchWildcard checks if the key matches any wildcard patterns in the rules
func (f *MetadataFilter) matchWildcard(key string, rules map[string]FilterRule) *FilterRule {
	for pattern, rule := range rules {
		if strings.Contains(pattern, "*") {
			if matchesPattern(key, pattern) {
				return &rule
			}
		}
	}
	return nil
}

// matchesPattern checks if a key matches a wildcard pattern
func matchesPattern(key, pattern string) bool {
	pattern = strings.ToLower(pattern)

	// Simple wildcard matching: *.suffix or prefix.*
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		return strings.HasSuffix(key, suffix)
	}

	if strings.HasSuffix(pattern, ".*") {
		prefix := pattern[:len(pattern)-2]
		return strings.HasPrefix(key, prefix)
	}

	// For patterns like *password*, check if key contains the pattern
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		contains := pattern[1 : len(pattern)-1]
		return strings.Contains(key, contains)
	}

	return false
}

// applyRule applies a filter rule to a value
func (f *MetadataFilter) applyRule(rule FilterRule, value any) any {
	switch rule.Action {
	case FilterActionRemove:
		return nil
	case FilterActionHash:
		return f.hashValue(value)
	case FilterActionMask:
		return f.maskValue(value)
	default:
		return value
	}
}

// hashValue creates a SHA256 hash of the value
func (f *MetadataFilter) hashValue(value any) string {
	str := fmt.Sprintf("%v", value)
	hash := sha256.Sum256([]byte(str))
	return hex.EncodeToString(hash[:])
}

// maskValue partially hides the value, showing only first and last few characters
func (f *MetadataFilter) maskValue(value any) string {
	str := fmt.Sprintf("%v", value)
	length := len(str)

	if length <= 4 {
		return strings.Repeat("*", length)
	}

	if length <= 8 {
		return str[:1] + strings.Repeat("*", length-2) + str[length-1:]
	}

	// For longer strings, show first 2 and last 2 characters
	return str[:2] + strings.Repeat("*", length-4) + str[length-2:]
}
