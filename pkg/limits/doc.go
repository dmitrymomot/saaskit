// Package limits provides resource limits and plan-based features for SaaS applications.
// It enables enforcement of subscription-based quotas, feature flags, and trial periods.
//
// The package is designed to be tenant-agnostic and can work with any ID type (string, UUID)
// via context-based resolution. It integrates seamlessly with the tenant package but does
// not require it.
//
// Key concepts:
//
//   - Plan: Defines subscription tiers with resource limits and features
//   - Resource: Countable entities like users, projects, storage
//   - Feature: Plan-specific capabilities like AI, SSO
//   - CounterFunc: Functions that count current resource usage
//
// Basic usage:
//
//	// Define plans
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
//	// Create source and counters
//	source := limits.NewInMemSource(plans)
//	counters := limits.NewRegistry()
//	counters.Register(limits.ResourceUsers, userCounter)
//
//	// Initialize service
//	svc, err := limits.NewLimitsService(ctx, source, counters, nil)
//
//	// Check limits
//	if err := svc.CanCreate(ctx, tenantID, limits.ResourceUsers); err != nil {
//	    // Handle limit exceeded
//	}
//
//	// Check features
//	if svc.HasFeature(ctx, tenantID, limits.FeatureAI) {
//	    // Enable AI features
//	}
package limits
