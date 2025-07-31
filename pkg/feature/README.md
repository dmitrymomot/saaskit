# Feature

Feature flag management system for controlled feature rollouts and A/B testing.

## Features

- **Flexible Strategies** - User targeting, percentage rollouts, environment-based activation
- **Provider Architecture** - Pluggable backend storage with in-memory implementation
- **Thread-Safe Operations** - Concurrent access with read-write locks
- **Consistent Rollouts** - Hash-based percentage distribution ensures stable user experience

## Installation

```go
import "github.com/dmitrymomot/saaskit/pkg/feature"
```

## Usage

```go
package main

import (
    "context"
    "log"

    "github.com/dmitrymomot/saaskit/pkg/feature"
)

func main() {
    // Create provider with initial flags
    provider, err := feature.NewMemoryProvider(
        &feature.Flag{
            Name:        "new-ui",
            Description: "Enable redesigned user interface",
            Enabled:     true,
            Strategy:    feature.NewAlwaysOnStrategy(),
        },
    )
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Close()

    // Check if feature is enabled
    ctx := context.Background()
    enabled, err := provider.IsEnabled(ctx, "new-ui")
    if err != nil {
        log.Fatal(err)
    }

    if enabled {
        // Show new UI
    }
}
```

## Common Operations

### Percentage-Based Rollout

```go
// Roll out to 25% of users
percentage := 25
flag := &feature.Flag{
    Name:    "experimental-feature",
    Enabled: true,
    Strategy: feature.NewTargetedStrategy(
        feature.TargetCriteria{
            Percentage: &percentage,
        },
        feature.WithUserIDExtractor(func(ctx context.Context) string {
            // Extract user ID from your context
            return ctx.Value("userID").(string)
        }),
    ),
}
```

### User and Group Targeting

```go
// Target specific users and groups
flag := &feature.Flag{
    Name:    "beta-feature",
    Enabled: true,
    Strategy: feature.NewTargetedStrategy(
        feature.TargetCriteria{
            UserIDs:  []string{"user123", "user456"},
            Groups:   []string{"beta-testers", "employees"},
            DenyList: []string{"user789"}, // Always exclude these users
        },
        feature.WithUserIDExtractor(getUserID),
        feature.WithUserGroupsExtractor(getUserGroups),
    ),
}
```

### Environment-Based Features

```go
// Enable in specific environments
flag := &feature.Flag{
    Name:    "debug-mode",
    Enabled: true,
    Strategy: feature.NewEnvironmentStrategy(
        []string{"development", "staging"},
        feature.WithEnvironmentExtractor(func(ctx context.Context) string {
            return os.Getenv("APP_ENV")
        }),
    ),
}
```

### Composite Strategies

```go
// Combine multiple strategies
flag := &feature.Flag{
    Name:    "complex-feature",
    Enabled: true,
    Strategy: feature.NewAndStrategy(
        feature.NewEnvironmentStrategy([]string{"production"}, envExtractor),
        feature.NewTargetedStrategy(
            feature.TargetCriteria{Percentage: &fifty},
            feature.WithUserIDExtractor(getUserID),
        ),
    ),
}
```

## Error Handling

```go
// Package errors:
var (
    ErrFlagNotFound           = errors.New("feature flag not found")
    ErrInvalidFlag            = errors.New("invalid feature flag parameters")
    ErrProviderNotInitialized = errors.New("feature provider not initialized")
    ErrInvalidContext         = errors.New("invalid context for feature evaluation")
    ErrInvalidStrategy        = errors.New("invalid feature rollout strategy")
    ErrOperationFailed        = errors.New("feature operation failed")
)

// Usage:
flag, err := provider.GetFlag(ctx, "unknown")
if errors.Is(err, feature.ErrFlagNotFound) {
    // Flag doesn't exist
}
```

## API Documentation

```bash
# Full API documentation
go doc github.com/dmitrymomot/saaskit/pkg/feature

# Specific function or type
go doc github.com/dmitrymomot/saaskit/pkg/feature.Provider
```

## Notes

- Percentage rollouts use FNV-1a hashing for consistent user bucketing
- DenyList has highest precedence in TargetedStrategy evaluation hierarchy
- MemoryProvider creates deep copies to prevent external flag modification
