package limits

// Resource represents a countable tenant resource type.
type Resource string

// Predefined resource types.
const (
	ResourceUsers    Resource = "users"
	ResourceProjects Resource = "projects"
	// extend as needed
)

// Limit constants
const (
	// Unlimited represents a resource with no limit (-1)
	Unlimited int64 = -1
)

// Feature is a string type representing a plan-specific feature flag.
type Feature string

// Predefined feature flags for plans.
const (
	FeatureAI  Feature = "ai"  // Enables AI-powered features
	FeatureSSO Feature = "sso" // Enables Single-Sign-On support
	// Add more as needed
)

// UsageInfo contains the current usage and limit for a resource.
type UsageInfo struct {
	Current int64 `json:"current"`
	Limit   int64 `json:"limit"`
}
