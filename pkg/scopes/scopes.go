package scopes

import (
	"slices"
	"sort"
	"strings"
)

const (
	// ScopeSeparator is used to separate multiple scopes in a string
	ScopeSeparator = " "

	// ScopeWildcard represents a wildcard scope that matches everything
	ScopeWildcard = "*"

	// ScopeDelimiter is used to separate scope parts (e.g., "admin.read")
	ScopeDelimiter = "."
)

// ParseScopes converts a space-separated string of scopes into a string slice.
//
// Trims spaces, removes empty entries, and minimizes allocations. Returns nil for empty input.
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
// Uses optimized comparison strategies based on slice size to minimize allocations.
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

	// For small slices, sorting is more efficient than maps
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
		return slices.Equal(s1, s2)
	}

	// For larger slices, use maps for O(n) performance
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
// Supports wildcards and uses optimized strategies based on collection size.
// Empty scopes are valid, but empty validScopes causes validation to fail.
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

	// Use map-based validation for large collections (better performance)
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

// validateScopesWithMap optimizes validation for large collections by preprocessing valid scopes.
func validateScopesWithMap(scopes, validScopes []string) bool {
	// Separate exact matches from wildcard patterns for O(1) lookup
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
		// Check exact match first (O(1))
		if _, ok := exactMatches[scope]; ok {
			continue
		}

		// Then check wildcard patterns
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
// Uses size-optimized strategies for consistent performance.
// Returns nil for empty input.
//
// Example:
//
//	normalized := scopes.NormalizeScopes([]string{"write", "read", "read", "admin.*"})
//	// Returns: []string{"admin.*", "read", "write"}
func NormalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}

	// Use simpler approach for small inputs to avoid map overhead
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
			// Optimize 3-item case without maps
			uniqueScopes := make([]string, 0, 3)

			// Deduplicate manually
			for i := range scopes {
				if !slices.Contains(uniqueScopes, scopes[i]) {
					uniqueScopes = append(uniqueScopes, scopes[i])
				}
			}

			// Sort manually for small collections
			if len(uniqueScopes) > 1 {
				sort.Strings(uniqueScopes)
			}

			return uniqueScopes
		}
	}

	// Use map-based deduplication for larger inputs
	uniqueMap := make(map[string]struct{}, len(scopes))
	for i := range scopes {
		uniqueMap[scopes[i]] = struct{}{}
	}

	// Extract unique scopes into slice
	normalizedScopes := make([]string, 0, len(uniqueMap))
	for scope := range uniqueMap {
		normalizedScopes = append(normalizedScopes, scope)
	}

	// Sort for consistent output
	sort.Strings(normalizedScopes)

	return normalizedScopes
}
