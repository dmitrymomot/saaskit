// Package subscription provides SaaS subscription management with resource limits,
// feature flags, trial periods, and billing provider integration.
//
// The package implements a flexible subscription system that can enforce usage limits,
// control feature access, and integrate with various payment providers (Paddle, Stripe,
// Lemonsqueezy) through a minimal interface. It's designed for solo developers building
// SaaS applications who need a pragmatic approach to subscription management.
//
// # Architecture
//
// The package follows a service-oriented architecture with clear separation of concerns:
//
//   - Service: Main interface providing all subscription operations
//   - Plan: Defines subscription tiers with limits and features
//   - Provider: Abstracts payment provider interactions
//   - Store: Persists subscription data
//   - Counter: Tracks resource usage
//
// Resource counting is delegated to the application layer through ResourceCounterFunc
// callbacks, allowing flexible implementation strategies (database aggregates, caching,
// external services).
//
// # Usage
//
// Basic usage involves creating a service with plans, provider, and store:
//
//	import "github.com/dmitrymomot/saaskit/pkg/subscription"
//
//	// Define subscription plans
//	plans := []subscription.Plan{
//		{
//			ID:       "free",
//			Name:     "Free Tier",
//			Interval: subscription.BillingIntervalNone,
//			Limits: map[subscription.Resource]int64{
//				subscription.ResourceUsers:    1,
//				subscription.ResourceProjects: 3,
//			},
//		},
//		{
//			ID:       "price_pro_monthly", // Provider's price ID
//			Name:     "Professional",
//			Interval: subscription.BillingIntervalMonthly,
//			Price:    subscription.Money{Amount: 9900, Currency: "USD"},
//			Limits: map[subscription.Resource]int64{
//				subscription.ResourceUsers:    subscription.Unlimited,
//				subscription.ResourceProjects: subscription.Unlimited,
//			},
//			Features: []subscription.Feature{
//				subscription.FeatureAI,
//				subscription.FeatureSSO,
//			},
//			TrialDays: 14,
//		},
//	}
//
//	// Create service
//	svc, err := subscription.NewService(
//		ctx,
//		subscription.NewInMemSource(plans...),
//		provider, // Your BillingProvider implementation
//		store,    // Your SubscriptionStore implementation
//		subscription.WithCounter(subscription.ResourceUsers, userCounter),
//		subscription.WithCounter(subscription.ResourceProjects, projectCounter),
//	)
//
// # Resource Limits
//
// The package enforces resource limits by consulting registered counter functions:
//
//	// Check if user can create a resource
//	err := svc.CanCreate(ctx, tenantID, subscription.ResourceUsers)
//	if errors.Is(err, subscription.ErrLimitExceeded) {
//		// Show upgrade prompt
//	}
//
//	// Get current usage
//	used, limit, err := svc.GetUsage(ctx, tenantID, subscription.ResourceProjects)
//
// Counter functions must be fast as they're called on every resource creation attempt.
// Consider using database aggregates or cached values.
//
// # Feature Flags
//
// Features are boolean flags that enable/disable functionality based on plan:
//
//	if svc.HasFeature(ctx, tenantID, subscription.FeatureAI) {
//		// Enable AI features
//	}
//
// # Billing Integration
//
// The package uses a minimal BillingProvider interface that leverages hosted checkouts
// and customer portals, eliminating PCI compliance concerns:
//
//	// Create checkout session
//	link, err := svc.CreateCheckoutLink(ctx, tenantID, "price_pro_monthly",
//		subscription.CheckoutOptions{
//			SuccessURL: "https://app.com/success",
//			CancelURL:  "https://app.com/cancel",
//		},
//	)
//	// Redirect user to link.URL
//
//	// Handle provider webhooks
//	err = svc.HandleWebhook(ctx, payload, signature)
//
// # Error Handling
//
// The package defines specific errors for different failure scenarios:
//
//	if errors.Is(err, subscription.ErrLimitExceeded) {
//		// Handle limit exceeded
//	}
//	if errors.Is(err, subscription.ErrNoCounterRegistered) {
//		// Counter not registered for resource
//	}
//	if errors.Is(err, subscription.ErrTrialExpired) {
//		// Trial period ended
//	}
//
// # Performance Considerations
//
// Resource counters are called frequently and must be optimized:
//   - Use database indexes for count queries
//   - Implement caching where appropriate
//   - Consider eventual consistency for non-critical resources
//   - Batch count operations when checking multiple resources
//
// The service holds plans in memory after loading, so plan changes require service restart.
//
// # Examples
//
// See the README.md file for complete usage examples including webhook handling,
// customer portal integration, and trial management.
package subscription
