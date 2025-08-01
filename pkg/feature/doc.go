// Package feature provides a comprehensive feature flag management system for Go applications.
//
// The feature package enables controlled feature rollouts through flexible strategies including
// user targeting, percentage-based rollouts, environment-based activation, and composite rules.
// It follows a provider-based architecture allowing for different backend implementations while
// maintaining a consistent API.
//
// # Architecture
//
// The package is built around three core concepts:
//
// 1. Flags - Configuration units that define features and their rollout rules
// 2. Strategies - Evaluation logic that determines feature availability
// 3. Providers - Backend storage and retrieval implementations
//
// Feature evaluation happens in two stages: first checking if a flag is globally enabled,
// then evaluating its strategy against the provided context. This allows for both simple
// on/off toggles and sophisticated rollout rules.
//
// The Provider interface is organized into three logical method groups:
//   - Evaluation methods: IsEnabled, GetFlag
//   - Management methods: ListFlags, CreateFlag, UpdateFlag, DeleteFlag
//   - Lifecycle methods: Close
//
// # Usage
//
// Basic feature flag setup:
//
//	import "github.com/dmitrymomot/saaskit/pkg/feature"
//
//	// Create a provider with initial flags
//	provider, err := feature.NewMemoryProvider(
//		&feature.Flag{
//			Name:        "new-ui",
//			Description: "Enable redesigned user interface",
//			Enabled:     true,
//			Strategy:    feature.NewAlwaysOnStrategy(),
//		},
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer provider.Close()
//
//	// Check if feature is enabled
//	enabled, err := provider.IsEnabled(ctx, "new-ui")
//	if err != nil {
//		// Handle error
//	}
//	if enabled {
//		// Show new UI
//	}
//
// # Strategies
//
// The package provides several built-in strategies:
//
// AlwaysStrategy - Returns a constant value (on/off)
// TargetedStrategy - Enables features for specific users, groups, or percentages
// EnvironmentStrategy - Activates features in specific environments
// CompositeStrategy - Combines multiple strategies with AND/OR logic
//
// Example of percentage-based rollout:
//
//	percentage := 25
//	flag := &feature.Flag{
//		Name:    "experimental-feature",
//		Enabled: true,
//		Strategy: feature.NewTargetedStrategy(
//			feature.TargetCriteria{
//				Percentage: &percentage,
//			},
//			feature.WithUserIDExtractor(getUserID),
//		),
//	}
//
// # Context Extractors
//
// The package uses extractor functions to retrieve evaluation data from context,
// maintaining decoupling between the feature system and application logic:
//
//	func getUserID(ctx context.Context) string {
//		if user, ok := ctx.Value("user").(*User); ok {
//			return user.ID
//		}
//		return ""
//	}
//
//	strategy := feature.NewTargetedStrategy(
//		criteria,
//		feature.WithUserIDExtractor(getUserID),
//	)
//
// # Error Handling
//
// The package defines specific errors for different failure scenarios:
//
//	flag, err := provider.GetFlag(ctx, "unknown")
//	if errors.Is(err, feature.ErrFlagNotFound) {
//		// Flag doesn't exist
//	}
//
// All errors follow consistent naming patterns and can be checked using errors.Is.
//
// # Performance Considerations
//
// The MemoryProvider uses read-write locks for thread-safe concurrent access.
// For high-throughput applications, consider caching IsEnabled results at the
// application level to reduce lock contention.
//
// Percentage-based rollouts use consistent hashing (FNV-1a) to ensure users
// always receive the same feature state across evaluations.
//
// The package includes comprehensive benchmarks for performance monitoring:
//   - Provider operations (IsEnabled, ListFlags) under various scenarios
//   - Strategy evaluation performance for all strategy types
//   - Concurrent access patterns
//
// Memory optimizations include pre-allocated slice capacity in ListFlags when
// filtering by tags, reducing allocations during filtering operations.
//
// Run benchmarks with: go test -bench=. ./pkg/feature/...
//
// # Examples
//
// See the package examples and README.md for detailed usage patterns including
// composite strategies, environment-based features, and user targeting.
package feature
