package scopes_test

import (
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/scopes"

	"github.com/stretchr/testify/assert"
)

func TestParseScopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single scope",
			input:    "read",
			expected: []string{"read"},
		},
		{
			name:     "multiple scopes",
			input:    "read write delete",
			expected: []string{"read", "write", "delete"},
		},
		{
			name:     "extra spaces",
			input:    "  read   write  ",
			expected: []string{"read", "write"},
		},
		{
			name:     "hierarchical scopes",
			input:    "admin.read user.write system.*",
			expected: []string{"admin.read", "user.write", "system.*"},
		},
		{
			name:     "mixed scopes with wildcards",
			input:    "* admin.read user.*",
			expected: []string{"*", "admin.read", "user.*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := scopes.ParseScopes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJoinScopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		scopes   []string
		expected string
	}{
		{
			name:     "empty scopes",
			scopes:   []string{},
			expected: "",
		},
		{
			name:     "nil scopes",
			scopes:   nil,
			expected: "",
		},
		{
			name:     "single scope",
			scopes:   []string{"read"},
			expected: "read",
		},
		{
			name:     "multiple scopes",
			scopes:   []string{"read", "write"},
			expected: "read write",
		},
		{
			name:     "hierarchical scopes",
			scopes:   []string{"admin.read", "user.write", "system.*"},
			expected: "admin.read user.write system.*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := scopes.JoinScopes(tt.scopes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		scopes   []string
		scope    string
		expected bool
	}{
		{
			name:     "empty scopes",
			scopes:   []string{},
			scope:    "read",
			expected: false,
		},
		{
			name:     "exact match",
			scopes:   []string{"read", "write"},
			scope:    "read",
			expected: true,
		},
		{
			name:     "no match",
			scopes:   []string{"read", "write"},
			scope:    "delete",
			expected: false,
		},
		{
			name:     "global wildcard",
			scopes:   []string{"*"},
			scope:    "anything",
			expected: true,
		},
		{
			name:     "namespace wildcard match",
			scopes:   []string{"admin.*"},
			scope:    "admin.read",
			expected: true,
		},
		{
			name:     "namespace wildcard no match",
			scopes:   []string{"admin.*"},
			scope:    "user.read",
			expected: false,
		},
		{
			name:     "deep scope match",
			scopes:   []string{"admin.users.*"},
			scope:    "admin.users.read",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := scopes.HasScope(tt.scopes, tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasAllScopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		scopes   []string
		required []string
		expected bool
	}{
		{
			name:     "empty required",
			scopes:   []string{"read"},
			required: []string{},
			expected: true,
		},
		{
			name:     "empty scopes",
			scopes:   []string{},
			required: []string{"read"},
			expected: false,
		},
		{
			name:     "has all required",
			scopes:   []string{"read", "write"},
			required: []string{"read"},
			expected: true,
		},
		{
			name:     "missing required",
			scopes:   []string{"read"},
			required: []string{"write"},
			expected: false,
		},
		{
			name:     "global wildcard",
			scopes:   []string{"*"},
			required: []string{"read", "write", "admin.users"},
			expected: true,
		},
		{
			name:     "namespace wildcard",
			scopes:   []string{"admin.*"},
			required: []string{"admin.read", "admin.write"},
			expected: true,
		},
		{
			name:     "mixed wildcards",
			scopes:   []string{"admin.*", "user.read"},
			required: []string{"admin.write", "user.read"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := scopes.HasAllScopes(tt.scopes, tt.required)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasAnyScopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		scopes   []string
		required []string
		expected bool
	}{
		{
			name:     "empty required",
			scopes:   []string{"read"},
			required: []string{},
			expected: true,
		},
		{
			name:     "empty scopes",
			scopes:   []string{},
			required: []string{"read"},
			expected: false,
		},
		{
			name:     "has one required",
			scopes:   []string{"read", "write"},
			required: []string{"write", "delete"},
			expected: true,
		},
		{
			name:     "has none required",
			scopes:   []string{"read"},
			required: []string{"write", "delete"},
			expected: false,
		},
		{
			name:     "global wildcard",
			scopes:   []string{"*"},
			required: []string{"anything", "whatever"},
			expected: true,
		},
		{
			name:     "namespace wildcard match",
			scopes:   []string{"admin.*"},
			required: []string{"admin.read", "user.write"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := scopes.HasAnyScopes(tt.scopes, tt.required)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEqualScopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		scopes1  []string
		scopes2  []string
		expected bool
	}{
		{
			name:     "empty scopes",
			scopes1:  []string{},
			scopes2:  []string{},
			expected: true,
		},
		{
			name:     "nil scopes",
			scopes1:  nil,
			scopes2:  nil,
			expected: true,
		},
		{
			name:     "same scopes different order",
			scopes1:  []string{"read", "write"},
			scopes2:  []string{"write", "read"},
			expected: true,
		},
		{
			name:     "different scopes",
			scopes1:  []string{"read"},
			scopes2:  []string{"write"},
			expected: false,
		},
		{
			name:     "different lengths",
			scopes1:  []string{"read", "write"},
			scopes2:  []string{"read"},
			expected: false,
		},
		{
			name:     "with wildcards",
			scopes1:  []string{"admin.*", "user.read"},
			scopes2:  []string{"user.read", "admin.*"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := scopes.EqualScopes(tt.scopes1, tt.scopes2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateScopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		scopes      []string
		validScopes []string
		expected    bool
	}{
		{
			name:        "empty scopes",
			scopes:      []string{},
			validScopes: []string{"read", "write"},
			expected:    true,
		},
		{
			name:        "empty valid scopes",
			scopes:      []string{"read"},
			validScopes: []string{},
			expected:    false,
		},
		{
			name:        "all valid",
			scopes:      []string{"read", "write"},
			validScopes: []string{"read", "write", "delete"},
			expected:    true,
		},
		{
			name:        "invalid scope",
			scopes:      []string{"read", "invalid"},
			validScopes: []string{"read", "write"},
			expected:    false,
		},
		{
			name:        "wildcard in valid scopes",
			scopes:      []string{"custom.scope", "another.scope"},
			validScopes: []string{"*"},
			expected:    true,
		},
		{
			name:        "namespace wildcard in valid scopes",
			scopes:      []string{"admin.read", "admin.write"},
			validScopes: []string{"admin.*"},
			expected:    true,
		},
		{
			name:        "mixed wildcards and explicit scopes",
			scopes:      []string{"admin.read", "user.write"},
			validScopes: []string{"admin.*", "user.write"},
			expected:    true,
		},
		{
			name:        "invalid with wildcards",
			scopes:      []string{"admin.read", "user.write"},
			validScopes: []string{"admin.*", "system.*"},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := scopes.ValidateScopes(tt.scopes, tt.validScopes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		original string
	}{
		{
			name:     "simple scopes",
			original: "read write delete",
		},
		{
			name:     "hierarchical scopes",
			original: "admin.read user.write system.*",
		},
		{
			name:     "mixed wildcards",
			original: "* admin.* user.read.write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			scopeSlice := scopes.ParseScopes(tt.original)
			result := scopes.JoinScopes(scopeSlice)
			assert.Equal(t, tt.original, result)
		})
	}
}

func TestNormalizeScopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		scopes   []string
		expected []string
	}{
		{
			name:     "empty scopes",
			scopes:   []string{},
			expected: nil,
		},
		{
			name:     "nil scopes",
			scopes:   nil,
			expected: nil,
		},
		{
			name:     "no duplicates",
			scopes:   []string{"read", "write", "delete"},
			expected: []string{"delete", "read", "write"},
		},
		{
			name:     "with duplicates",
			scopes:   []string{"read", "write", "read", "delete", "write"},
			expected: []string{"delete", "read", "write"},
		},
		{
			name:     "already sorted",
			scopes:   []string{"admin", "delete", "read", "write"},
			expected: []string{"admin", "delete", "read", "write"},
		},
		{
			name:     "with wildcards",
			scopes:   []string{"user.*", "admin.*", "*", "admin.*"},
			expected: []string{"*", "admin.*", "user.*"},
		},
		{
			name:     "with hierarchical scopes",
			scopes:   []string{"admin.write", "user.read", "admin.read", "user.read"},
			expected: []string{"admin.read", "admin.write", "user.read"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := scopes.NormalizeScopes(tt.scopes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		scope    string
		pattern  string
		expected bool
	}{
		{
			name:     "exact match",
			scope:    "read",
			pattern:  "read",
			expected: true,
		},
		{
			name:     "global wildcard",
			scope:    "anything",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "namespace wildcard match",
			scope:    "admin.read",
			pattern:  "admin.*",
			expected: true,
		},
		{
			name:     "deep namespace wildcard match",
			scope:    "admin.users.read",
			pattern:  "admin.users.*",
			expected: true,
		},
		{
			name:     "namespace wildcard no match",
			scope:    "user.read",
			pattern:  "admin.*",
			expected: false,
		},
		{
			name:     "no match different strings",
			scope:    "write",
			pattern:  "read",
			expected: false,
		},
		{
			name:     "partial string no match",
			scope:    "reading",
			pattern:  "read",
			expected: false,
		},
		{
			name:     "wildcard at wrong position",
			scope:    "read.admin",
			pattern:  "read.*admin",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := scopes.ScopeMatches(tt.scope, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateScopesPerformance(t *testing.T) {
	t.Parallel()
	// Generate large collections for testing optimization paths
	const smallSize = 5
	const largeSize = 15

	// Generate small and large scope collections
	smallScopes := make([]string, smallSize)
	largeScopes := make([]string, largeSize)
	smallValidScopes := make([]string, smallSize)
	largeValidScopes := make([]string, largeSize)

	// Fill with unique values
	for i := 0; i < largeSize; i++ {
		if i < smallSize {
			smallScopes[i] = "scope" + string(rune('a'+i))
			smallValidScopes[i] = "scope" + string(rune('a'+i))
		}
		largeScopes[i] = "scope" + string(rune('a'+i))
		largeValidScopes[i] = "scope" + string(rune('a'+i))
	}

	// Add some wildcard patterns to the valid scopes
	largeValidScopes[0] = "other.*"
	smallValidScopes[0] = "other.*"

	// Test cases for different collection sizes
	tests := []struct {
		name        string
		scopes      []string
		validScopes []string
		expected    bool
	}{
		{
			name:        "small collections all valid",
			scopes:      []string{"scopea", "scopeb", "scopec"},
			validScopes: []string{"scopea", "scopeb", "scopec", "*"},
			expected:    true,
		},
		{
			name:        "large collections all valid",
			scopes:      largeScopes,
			validScopes: append([]string{"*"}, largeValidScopes...),
			expected:    true,
		},
		{
			name:        "small collections with invalid scope",
			scopes:      []string{"scopea", "scopeb", "invalid"},
			validScopes: []string{"scopea", "scopeb", "scopec"},
			expected:    false,
		},
		{
			name:        "large collections with invalid scope",
			scopes:      append(append([]string{}, largeScopes...), "invalid"),
			validScopes: largeValidScopes,
			expected:    false,
		},
		{
			name:        "mixed sizes: small scopes, large valid",
			scopes:      smallScopes,
			validScopes: append(largeValidScopes, "*"),
			expected:    true,
		},
		{
			name:        "mixed sizes: large scopes, small valid with wildcard",
			scopes:      largeScopes,
			validScopes: append([]string{"*"}, smallValidScopes...),
			expected:    true,
		},
		{
			name:        "wildcard match in large collections",
			scopes:      []string{"other.thing", "other.value"},
			validScopes: []string{"other.*", "specific"},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := scopes.ValidateScopes(tt.scopes, tt.validScopes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEqualScopesOptimized(t *testing.T) {
	t.Parallel()
	// Test different collection sizes
	tests := []struct {
		name     string
		scopes1  []string
		scopes2  []string
		expected bool
	}{
		{
			name:     "single item equal",
			scopes1:  []string{"read"},
			scopes2:  []string{"read"},
			expected: true,
		},
		{
			name:     "single item not equal",
			scopes1:  []string{"read"},
			scopes2:  []string{"write"},
			expected: false,
		},
		{
			name:     "two items equal same order",
			scopes1:  []string{"read", "write"},
			scopes2:  []string{"read", "write"},
			expected: true,
		},
		{
			name:     "two items equal different order",
			scopes1:  []string{"read", "write"},
			scopes2:  []string{"write", "read"},
			expected: true,
		},
		{
			name:     "three items equal different order",
			scopes1:  []string{"read", "write", "delete"},
			scopes2:  []string{"delete", "read", "write"},
			expected: true,
		},
		{
			name:     "three items not equal",
			scopes1:  []string{"read", "write", "delete"},
			scopes2:  []string{"read", "write", "admin"},
			expected: false,
		},
		{
			name:     "four items equal different order",
			scopes1:  []string{"read", "write", "delete", "admin"},
			scopes2:  []string{"admin", "delete", "read", "write"},
			expected: true,
		},
		{
			name:     "five items equal different order",
			scopes1:  []string{"read", "write", "delete", "admin", "user.*"},
			scopes2:  []string{"user.*", "admin", "delete", "read", "write"},
			expected: true,
		},
		{
			name:     "different counts",
			scopes1:  []string{"read", "write", "delete", "admin", "user.*"},
			scopes2:  []string{"user.*", "admin", "delete", "read"},
			expected: false,
		},
		{
			name:     "duplicates in one but not the other",
			scopes1:  []string{"read", "write", "read"},
			scopes2:  []string{"read", "write"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := scopes.EqualScopes(tt.scopes1, tt.scopes2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeScopesOptimized(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		scopes   []string
		expected []string
	}{
		{
			name:     "single item",
			scopes:   []string{"read"},
			expected: []string{"read"},
		},
		{
			name:     "two items already sorted",
			scopes:   []string{"read", "write"},
			expected: []string{"read", "write"},
		},
		{
			name:     "two items need sorting",
			scopes:   []string{"write", "read"},
			expected: []string{"read", "write"},
		},
		{
			name:     "two items with duplicate",
			scopes:   []string{"read", "read"},
			expected: []string{"read"},
		},
		{
			name:     "three items need sorting",
			scopes:   []string{"write", "read", "admin"},
			expected: []string{"admin", "read", "write"},
		},
		{
			name:     "three items with duplicates",
			scopes:   []string{"read", "write", "read"},
			expected: []string{"read", "write"},
		},
		{
			name:     "three items all same",
			scopes:   []string{"read", "read", "read"},
			expected: []string{"read"},
		},
		{
			name:     "four items with duplicates",
			scopes:   []string{"read", "write", "admin", "read"},
			expected: []string{"admin", "read", "write"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := scopes.NormalizeScopes(tt.scopes)
			assert.Equal(t, tt.expected, result)
		})
	}
}
