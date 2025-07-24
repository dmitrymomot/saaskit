// Package scopes provides a flexible and high-performance toolkit for working
// with OAuth-style scope strings used in authorization systems.
//
// Scopes are permissions encoded as plain strings, e.g. "read", "admin.users",
// that can be combined into white-space separated lists such as
// "read write admin.users". The package helps you parse, validate, compare
// and normalise such collections while supporting hierarchical scopes and
// wild-cards.
//
// # Overview
//
// The package treats a scope as an opaque token but understands three
// syntactic conventions:
//
//   • ScopeSeparator (" ") white-space between individual scopes inside a
//     scope list string.
//   • ScopeDelimiter (".") hierarchy delimiter that allows prefixes such as
//     "admin.*" to match all sub-scopes starting with "admin.".
//   • ScopeWildcard ("*") a wild-card that matches everything or, when used
//     as a suffix (e.g. "admin.*"), everything inside a hierarchy.
//
// Together these rules make it trivial to express rich permission models while
// keeping the implementation dependency-free and fast.
//
// # Usage
//
//	import "github.com/dmitrymomot/saaskit/pkg/scopes"
//
//	// Parse a list that came from an OAuth2 access token
//	userScopes := scopes.ParseScopes("read write admin.users")
//
//	// Convert a slice back to the canonical white-space representation
//	token := scopes.JoinScopes(userScopes) // "read write admin.users"
//
//	// Test membership (wild-cards are understood automatically)
//	if scopes.HasScope(userScopes, "admin.users.read") {
//	    // …
//	}
//
//	// Require that a user possesses all or any scopes from a set
//	if !scopes.HasAllScopes(userScopes, []string{"read", "write"}) {
//	    return errors.New("insufficient permissions")
//	}
//
// # Validation
//
// To limit the universe of permissions, validate user provided scopes against
// a predefined allow-list:
//
//	valid := scopes.ValidateScopes(
//	    userScopes,
//	    []string{"read", "write", "admin.*", "user.*"},
//	)
//	if !valid {
//	    return scopes.ErrScopeNotAllowed
//	}
//
// # Error Handling
//
// The package exposes two sentinel errors:
//
//   • ErrInvalidScope    – the supplied scope string is syntactically invalid.
//   • ErrScopeNotAllowed – the scope is not in the list of allowed scopes.
//
// These can be matched with errors.Is.
//
// # Performance
//
// All helpers are allocation-aware and choose different strategies depending
// on the slice size (see NormalizeScopes, EqualScopes, ValidateScopes).
// The public API remains allocation-free for common read-only operations.
//
// # See Also
//
// RFC 6749 §3.3 for the formal definition of the scope parameter in OAuth 2.0.
package scopes
