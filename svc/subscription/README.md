# Subscription

SaaS subscription management with resource limits, feature flags, and billing provider integration.

## Features

- **Resource Limits** - Enforce usage limits for countable resources (users, projects, API calls, etc.)
- **Feature Flags** - Control access to features based on subscription plan
- **Billing Integration** - Provider-agnostic integration with Paddle, Stripe, or Lemonsqueezy
- **Trial Management** - Built-in trial period handling with automatic expiration

## Installation

```go
import "github.com/dmitrymomot/saaskit/svc/subscription"
```

## Usage

```go
// Define plans
plans := []subscription.Plan{
    {
        ID:       "free",
        Name:     "Free",
        Interval: subscription.BillingIntervalNone,
        Limits: map[subscription.Resource]int64{
            subscription.ResourceUsers:    1,
            subscription.ResourceProjects: 3,
        },
    },
    {
        ID:       "price_pro_monthly", // Provider's price ID
        Name:     "Professional",
        Interval: subscription.BillingIntervalMonthly,
        Price:    subscription.Money{Amount: 9900, Currency: "USD"},
        Limits: map[subscription.Resource]int64{
            subscription.ResourceUsers:    subscription.Unlimited,
            subscription.ResourceProjects: subscription.Unlimited,
        },
        Features: []subscription.Feature{
            subscription.FeatureAI,
            subscription.FeatureSSO,
        },
        TrialDays: 14,
    },
}

// Create service
svc, err := subscription.NewService(
    ctx,
    subscription.NewInMemSource(plans...),
    paddleProvider, // Your BillingProvider implementation
    store,          // Your SubscriptionStore implementation
    // Register resource counters (panic if duplicate)
    subscription.WithCounter(subscription.ResourceUsers, userCounter),
    subscription.WithCounter(subscription.ResourceProjects, projectCounter),
)
```

## Common Operations

### Check Resource Limits

```go
// Check if user can create a resource
err := svc.CanCreate(ctx, tenantID, subscription.ResourceUsers)
if errors.Is(err, subscription.ErrLimitExceeded) {
    // Show upgrade prompt
}

// Get current usage
used, limit, err := svc.GetUsage(ctx, tenantID, subscription.ResourceProjects)
```

### Check Feature Access

```go
// Check if feature is available
if svc.HasFeature(ctx, tenantID, subscription.FeatureAI) {
    // Enable AI features
}
```

### Handle Subscriptions

```go
// Create checkout session for paid plan
link, err := svc.CreateCheckoutLink(ctx, tenantID, "price_pro_monthly",
    subscription.CheckoutOptions{
        Email:      "user@example.com",
        SuccessURL: "https://app.com/success",
        CancelURL:  "https://app.com/cancel",
    },
)
// Redirect to link.URL

// Handle webhook from provider (in HTTP handler)
err = svc.HandleWebhook(r) // r is *http.Request

// Get customer portal link
portal, err := svc.GetCustomerPortalLink(ctx, tenantID)
// Redirect to portal.URL
```

## Error Handling

```go
// Package errors:
var (
    ErrPlanNotFound        = errors.New("subscription plan not found")
    ErrLimitExceeded       = errors.New("subscription limit exceeded")
    ErrNoCounterRegistered = errors.New("no usage counter registered for resource")
    ErrTrialExpired        = errors.New("subscription trial has expired")
    ErrSubscriptionNotFound = errors.New("subscription not found")
)

// Usage:
if errors.Is(err, subscription.ErrLimitExceeded) {
    // handle limit exceeded
}
```

## Configuration

```go
// Implement resource counters
func userCounter(ctx context.Context, tenantID uuid.UUID) (int64, error) {
    return db.CountUsers(ctx, tenantID)
}

// Custom plan ID resolver (default uses context)
func customResolver(ctx context.Context, tenantID uuid.UUID) (string, error) {
    return db.GetPlanID(ctx, tenantID)
}

svc, err := subscription.NewService(
    ctx, plansSource, provider, store,
    subscription.WithPlanIDResolver(customResolver),
)
```

## API Documentation

```bash
# Full API documentation
go doc github.com/dmitrymomot/saaskit/svc/subscription

# Specific function or type
go doc github.com/dmitrymomot/saaskit/svc/subscription.Service
```

## Notes

- Resource counters must be fast (use caching or database aggregates)
- Plan IDs should match your payment provider's price IDs for paid plans
- Registering the same resource counter twice will panic - this is intentional to prevent configuration errors
