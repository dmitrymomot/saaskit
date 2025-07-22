package feature

import (
	"context"
	"time"
)

// Flag represents a feature flag with its configuration.
type Flag struct {
	// Name is the unique identifier for the flag.
	Name string `json:"name"`

	// Description is a human-readable description of the flag's purpose.
	Description string `json:"description,omitempty"`

	// Enabled indicates if the flag is globally enabled or disabled.
	Enabled bool `json:"enabled"`

	// Strategy defines how the flag is rolled out to users/services.
	Strategy Strategy `json:"strategy,omitempty"`

	// Tags are optional metadata for categorizing flags.
	Tags []string `json:"tags,omitempty"`

	// CreatedAt is when the flag was created.
	CreatedAt time.Time `json:"created_at,omitzero"`

	// UpdatedAt is when the flag was last updated.
	UpdatedAt time.Time `json:"updated_at,omitzero"`
}

// Strategy defines different ways to roll out a feature.
type Strategy interface {
	// Evaluate determines if the feature should be enabled for a specific context.
	// Context should contain data required by the strategy (user ID, groups, etc.).
	Evaluate(ctx context.Context) (bool, error)
}

// TargetCriteria defines targeting criteria for a flag.
type TargetCriteria struct {
	// UserIDs is a list of specific user IDs for which the flag should be enabled.
	UserIDs []string `json:"user_ids,omitempty"`

	// Groups is a list of group names for which the flag should be enabled.
	Groups []string `json:"groups,omitempty"`

	// Percentage defines what percentage of users should have the flag enabled.
	Percentage *int `json:"percentage,omitempty"`

	// AllowList identifies entities that should always get the feature.
	AllowList []string `json:"allow_list,omitempty"`

	// DenyList identifies entities that should never get the feature.
	DenyList []string `json:"deny_list,omitempty"`
}

// ContextKey is the type used for context keys.
type ContextKey string

// Context keys for accessing flag evaluation data.
const (
	// UserIDKey is the context key for user ID.
	UserIDKey ContextKey = "user_id"

	// UserGroupsKey is the context key for user groups.
	UserGroupsKey ContextKey = "user_groups"

	// EnvironmentKey is the context key for the environment.
	EnvironmentKey ContextKey = "environment"
)

// Provider is the interface that all feature flag providers must implement.
type Provider interface {
	// IsEnabled checks if a feature flag is enabled for the given context.
	// If the flag doesn't exist, it returns false and ErrFlagNotFound.
	IsEnabled(ctx context.Context, flagName string) (bool, error)

	// GetFlag returns the full flag configuration.
	// If the flag doesn't exist, it returns nil and ErrFlagNotFound.
	GetFlag(ctx context.Context, flagName string) (*Flag, error)

	// ListFlags returns all available flags, optionally filtered by tags.
	ListFlags(ctx context.Context, tags ...string) ([]*Flag, error)

	// CreateFlag creates a new feature flag.
	// If a flag with the same name already exists, it returns an error.
	CreateFlag(ctx context.Context, flag *Flag) error

	// UpdateFlag updates an existing feature flag.
	// If the flag doesn't exist, it returns ErrFlagNotFound.
	UpdateFlag(ctx context.Context, flag *Flag) error

	// DeleteFlag deletes a feature flag.
	// If the flag doesn't exist, it returns ErrFlagNotFound.
	DeleteFlag(ctx context.Context, flagName string) error

	// Close releases any resources used by the provider.
	Close() error
}
