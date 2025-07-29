// Package limits provides resource limits and plan-based features for SaaS applications.
//
// The limits package enables enforcement of subscription-based quotas, feature flags,
// and trial periods. It provides a flexible system for defining subscription plans with
// resource limits and features, then enforcing those limits at runtime.
//
// # Architecture
//
// The package follows a clean architecture with clear separation of concerns:
//
//   - Service Layer: The LimitsService interface provides all business operations
//   - Data Sources: The Source interface abstracts plan storage (in-memory, database, etc.)
//   - Resource Counting: CounterRegistry maps resources to counting functions
//   - Plan Resolution: Flexible resolution of tenant plan IDs via context or custom resolvers
//
// The design is tenant-agnostic and can work with any ID type (string, UUID) via
// context-based resolution. It integrates seamlessly with the tenant package but
// does not require it as a dependency.
//
// # Core Concepts
//
//   - Plan: Defines subscription tiers with resource limits, features, and trial periods
//   - Resource: Countable entities like users, projects, or storage (extensible)
//   - Feature: Plan-specific capabilities like AI, SSO, or advanced analytics
//   - CounterFunc: Functions that count current resource usage for a tenant
//   - UsageInfo: Current usage and limit information for resources
//
// # Usage
//
// Basic example showing plan definition and limit checking:
//
//	import "github.com/dmitrymomot/saaskit/pkg/limits"
//
//	// Define subscription plans
//	plans := map[string]limits.Plan{
//	    "free": {
//	        ID:   "free",
//	        Name: "Free Plan",
//	        Limits: map[limits.Resource]int64{
//	            limits.ResourceUsers:    5,
//	            limits.ResourceProjects: 3,
//	        },
//	        Features:  []limits.Feature{},
//	        TrialDays: 0,
//	    },
//	    "pro": {
//	        ID:   "pro",
//	        Name: "Pro Plan",
//	        Limits: map[limits.Resource]int64{
//	            limits.ResourceUsers:    limits.Unlimited,
//	            limits.ResourceProjects: 50,
//	        },
//	        Features:  []limits.Feature{limits.FeatureAI, limits.FeatureSSO},
//	        TrialDays: 14,
//	    },
//	}
//
//	// Create plan source and resource counters
//	source := limits.NewInMemSource(plans)
//	counters := limits.NewRegistry()
//
//	// Register resource counting functions
//	counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
//	    return db.CountUsers(ctx, tenantID)
//	})
//	counters.Register(limits.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
//	    return db.CountProjects(ctx, tenantID)
//	})
//
//	// Initialize service
//	svc, err := limits.NewLimitsService(ctx, source, counters, nil)
//	if err != nil {
//	    return err
//	}
//
//	// Check if tenant can create a new resource
//	if err := svc.CanCreate(ctx, tenantID, limits.ResourceUsers); err != nil {
//	    if errors.Is(err, limits.ErrLimitExceeded) {
//	        // Handle limit exceeded - show upgrade prompt
//	    }
//	}
//
//	// Check if feature is available
//	if svc.HasFeature(ctx, tenantID, limits.FeatureAI) {
//	    // Enable AI features in the UI
//	}
//
// # Configuration
//
// The service supports multiple configuration options:
//
// Plan ID Resolution: By default, the service uses context-based plan resolution.
// You can provide a custom resolver for database-backed tenant plans:
//
//	resolver := func(ctx context.Context, tenantID uuid.UUID) (string, error) {
//	    return db.GetTenantPlanID(ctx, tenantID)
//	}
//	svc, err := limits.NewLimitsService(ctx, source, counters, resolver)
//
// Context-Based Resolution: For simpler use cases, store the plan ID in context:
//
//	ctx = limits.SetPlanIDToContext(ctx, "pro")
//	// Service will automatically use this plan ID for the request
//
// Custom Resources: Define your own resource types beyond the built-in ones:
//
//	const (
//	    ResourceStorage  limits.Resource = "storage"
//	    ResourceAPIKeys  limits.Resource = "api_keys"
//	)
//
// Custom Features: Define feature flags specific to your application:
//
//	const (
//	    FeatureWebhooks    limits.Feature = "webhooks"
//	    FeatureCustomDomain limits.Feature = "custom_domain"
//	)
//
// # Error Handling
//
// The package defines domain-specific errors with i18n-friendly keys:
//
//	// Plan-related errors
//	ErrPlanNotFound       - The requested plan doesn't exist
//	ErrPlanIDNotFound     - Unable to resolve plan ID for tenant
//	ErrPlanIDNotInContext - Context-based resolution failed
//
//	// Resource limit errors
//	ErrLimitExceeded        - Resource limit has been reached
//	ErrInvalidResource      - Unknown resource type
//	ErrNoCounterRegistered  - No counter function for resource
//	ErrDowngradeNotPossible - Current usage exceeds target plan
//
//	// Trial errors
//	ErrTrialExpired      - Trial period has ended
//	ErrTrialNotAvailable - Plan doesn't offer trials
//
// Example error handling:
//
//	err := svc.CanCreate(ctx, tenantID, limits.ResourceUsers)
//	switch {
//	case errors.Is(err, limits.ErrLimitExceeded):
//	    // Show upgrade prompt
//	case errors.Is(err, limits.ErrPlanNotFound):
//	    // Handle missing plan - maybe set default
//	case err != nil:
//	    // Handle other errors
//	}
//
// # Performance Considerations
//
//   - Counter functions are called on every limit check - implement caching if needed
//   - Plans are loaded once during service initialization and cached internally
//   - The service is thread-safe and can handle concurrent requests
//   - For high-traffic applications, consider caching usage counts in Redis/memory
//   - Benchmark results show sub-microsecond performance for limit checks
//
// # Testing Support
//
// The package includes comprehensive test coverage with integration tests demonstrating
// common workflows. Use NewInMemSource for testing without external dependencies:
//
//	testPlans := map[string]limits.Plan{...}
//	source := limits.NewInMemSource(testPlans)
//
//	// Mock counter for testing
//	counters := limits.NewRegistry()
//	counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
//	    return 3, nil // Fixed count for testing
//	})
//
// # Examples
//
// See the README.md file for additional usage examples including:
//   - Trial period management
//   - Usage percentage calculations for UI
//   - Plan comparison and downgrade validation
//   - Batch usage retrieval for dashboards
//
// For complete API documentation, use:
//
//	go doc github.com/dmitrymomot/saaskit/pkg/limits
package limits
