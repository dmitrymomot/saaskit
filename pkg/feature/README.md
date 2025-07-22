# Feature Flags

A flexible and extensible feature flagging system for Go applications.

## Overview

The `feature` package provides a robust feature flagging system to control feature rollouts in your Go applications. It supports various strategies including percentage-based rollouts, targeted users/groups, environment-based flags, and composite conditions. The package is designed to be thread-safe and suitable for concurrent usage across your application.

## Features

- Pluggable storage backends with a generic `Provider` interface
- Ready-to-use in-memory implementation for quick setup and testing
- Multiple rollout strategies (always on/off, targeted, percentage-based, environment-based)
- Support for composite conditions with logical AND/OR operations
- Context-based evaluation for user-specific and environment-specific flags
- Tag-based flag organization for easier management
- Thread-safe implementations for concurrent usage

## Usage

### Basic Setup

```go
import (
	"context"
	"log"

	"github.com/dmitrymomot/saaskit/pkg/feature"
)

func main() {
	// Create a provider with initial flags
	provider, err := feature.NewMemoryProvider(
		&feature.Flag{
			Name:        "dark-mode",
			Description: "Enable dark mode UI",
			Enabled:     true,
			Tags:        []string{"ui", "theme"},
		},
		&feature.Flag{
			Name:        "beta-features",
			Description: "Enable beta features",
			Enabled:     true,
			Strategy:    feature.NewTargetedStrategy(feature.TargetCriteria{
				Groups: []string{"beta-users", "internal"},
			}),
		},
	)
	if err != nil {
		log.Fatalf("Failed to create feature provider: %v", err)
	}
	defer provider.Close()

	// Check if a feature is enabled
	ctx := context.Background()
	darkModeEnabled, err := provider.IsEnabled(ctx, "dark-mode")
	if err != nil {
		log.Printf("Error checking flag: %v", err)
	}

	if darkModeEnabled {
		// Enable dark mode UI
		log.Println("Dark mode is enabled!")
	}
}
```

### User-Specific Features

```go
// Add user information to context
ctx := context.Background()
ctx = context.WithValue(ctx, feature.UserIDKey, "user-123")
ctx = context.WithValue(ctx, feature.UserGroupsKey, []string{"beta-users"})

// Check if beta features are enabled for this user
betaEnabled, err := provider.IsEnabled(ctx, "beta-features")
// Returns: true if user is in "beta-users" group, false otherwise
```

### Rollout Strategies

```go
// Always on strategy
alwaysOn := feature.NewAlwaysOnStrategy()

// Environment-based strategy
envStrategy := feature.NewEnvironmentStrategy("dev", "staging")

// Targeted strategy
targetedStrategy := feature.NewTargetedStrategy(feature.TargetCriteria{
	UserIDs:    []string{"user-1", "user-2"},  // Specific users
	Groups:     []string{"beta", "internal"},  // User groups
	Percentage: intPtr(20),                    // 20% of users
	AllowList:  []string{"vip-1"},             // Always enabled
	DenyList:   []string{"banned-user"},       // Never enabled
})

// Composite strategy (both conditions must be true)
compositeStrategy := feature.NewAndStrategy(
	envStrategy,      // Must be in dev/staging
	targetedStrategy, // Must match target criteria
)

// Helper function for percentage
func intPtr(i int) *int {
	return &i
}
```

### Managing Flags

```go
// Create a new flag
newFlag := &feature.Flag{
	Name:        "new-feature",
	Description: "A new experimental feature",
	Enabled:     true,
	Strategy:    feature.NewTargetedStrategy(feature.TargetCriteria{
		Percentage: intPtr(10), // 10% rollout
	}),
	Tags: []string{"experimental"},
}
err = provider.CreateFlag(ctx, newFlag)

// Update an existing flag
flag, err := provider.GetFlag(ctx, "new-feature")
if err != nil {
	// Handle error
}
flag.Strategy = feature.NewTargetedStrategy(feature.TargetCriteria{
	Percentage: intPtr(50), // Increase to 50% rollout
})
err = provider.UpdateFlag(ctx, flag)

// Delete a flag
err = provider.DeleteFlag(ctx, "deprecated-feature")
```

### Listing Flags

```go
// List all flags
allFlags, err := provider.ListFlags(ctx)

// List flags by tags (flags with any of these tags)
uiFlags, err := provider.ListFlags(ctx, "ui", "theme")
```

## Best Practices

1. **Configuration Management**:
    - Keep flag definitions in a central location for easier maintenance
    - Document the purpose of each flag in its description field
    - Use meaningful names and consistent naming conventions

2. **Rollout Strategy**:
    - Start with small percentage rollouts for risky features
    - Use environment-based strategies for proper staging
    - Leverage allow-lists for internal testing before wider rollout

3. **Context Usage**:
    - Add all required data to the context early in your request lifecycle
    - Standardize how user IDs and groups are populated in your application
    - Consider creating middleware to automatically enrich context with user data

4. **Error Handling**:
    - Always check errors from feature flag operations
    - Implement graceful fallbacks when flags cannot be evaluated
    - Log flag evaluation failures for debugging

## API Reference

### Types

```go
type Flag struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Enabled     bool      `json:"enabled"`
	Strategy    Strategy  `json:"strategy,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type Strategy interface {
	Evaluate(ctx context.Context) (bool, error)
}

type TargetCriteria struct {
	UserIDs    []string `json:"user_ids,omitempty"`
	Groups     []string `json:"groups,omitempty"`
	Percentage *int     `json:"percentage,omitempty"`
	AllowList  []string `json:"allow_list,omitempty"`
	DenyList   []string `json:"deny_list,omitempty"`
}
```

### Context Keys

```go
type ContextKey string

const (
	UserIDKey      ContextKey = "user_id"
	UserGroupsKey  ContextKey = "user_groups"
	EnvironmentKey ContextKey = "environment"
)
```

### Provider Interface

```go
type Provider interface {
	IsEnabled(ctx context.Context, flagName string) (bool, error)
	GetFlag(ctx context.Context, flagName string) (*Flag, error)
	ListFlags(ctx context.Context, tags ...string) ([]*Flag, error)
	CreateFlag(ctx context.Context, flag *Flag) error
	UpdateFlag(ctx context.Context, flag *Flag) error
	DeleteFlag(ctx context.Context, flagName string) error
	Close() error
}
```

### Factory Functions

```go
func NewMemoryProvider(flags ...*Flag) (Provider, error)
func NewAlwaysOnStrategy() Strategy
func NewAlwaysOffStrategy() Strategy
func NewTargetedStrategy(criteria TargetCriteria) Strategy
func NewEnvironmentStrategy(environments ...string) Strategy
func NewAndStrategy(strategies ...Strategy) Strategy
func NewOrStrategy(strategies ...Strategy) Strategy
```

### Error Types

```go
var ErrFlagNotFound = errors.New("feature: flag not found")
var ErrInvalidFlag = errors.New("feature: invalid flag parameters")
var ErrProviderNotInitialized = errors.New("feature: provider not initialized")
var ErrInvalidContext = errors.New("feature: invalid context")
var ErrInvalidStrategy = errors.New("feature: invalid rollout strategy")
var ErrOperationFailed = errors.New("feature: operation failed")
```
