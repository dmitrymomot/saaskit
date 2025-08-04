package audit_test

import (
	"fmt"

	"github.com/dmitrymomot/saaskit/pkg/audit"
)

func ExampleMetadataFilter() {
	// Create a filter with custom rules
	filter := audit.NewMetadataFilter(
		// Remove internal endpoints from logs
		audit.WithCustomField("endpoint", audit.FilterActionRemove),
		// Hash user IDs for privacy
		audit.WithCustomField("user_id", audit.FilterActionHash),
		// Allow certain fields to pass through
		audit.WithAllowedField("request_id"),
		// Remove all fields matching wildcard patterns
		audit.WithCustomField("*.secret", audit.FilterActionRemove),
		audit.WithCustomField("internal.*", audit.FilterActionRemove),
	)

	// Example metadata that might contain sensitive information
	metadata := map[string]any{
		"user_id":         "user-12345",
		"email":           "user@example.com",
		"password":        "supersecret",
		"endpoint":        "/api/v1/internal/admin",
		"request_id":      "req-67890",
		"api.secret":      "api-key-123",
		"internal.config": "sensitive-config",
		"action":          "user.login",
	}

	// Apply the filter
	filtered := filter.Filter(metadata)

	// Password is removed by default PII rules
	fmt.Printf("password: %v\n", filtered["password"])
	// Email is hashed by default PII rules
	fmt.Printf("email is hashed: %v\n", filtered["email"] != "user@example.com")
	// user_id is hashed by custom rule
	fmt.Printf("user_id is hashed: %v\n", filtered["user_id"] != "user-12345")
	// endpoint is removed by custom rule
	fmt.Printf("endpoint: %v\n", filtered["endpoint"])
	// request_id is allowed through
	fmt.Printf("request_id: %v\n", filtered["request_id"])
	// Wildcard patterns removed these
	fmt.Printf("api.secret: %v\n", filtered["api.secret"])
	fmt.Printf("internal.config: %v\n", filtered["internal.config"])
	// Regular fields pass through
	fmt.Printf("action: %v\n", filtered["action"])

	// Output:
	// password: <nil>
	// email is hashed: true
	// user_id is hashed: true
	// endpoint: <nil>
	// request_id: req-67890
	// api.secret: <nil>
	// internal.config: <nil>
	// action: user.login
}

func ExampleMetadataFilter_withLogger() {
	// Create a metadata filter
	filter := audit.NewMetadataFilter(
		audit.WithCustomField("internal.*", audit.FilterActionRemove),
		audit.WithAllowedField("user_email"),
	)

	// Example event metadata that needs filtering
	metadata := map[string]any{
		"user_email":      "allowed@example.com",
		"password":        "newpassword123",
		"internal.debug":  "sensitive debug info",
		"profile_updated": true,
	}

	// Apply filter before storing or logging
	filtered := filter.Filter(metadata)

	fmt.Printf("Filtered metadata contains password: %v\n", filtered["password"] != nil)
	fmt.Printf("User email preserved: %v\n", filtered["user_email"] == "allowed@example.com")
	fmt.Printf("Internal debug removed: %v\n", filtered["internal.debug"] == nil)
	fmt.Printf("Profile updated preserved: %v\n", filtered["profile_updated"] == true)

	// Output:
	// Filtered metadata contains password: false
	// User email preserved: true
	// Internal debug removed: true
	// Profile updated preserved: true
}
