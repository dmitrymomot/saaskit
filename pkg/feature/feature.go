package feature

import (
	"context"
	"time"
)

// Flag represents a feature flag with its configuration.
type Flag struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Enabled     bool      `json:"enabled"`
	Strategy    Strategy  `json:"strategy,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitzero"`
	UpdatedAt   time.Time `json:"updated_at,omitzero"`
}

// Strategy defines different ways to roll out a feature.
type Strategy interface {
	// Evaluate determines if the feature should be enabled for a specific context.
	// Context should contain data required by the strategy (user ID, groups, etc.).
	Evaluate(ctx context.Context) (bool, error)
}

// TargetCriteria defines targeting criteria for a flag.
type TargetCriteria struct {
	UserIDs    []string `json:"user_ids,omitempty"`
	Groups     []string `json:"groups,omitempty"`
	Percentage *int     `json:"percentage,omitempty"`
	// AllowList overrides all other criteria except DenyList
	AllowList []string `json:"allow_list,omitempty"`
	// DenyList overrides all other criteria (highest precedence)
	DenyList []string `json:"deny_list,omitempty"`
}

// Extractor function types for retrieving data from context.
// These allow users to define how to extract feature flag evaluation data
// from their application's context, maintaining decoupling from the feature package.
type (
	UserIDExtractor      func(ctx context.Context) string
	UserGroupsExtractor  func(ctx context.Context) []string
	EnvironmentExtractor func(ctx context.Context) string
)

// Provider is the interface that all feature flag providers must implement.
type Provider interface {
	// Evaluation methods
	IsEnabled(ctx context.Context, flagName string) (bool, error)
	GetFlag(ctx context.Context, flagName string) (*Flag, error)

	// Management methods
	ListFlags(ctx context.Context, tags ...string) ([]*Flag, error)
	CreateFlag(ctx context.Context, flag *Flag) error
	UpdateFlag(ctx context.Context, flag *Flag) error
	DeleteFlag(ctx context.Context, flagName string) error

	// Lifecycle methods
	Close() error
}
