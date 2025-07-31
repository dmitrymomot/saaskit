package scopes_test

import (
	"fmt"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/scopes"
)

// Benchmark data sets
var (
	// Small scope sets for testing
	smallScopes      = []string{"read", "write", "delete"}
	smallValidScopes = []string{"read", "write", "delete", "admin"}

	// Medium scope sets
	mediumScopes = []string{
		"users.read", "users.write", "users.delete",
		"posts.read", "posts.write", "posts.delete",
		"comments.read", "comments.write", "comments.delete",
		"admin.read", "admin.write", "admin.delete",
		"settings.read", "settings.write",
	}
	mediumValidScopes = []string{
		"users.*", "posts.*", "comments.*",
		"admin.*", "settings.*", "analytics.*",
		"billing.*", "support.*",
	}

	// Large scope sets
	largeScopes      = generateLargeScopes(100)
	largeValidScopes = generateLargeValidScopes(50)

	// Scopes with wildcards
	wildcardScopes = []string{"admin.*", "users.*", "system.*"}
)

func generateLargeScopes(n int) []string {
	scopes := make([]string, n)
	resources := []string{"users", "posts", "comments", "settings", "admin", "analytics", "billing"}
	actions := []string{"read", "write", "delete", "update", "create"}

	for i := 0; i < n; i++ {
		resource := resources[i%len(resources)]
		action := actions[i%len(actions)]
		scopes[i] = fmt.Sprintf("%s.%s", resource, action)
	}
	return scopes
}

func generateLargeValidScopes(n int) []string {
	scopes := make([]string, n)
	resources := []string{"users", "posts", "comments", "settings", "admin", "analytics", "billing", "support", "api", "webhooks"}

	for i := 0; i < n; i++ {
		if i%3 == 0 {
			scopes[i] = resources[i%len(resources)] + ".*"
		} else {
			scopes[i] = fmt.Sprintf("%s.%s", resources[i%len(resources)], []string{"read", "write", "delete"}[i%3])
		}
	}
	return scopes
}

// BenchmarkParseScopes benchmarks the ParseScopes function
func BenchmarkParseScopes(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"Empty", ""},
		{"Single", "read"},
		{"Small", "read write delete"},
		{"Medium", "users.read users.write posts.read posts.write admin.read admin.write settings.read settings.write"},
		{"Large", generateScopeString(100)},
		{"WithSpaces", "  read   write    delete  "},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_ = scopes.ParseScopes(tc.input)
			}
		})
	}
}

// BenchmarkValidateScopes benchmarks the ValidateScopes function
func BenchmarkValidateScopes(b *testing.B) {
	testCases := []struct {
		name        string
		scopes      []string
		validScopes []string
	}{
		{"Small/Small", smallScopes, smallValidScopes},
		{"Medium/Medium", mediumScopes, mediumValidScopes},
		{"Large/Large", largeScopes, largeValidScopes},
		{"Small/Wildcards", smallScopes, wildcardScopes},
		{"Medium/Wildcards", mediumScopes, wildcardScopes},
		{"Large/Wildcards", largeScopes, wildcardScopes},
		{"Empty/Valid", []string{}, smallValidScopes},
		{"Scopes/Empty", smallScopes, []string{}},
		{"GlobalWildcard", mediumScopes, []string{"*"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_ = scopes.ValidateScopes(tc.scopes, tc.validScopes)
			}
		})
	}
}

// BenchmarkNormalizeScopes benchmarks the NormalizeScopes function
func BenchmarkNormalizeScopes(b *testing.B) {
	testCases := []struct {
		name   string
		scopes []string
	}{
		{"Empty", []string{}},
		{"Single", []string{"read"}},
		{"SmallNoDuplicates", []string{"read", "write", "delete"}},
		{"SmallWithDuplicates", []string{"read", "write", "read", "write", "delete"}},
		{"MediumNoDuplicates", mediumScopes},
		{"MediumWithDuplicates", append(mediumScopes, mediumScopes[:5]...)},
		{"LargeNoDuplicates", largeScopes},
		{"LargeWithDuplicates", append(largeScopes, largeScopes[:20]...)},
		{"AlreadySorted", []string{"admin.read", "admin.write", "users.read", "users.write"}},
		{"ReverseOrder", []string{"write", "update", "read", "delete", "create", "admin"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_ = scopes.NormalizeScopes(tc.scopes)
			}
		})
	}
}

// BenchmarkEqualScopes benchmarks the EqualScopes function
func BenchmarkEqualScopes(b *testing.B) {
	testCases := []struct {
		name    string
		scopes1 []string
		scopes2 []string
	}{
		{"Small/Equal", smallScopes, smallScopes},
		{"Small/EqualDifferentOrder", []string{"write", "read", "delete"}, []string{"read", "delete", "write"}},
		{"Small/NotEqual", smallScopes, []string{"read", "write", "admin"}},
		{"Medium/Equal", mediumScopes, mediumScopes},
		{"Medium/EqualDifferentOrder", reverseSlice(mediumScopes), mediumScopes},
		{"Medium/NotEqual", mediumScopes, append(mediumScopes[:10], "extra.scope")},
		{"Large/Equal", largeScopes, largeScopes},
		{"Large/NotEqual", largeScopes, largeScopes[:90]},
		{"DifferentLengths", smallScopes, mediumScopes},
		{"Empty/Empty", []string{}, []string{}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_ = scopes.EqualScopes(tc.scopes1, tc.scopes2)
			}
		})
	}
}

// BenchmarkScopeMatches benchmarks the ScopeMatches function
func BenchmarkScopeMatches(b *testing.B) {
	testCases := []struct {
		name    string
		scope   string
		pattern string
	}{
		{"ExactMatch", "users.read", "users.read"},
		{"NoMatch", "users.read", "posts.read"},
		{"GlobalWildcard", "users.read", "*"},
		{"NamespaceWildcard/Match", "users.read", "users.*"},
		{"NamespaceWildcard/NoMatch", "posts.read", "users.*"},
		{"DeepHierarchy/Match", "admin.users.settings.read", "admin.*"},
		{"DeepHierarchy/NoMatch", "admin.users.settings.read", "users.*"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_ = scopes.ScopeMatches(tc.scope, tc.pattern)
			}
		})
	}
}

// BenchmarkHasScope benchmarks the HasScope function
func BenchmarkHasScope(b *testing.B) {
	testCases := []struct {
		name   string
		scopes []string
		scope  string
	}{
		{"Small/Found", smallScopes, "read"},
		{"Small/NotFound", smallScopes, "admin"},
		{"Medium/Found", mediumScopes, "posts.read"},
		{"Medium/NotFound", mediumScopes, "billing.read"},
		{"Large/Found", largeScopes, largeScopes[50]},
		{"Large/NotFound", largeScopes, "nonexistent.scope"},
		{"Wildcards/Match", wildcardScopes, "admin.read"},
		{"Wildcards/NoMatch", wildcardScopes, "billing.read"},
		{"GlobalWildcard", []string{"*"}, "any.scope.here"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_ = scopes.HasScope(tc.scopes, tc.scope)
			}
		})
	}
}

// BenchmarkHasAllScopes benchmarks the HasAllScopes function
func BenchmarkHasAllScopes(b *testing.B) {
	testCases := []struct {
		name     string
		scopes   []string
		required []string
	}{
		{"Small/HasAll", []string{"read", "write", "delete", "admin"}, []string{"read", "write"}},
		{"Small/Missing", smallScopes, []string{"read", "admin"}},
		{"Medium/HasAll", mediumScopes, []string{"users.read", "posts.write"}},
		{"Medium/Missing", mediumScopes, []string{"users.read", "billing.write"}},
		{"Large/HasAll", largeScopes, largeScopes[:10]},
		{"Large/Missing", largeScopes, append(largeScopes[:10], "missing.scope")},
		{"Wildcards/HasAll", append(wildcardScopes, "specific.read"), []string{"admin.read", "users.write", "specific.read"}},
		{"GlobalWildcard", []string{"*"}, mediumScopes},
		{"EmptyRequired", mediumScopes, []string{}},
		{"EmptyScopes", []string{}, []string{"read"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_ = scopes.HasAllScopes(tc.scopes, tc.required)
			}
		})
	}
}

// BenchmarkHasAnyScopes benchmarks the HasAnyScopes function
func BenchmarkHasAnyScopes(b *testing.B) {
	testCases := []struct {
		name     string
		scopes   []string
		required []string
	}{
		{"Small/HasSome", smallScopes, []string{"read", "admin"}},
		{"Small/HasNone", smallScopes, []string{"admin", "billing"}},
		{"Medium/HasSome", mediumScopes, []string{"billing.read", "users.read"}},
		{"Medium/HasNone", mediumScopes, []string{"billing.read", "api.write"}},
		{"Large/HasSome", largeScopes, append([]string{"missing.scope"}, largeScopes[50])},
		{"Large/HasNone", largeScopes, []string{"missing1.scope", "missing2.scope"}},
		{"Wildcards/HasSome", wildcardScopes, []string{"admin.read", "billing.write"}},
		{"GlobalWildcard", []string{"*"}, []string{"any.scope"}},
		{"EmptyRequired", mediumScopes, []string{}},
		{"EmptyScopes", []string{}, []string{"read"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_ = scopes.HasAnyScopes(tc.scopes, tc.required)
			}
		})
	}
}

// BenchmarkJoinScopes benchmarks the JoinScopes function
func BenchmarkJoinScopes(b *testing.B) {
	testCases := []struct {
		name   string
		scopes []string
	}{
		{"Empty", []string{}},
		{"Single", []string{"read"}},
		{"Small", smallScopes},
		{"Medium", mediumScopes},
		{"Large", largeScopes},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_ = scopes.JoinScopes(tc.scopes)
			}
		})
	}
}

// Helper functions
func generateScopeString(n int) string {
	s := generateLargeScopes(n)
	return scopes.JoinScopes(s)
}

func reverseSlice(s []string) []string {
	result := make([]string, len(s))
	for i, v := range s {
		result[len(s)-1-i] = v
	}
	return result
}
