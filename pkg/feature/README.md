# Feature

A flexible feature flag management package for Go applications supporting multiple rollout strategies.

## Overview

The feature package provides a comprehensive feature flag system with provider-based architecture and sophisticated rollout strategies. It enables controlled feature releases through user targeting, percentage rollouts, environment-based activation, and composite strategies.

## Internal Usage

This package is internal to the project and provides feature flag capabilities for controlled feature rollouts and A/B testing across saaskit applications.

## Features

- Provider-based architecture with in-memory implementation
- Multiple rollout strategies (always on/off, targeted, environment, composite)
- User and group targeting with allow/deny lists
- Percentage-based gradual rollouts
- Thread-safe operations for concurrent access
- Context-based evaluation with custom extractors

## Usage

### Basic Example

```go
import "github.com/dmitrymomot/saaskit/pkg/feature"

// Create an in-memory provider
provider, err := feature.NewMemoryProvider(
    &feature.Flag{
        Name:        "new-dashboard",
        Description: "Enable new dashboard UI",
        Enabled:     true,
        Strategy:    feature.NewAlwaysOnStrategy(),
    },
)

// Check if feature is enabled
enabled, err := provider.IsEnabled(ctx, "new-dashboard")
// enabled: true
```

### Additional Usage Scenarios

```go
// Percentage-based rollout
percentageFlag := &feature.Flag{
    Name:    "beta-feature",
    Enabled: true,
    Strategy: feature.NewTargetedStrategy(
        feature.TargetCriteria{
            Percentage: intPtr(25), // 25% of users
        },
        feature.WithUserIDExtractor(func(ctx context.Context) string {
            // Extract user ID from context
            return ctx.Value("userID").(string)
        }),
    ),
}

// Environment-based feature
envFlag := &feature.Flag{
    Name:    "debug-mode",
    Enabled: true,
    Strategy: feature.NewEnvironmentStrategy(
        []string{"development", "staging"},
        feature.WithEnvironmentExtractor(func(ctx context.Context) string {
            return ctx.Value("environment").(string)
        }),
    ),
}

// Composite strategy (AND logic)
compositeFlag := &feature.Flag{
    Name:    "premium-beta",
    Enabled: true,
    Strategy: feature.NewAndStrategy(
        feature.NewEnvironmentStrategy([]string{"production"}),
        feature.NewTargetedStrategy(feature.TargetCriteria{
            Groups: []string{"premium-users"},
        }),
    ),
}
```

### Error Handling

```go
// Check for specific errors
flag, err := provider.GetFlag(ctx, "unknown-flag")
if errors.Is(err, feature.ErrFlagNotFound) {
    // Handle missing flag
}

// Create flag with validation
err = provider.CreateFlag(ctx, &feature.Flag{Name: ""})
if errors.Is(err, feature.ErrInvalidFlag) {
    // Handle invalid flag configuration
}
```

## Best Practices

### Integration Guidelines

- Initialize providers early in application startup
- Use context extractors to decouple feature evaluation from business logic
- Implement fallback behavior for feature flag failures

### Project-Specific Considerations

- Cache feature flag evaluations for performance-critical paths
- Use targeted strategies for gradual rollouts in production
- Leverage composite strategies for complex activation rules

## API Reference

### Configuration Variables

```go
// No exported configuration variables
```

### Types

```go
type Flag struct {
    Name        string      `json:"name"`
    Description string      `json:"description,omitempty"`
    Enabled     bool        `json:"enabled"`
    Strategy    Strategy    `json:"strategy,omitempty"`
    Tags        []string    `json:"tags,omitempty"`
    CreatedAt   time.Time   `json:"created_at,omitzero"`
    UpdatedAt   time.Time   `json:"updated_at,omitzero"`
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

type Provider interface {
    IsEnabled(ctx context.Context, flagName string) (bool, error)
    GetFlag(ctx context.Context, flagName string) (*Flag, error)
    ListFlags(ctx context.Context, tags ...string) ([]*Flag, error)
    CreateFlag(ctx context.Context, flag *Flag) error
    UpdateFlag(ctx context.Context, flag *Flag) error
    DeleteFlag(ctx context.Context, flagName string) error
    Close() error
}

type MemoryProvider struct {
    // Internal fields
}

type AlwaysStrategy struct {
    Value bool
}

type TargetedStrategy struct {
    Criteria TargetCriteria
    // Internal fields
}

type EnvironmentStrategy struct {
    EnabledEnvironments []string
    // Internal fields
}

type CompositeStrategy struct {
    Strategies []Strategy
    Operator   string // "and" or "or"
}

// Extractor function types
type UserIDExtractor func(ctx context.Context) string
type UserGroupsExtractor func(ctx context.Context) []string
type EnvironmentExtractor func(ctx context.Context) string
type TargetedStrategyOption func(*TargetedStrategy)
type EnvironmentStrategyOption func(*EnvironmentStrategy)
```

### Functions

```go
func NewMemoryProvider(initialFlags ...*Flag) (*MemoryProvider, error)
func NewAlwaysOnStrategy() Strategy
func NewAlwaysOffStrategy() Strategy
func NewTargetedStrategy(criteria TargetCriteria, opts ...TargetedStrategyOption) Strategy
func NewEnvironmentStrategy(environments []string, opts ...EnvironmentStrategyOption) Strategy
func NewAndStrategy(strategies ...Strategy) Strategy
func NewOrStrategy(strategies ...Strategy) Strategy
func WithUserIDExtractor(extractor UserIDExtractor) TargetedStrategyOption
func WithUserGroupsExtractor(extractor UserGroupsExtractor) TargetedStrategyOption
func WithEnvironmentExtractor(extractor EnvironmentExtractor) EnvironmentStrategyOption
```

### Methods

```go
// MemoryProvider methods
func (m *MemoryProvider) IsEnabled(ctx context.Context, flagName string) (bool, error)
func (m *MemoryProvider) GetFlag(ctx context.Context, flagName string) (*Flag, error)
func (m *MemoryProvider) ListFlags(ctx context.Context, tags ...string) ([]*Flag, error)
func (m *MemoryProvider) CreateFlag(ctx context.Context, flag *Flag) error
func (m *MemoryProvider) UpdateFlag(ctx context.Context, flag *Flag) error
func (m *MemoryProvider) DeleteFlag(ctx context.Context, flagName string) error
func (m *MemoryProvider) Close() error

// Strategy methods
func (s *AlwaysStrategy) Evaluate(ctx context.Context) (bool, error)
func (s *TargetedStrategy) Evaluate(ctx context.Context) (bool, error)
func (s *EnvironmentStrategy) Evaluate(ctx context.Context) (bool, error)
func (s *CompositeStrategy) Evaluate(ctx context.Context) (bool, error)
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
