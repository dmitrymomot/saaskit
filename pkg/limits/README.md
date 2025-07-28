# limits

Resource limits and plan-based features for SaaS applications.

## Features

- Plan-based resource limits (users, projects, storage)
- Feature flags for subscription tiers (AI, SSO, etc.)
- Trial period management with automatic expiration
- Plan comparison and downgrade validation

## Installation

```go
import "github.com/dmitrymomot/saaskit/pkg/limits"
```

## Usage

```go
// Define plans
plans := map[string]limits.Plan{
    "free": {
        ID:   "free",
        Name: "Free Plan",
        Limits: map[limits.Resource]int64{
            limits.ResourceUsers:    5,
            limits.ResourceProjects: 3,
        },
        Features:  []limits.Feature{},
        TrialDays: 0,
    },
    "pro": {
        ID:   "pro",
        Name: "Pro Plan",
        Limits: map[limits.Resource]int64{
            limits.ResourceUsers:    limits.Unlimited,
            limits.ResourceProjects: 50,
        },
        Features:  []limits.Feature{limits.FeatureAI, limits.FeatureSSO},
        TrialDays: 14,
    },
}

// Create source and counters
source := limits.NewInMemSource(plans)
counters := limits.NewRegistry()
counters.Register(limits.ResourceUsers, userCounter)

// Initialize service
svc, err := limits.NewLimitsService(ctx, source, counters, nil)
if err != nil {
    return err
}

// Check limits
if err := svc.CanCreate(ctx, tenantID, limits.ResourceUsers); err != nil {
    // Handle limit exceeded
}

// Check features
if svc.HasFeature(ctx, tenantID, limits.FeatureAI) {
    // Enable AI features
}
```

## Common Operations

### Register Resource Counters

```go
counters := limits.NewRegistry()
counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
    return db.CountUsers(ctx, tenantID)
})
counters.Register(limits.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
    return db.CountProjects(ctx, tenantID)
})
```

### Check Trial Status

```go
// Check if trial is active
err := svc.CheckTrial(ctx, tenantID, tenant.CreatedAt)
if errors.Is(err, limits.ErrTrialExpired) {
    // Handle expired trial
}
```

### Get Usage Information

```go
// Get specific resource usage
used, limit, err := svc.GetUsage(ctx, tenantID, limits.ResourceUsers)

// Get all resource usage
usage, err := svc.GetAllUsage(ctx, tenantID)
for resource, info := range usage {
    fmt.Printf("%s: %d/%d\n", resource, info.Current, info.Limit)
}

// Get usage percentage for UI
percentage := svc.GetUsagePercentage(ctx, tenantID, limits.ResourceUsers)
```

### Plan Validation and Comparison

```go
// Verify plan exists
err := svc.VerifyPlan(ctx, "pro")

// Check if downgrade is possible
err := svc.CanDowngrade(ctx, tenantID, "free")
if errors.Is(err, limits.ErrDowngradeNotPossible) {
    // Current usage exceeds target plan limits
}

// Compare plans
comparison := limits.ComparePlans(&currentPlan, &targetPlan)
if comparison.HasResourceDecreases() {
    // Handle resources with decreased limits
}
```

## Error Handling

```go
// Package errors:
var (
    // Plan errors
    ErrPlanNotFound       = errors.New("limits.errors.plan_not_found")
    ErrPlanIDNotFound     = errors.New("limits.errors.plan_id_not_found")

    // Resource limit errors
    ErrLimitExceeded        = errors.New("limits.errors.limit_exceeded")
    ErrInvalidResource      = errors.New("limits.errors.invalid_resource")
    ErrNoCounterRegistered  = errors.New("limits.errors.no_counter_registered")
    ErrDowngradeNotPossible = errors.New("limits.errors.downgrade_not_possible")

    // Trial errors
    ErrTrialExpired      = errors.New("limits.errors.trial_expired")
    ErrTrialNotAvailable = errors.New("limits.errors.trial_not_available")
)

// Usage:
if errors.Is(err, limits.ErrLimitExceeded) {
    // Handle resource limit exceeded
}
```

## Configuration

```go
// Custom plan ID resolver (default uses context)
resolver := func(ctx context.Context, tenantID uuid.UUID) (string, error) {
    return db.GetTenantPlanID(ctx, tenantID)
}

svc, err := limits.NewLimitsService(ctx, source, counters, resolver)

// Or use context-based resolution
ctx = limits.WithPlanID(ctx, "pro")
```

## API Documentation

```bash
# Full API documentation
go doc github.com/dmitrymomot/saaskit/pkg/limits

# Specific function or type
go doc github.com/dmitrymomot/saaskit/pkg/limits.LimitsService
```

## Notes

- Counter functions should be fast - use caching or pre-aggregated values
- Plans are immutable after service initialization for thread safety
- Default plan ID resolver uses context; provide custom resolver for database-backed plans
