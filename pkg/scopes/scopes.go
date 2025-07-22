package scopes

import (
	"slices"
	"sort"
	"strings"
)

var (
	// ScopeSeparator is used to separate multiple scopes in a string
	// This can be modified to use a different separator (e.g., ",")
	ScopeSeparator = " "

	// ScopeWildcard represents a wildcard scope that matches everything
	// This can be modified to use a different wildcard character (e.g., "?")
	ScopeWildcard = "*"

	// ScopeDelimiter is used to separate scope parts (e.g., "admin.read")
	// This can be modified to use a different delimiter (e.g., ":")
	ScopeDelimiter = "."
)

// ParseScopes converts a space-separated string of scopes into a string slice.
//
// It handles trimming of extra spaces and removes empty entries. Returns nil if the input is empty.
// Efficiently processes the input string to minimize allocations.
//
// Example:
//
//	scopes := scopes.ParseScopes("read write admin.users")
//	// Returns: []string{"read", "write", "admin.users"}
func ParseScopes(scopesStr string) []string {
	if scopesStr == "" {
		return nil
	}

	scopesStr = strings.TrimSpace(scopesStr)
	if scopesStr == "" {
		return nil
	}

	parts := strings.Split(scopesStr, ScopeSeparator)
	scopes := make([]string, 0, len(parts))

	for i := range parts {
		if parts[i] = strings.TrimSpace(parts[i]); parts[i] != "" {
			scopes = append(scopes, parts[i])
		}
	}

	return scopes
}

// JoinScopes converts a slice of scopes back to a space-separated string.
//
// Returns an empty string if the input slice is empty or nil.
//
// Example:
//
//	str := scopes.JoinScopes([]string{"read", "write", "admin.*"})
//	// Returns: "read write admin.*"
func JoinScopes(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}
	return strings.Join(scopes, ScopeSeparator)
}

// ScopeMatches checks if a single scope matches a pattern.
// It supports wildcards (*) and hierarchical scopes (scope1.scope2).
//
// Pattern matching rules:
// - Direct match: "read" matches "read"
// - Global wildcard: "*" matches any scope
// - Namespace wildcard: "admin.*" matches any scope starting with "admin."
func ScopeMatches(scope, pattern string) bool {
	// Direct match or full wildcard
	if scope == pattern || pattern == ScopeWildcard {
		return true
	}

	// Handle wildcard suffix (e.g., "admin.*")
	if strings.HasSuffix(pattern, ScopeWildcard) {
		prefix := strings.TrimSuffix(pattern, ScopeWildcard)
		prefix = strings.TrimSuffix(prefix, ScopeDelimiter)
		return strings.HasPrefix(scope, prefix+ScopeDelimiter)
	}

	return false
}

// HasScope checks if scopes contain a specific scope.
//
// Supports wildcards and hierarchical scope matching.
//
// Example:
//
//	hasScope := scopes.HasScope([]string{"admin.*", "read"}, "admin.users")
//	// Returns: true (because "admin.*" matches "admin.users")
func HasScope(scopes []string, scope string) bool {
	for _, s := range scopes {
		if ScopeMatches(scope, s) {
			return true
		}
	}
	return false
}

// hasWildcard checks if any scope in the collection is the global wildcard.
// Extracted as a helper function to reduce code duplication.
func hasWildcard(scopes []string) bool {
	return slices.Contains(scopes, ScopeWildcard)
}

// HasAllScopes checks if scopes contain all of the required scopes.
//
// Returns true if:
// - The required slice is empty
// - The scopes include a global wildcard "*"
// - Each scope in required is matched by at least one scope in scopes
//
// Example:
//
//	hasAll := scopes.HasAllScopes(
//	    []string{"admin.*", "read", "write"},
//	    []string{"admin.users", "read"}
//	)
//	// Returns: true
func HasAllScopes(scopes, required []string) bool {
	if len(required) == 0 {
		return true
	}
	if len(scopes) == 0 {
		return false
	}

	// Check for global wildcard in scopes
	if hasWildcard(scopes) {
		return true
	}

	for _, req := range required {
		if !HasScope(scopes, req) {
			return false
		}
	}
	return true
}

// HasAnyScopes checks if scopes contain any of the required scopes.
//
// Returns true if:
// - The required slice is empty
// - The scopes include a global wildcard "*"
// - At least one scope in required is matched by at least one scope in scopes
//
// Example:
//
//	hasAny := scopes.HasAnyScopes(
//	    []string{"read", "write"},
//	    []string{"delete", "read"}
//	)
//	// Returns: true
func HasAnyScopes(scopes, required []string) bool {
	if len(required) == 0 {
		return true
	}
	if len(scopes) == 0 {
		return false
	}

	// Check for global wildcard in scopes
	if hasWildcard(scopes) {
		return true
	}

	for _, req := range required {
		if HasScope(scopes, req) {
			return true
		}
	}
	return false
}

// EqualScopes checks if two scope collections are identical (same scopes, regardless of order).
//
// It sorts both collections before comparison to handle different ordering.
// This implementation minimizes memory allocations when possible.
//
// Example:
//
//	equal := scopes.EqualScopes(
//	    []string{"read", "write"},
//	    []string{"write", "read"}
//	)
//	// Returns: true
func EqualScopes(scopes1, scopes2 []string) bool {
	if len(scopes1) != len(scopes2) {
		return false
	}

	// For small slice sizes, sorting in-place is more efficient
	if len(scopes1) <= 4 {
		// Create copies to avoid modifying originals
		s1 := make([]string, len(scopes1))
		s2 := make([]string, len(scopes2))
		copy(s1, scopes1)
		copy(s2, scopes2)

		// Sort both copies
		sort.Strings(s1)
		sort.Strings(s2)

		// Compare sorted slices
		for i := range s1 {
			if s1[i] != s2[i] {
				return false
			}
		}
		return true
	}

	// For larger slices, use maps for O(n) comparison
	scopeMap := make(map[string]int, len(scopes1))
	for _, s := range scopes1 {
		scopeMap[s]++
	}

	for _, s := range scopes2 {
		count, exists := scopeMap[s]
		if !exists || count == 0 {
			return false
		}
		scopeMap[s]--
	}

	return true
}

// ValidateScopes checks if all scopes are valid according to the provided valid scopes.
//
// A scope is considered valid if it matches any of the validScopes (including wildcards).
// Empty scopes are always considered valid, but empty validScopes will cause validation to fail.
// Uses optimized validation strategy based on the size of the input collections.
//
// Example:
//
//	valid := scopes.ValidateScopes(
//	    []string{"admin.read", "user.write"},
//	    []string{"admin.*", "user.*", "system.*"}
//	)
//	// Returns: true
func ValidateScopes(scopes, validScopes []string) bool {
	if len(scopes) == 0 {
		return true
	}
	if len(validScopes) == 0 {
		return false
	}

	// Check for global wildcard in valid scopes
	if hasWildcard(validScopes) {
		return true
	}

	// For larger valid scopes collections, use map-based approach for better performance
	if len(validScopes) > 10 && len(scopes) > 5 {
		return validateScopesWithMap(scopes, validScopes)
	}

	// Standard approach for smaller collections
	for _, scope := range scopes {
		isValid := false
		for _, validScope := range validScopes {
			if ScopeMatches(scope, validScope) {
				isValid = true
				break
			}
		}
		if !isValid {
			return false
		}
	}
	return true
}

// validateScopesWithMap is an optimized validation approach for large collections
// that preprocesses valid scopes for faster lookup.
func validateScopesWithMap(scopes, validScopes []string) bool {
	// Build a map of exact matches for faster lookup
	exactMatches := make(map[string]struct{}, len(validScopes))
	wildcardPatterns := make([]string, 0, len(validScopes)/2)

	for _, vs := range validScopes {
		if strings.Contains(vs, ScopeWildcard) {
			wildcardPatterns = append(wildcardPatterns, vs)
		} else {
			exactMatches[vs] = struct{}{}
		}
	}

	// Validate each scope
	for _, scope := range scopes {
		// First check for exact match (O(1) operation)
		if _, ok := exactMatches[scope]; ok {
			continue
		}

		// Then check for wildcard patterns (more expensive)
		isValid := false
		for _, pattern := range wildcardPatterns {
			if ScopeMatches(scope, pattern) {
				isValid = true
				break
			}
		}

		if !isValid {
			return false
		}
	}

	return true
}

// NormalizeScopes removes duplicate scopes and sorts them alphabetically.
//
// This is useful for consistent scope handling and storage.
// Returns nil if the input slice is nil or empty.
// Optimized for different input sizes to balance performance.
//
// Example:
//
//	normalized := scopes.NormalizeScopes([]string{"write", "read", "read", "admin.*"})
//	// Returns: []string{"admin.*", "read", "write"}
func NormalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}

	// For very small inputs, use a simpler approach to avoid map overhead
	if len(scopes) <= 3 {
		switch len(scopes) {
		case 1:
			return []string{scopes[0]}
		case 2:
			if scopes[0] == scopes[1] {
				return []string{scopes[0]}
			}
			if scopes[0] < scopes[1] {
				return []string{scopes[0], scopes[1]}
			}
			return []string{scopes[1], scopes[0]}
		case 3:
			// For 3 items, we can still optimize without using maps
			uniqueScopes := make([]string, 0, 3)

			// Simple deduplication
			for i := range scopes {
				isDuplicate := false
				for j := range uniqueScopes {
					if scopes[i] == uniqueScopes[j] {
						isDuplicate = true
						break
					}
				}
				if !isDuplicate {
					uniqueScopes = append(uniqueScopes, scopes[i])
				}
			}

			// Simple sort for up to 3 items
			if len(uniqueScopes) > 1 {
				sort.Strings(uniqueScopes)
			}

			return uniqueScopes
		}
	}

	// For larger inputs, use map-based approach
	uniqueMap := make(map[string]struct{}, len(scopes))
	for i := range scopes {
		uniqueMap[scopes[i]] = struct{}{}
	}

	// Create a new slice with unique scopes
	normalizedScopes := make([]string, 0, len(uniqueMap))
	for scope := range uniqueMap {
		normalizedScopes = append(normalizedScopes, scope)
	}

	// Sort the slice for consistent output
	sort.Strings(normalizedScopes)

	return normalizedScopes
}
