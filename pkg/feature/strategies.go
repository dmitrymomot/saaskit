package feature

import (
	"context"
	"errors"
	"hash/fnv"
	"slices"
)

// AlwaysStrategy is a strategy that always returns the same value.
type AlwaysStrategy struct {
	Value bool
}

// Evaluate returns the configured value for all contexts.
func (s *AlwaysStrategy) Evaluate(ctx context.Context) (bool, error) {
	return s.Value, nil
}

// NewAlwaysOnStrategy creates a strategy that enables the feature for all users.
func NewAlwaysOnStrategy() Strategy {
	return &AlwaysStrategy{Value: true}
}

// NewAlwaysOffStrategy creates a strategy that disables the feature for all users.
func NewAlwaysOffStrategy() Strategy {
	return &AlwaysStrategy{Value: false}
}

// TargetedStrategy enables features for specific users, groups, or percentages.
type TargetedStrategy struct {
	// Criteria for enabling the feature.
	Criteria TargetCriteria
}

// Evaluate determines if a feature should be enabled based on the context and criteria.
func (s *TargetedStrategy) Evaluate(ctx context.Context) (bool, error) {
	// Check for nil criteria
	if s.Criteria.UserIDs == nil && s.Criteria.Groups == nil &&
		s.Criteria.Percentage == nil && s.Criteria.AllowList == nil &&
		s.Criteria.DenyList == nil {
		return false, ErrInvalidStrategy
	}

	// Check deny list first (always takes precedence)
	if len(s.Criteria.DenyList) > 0 {
		userID, ok := ctx.Value(UserIDKey).(string)
		if !ok || userID == "" {
			// If we can't determine the user ID and there's a deny list, fail safe
			return false, nil
		}

		if slices.Contains(s.Criteria.DenyList, userID) {
			return false, nil
		}
	}

	// Check allow list (if a user is on the allow list, they get the feature)
	if len(s.Criteria.AllowList) > 0 {
		userID, ok := ctx.Value(UserIDKey).(string)
		if ok && userID != "" && slices.Contains(s.Criteria.AllowList, userID) {
			return true, nil
		}
	}

	// Check for specific user IDs
	if len(s.Criteria.UserIDs) > 0 {
		userID, ok := ctx.Value(UserIDKey).(string)
		if ok && userID != "" && slices.Contains(s.Criteria.UserIDs, userID) {
			return true, nil
		}
	}

	// Check for groups
	if len(s.Criteria.Groups) > 0 {
		userGroups, ok := ctx.Value(UserGroupsKey).([]string)
		if ok && len(userGroups) > 0 {
			// Check if any user group is in the targeted groups
			for _, userGroup := range userGroups {
				if slices.Contains(s.Criteria.Groups, userGroup) {
					return true, nil
				}
			}
		}
	}

	// Check for percentage rollout
	if s.Criteria.Percentage != nil {
		percentage := *s.Criteria.Percentage
		if percentage < 0 || percentage > 100 {
			return false, errors.Join(ErrInvalidStrategy,
				errors.New("percentage must be between 0 and 100"))
		}

		// If percentage is 0, feature is off for everyone
		if percentage == 0 {
			return false, nil
		}

		// If percentage is 100, feature is on for everyone
		if percentage == 100 {
			return true, nil
		}

		// We need a user ID for percentage-based rollouts
		userID, ok := ctx.Value(UserIDKey).(string)
		if !ok || userID == "" {
			return false, nil
		}

		// Determine if this user is within the percentage
		hash := fnv.New32a()
		hash.Write([]byte(userID))
		hashValue := hash.Sum32() % 100
		return int(hashValue) < percentage, nil
	}

	// If we've gone through all criteria and nothing matched, return false
	return false, nil
}

// NewTargetedStrategy creates a strategy based on targeting criteria.
func NewTargetedStrategy(criteria TargetCriteria) Strategy {
	return &TargetedStrategy{
		Criteria: criteria,
	}
}

// EnvironmentStrategy enables features based on the environment.
type EnvironmentStrategy struct {
	// EnabledEnvironments lists environments where the feature is enabled.
	EnabledEnvironments []string
}

// Evaluate checks if the feature should be enabled for the current environment.
func (s *EnvironmentStrategy) Evaluate(ctx context.Context) (bool, error) {
	if len(s.EnabledEnvironments) == 0 {
		return false, ErrInvalidStrategy
	}

	// Extract environment from context
	env, ok := ctx.Value(EnvironmentKey).(string)
	if !ok || env == "" {
		return false, nil
	}

	// Check if the environment is in the enabled list
	if slices.Contains(s.EnabledEnvironments, env) {
		return true, nil
	}

	return false, nil
}

// NewEnvironmentStrategy creates a strategy that enables features in specific environments.
func NewEnvironmentStrategy(environments ...string) Strategy {
	return &EnvironmentStrategy{
		EnabledEnvironments: environments,
	}
}

// CompositeStrategy combines multiple strategies with an operator.
type CompositeStrategy struct {
	Strategies []Strategy
	Operator   string // "and" or "or"
}

// Evaluate combines the results of multiple strategies.
func (s *CompositeStrategy) Evaluate(ctx context.Context) (bool, error) {
	if len(s.Strategies) == 0 {
		return false, ErrInvalidStrategy
	}

	switch s.Operator {
	case "and":
		// All strategies must return true
		for _, strategy := range s.Strategies {
			enabled, err := strategy.Evaluate(ctx)
			if err != nil {
				return false, err
			}
			if !enabled {
				return false, nil
			}
		}
		return true, nil

	case "or":
		// At least one strategy must return true
		for _, strategy := range s.Strategies {
			enabled, err := strategy.Evaluate(ctx)
			if err != nil {
				return false, err
			}
			if enabled {
				return true, nil
			}
		}
		return false, nil

	default:
		return false, errors.Join(ErrInvalidStrategy,
			errors.New("composite operator must be 'and' or 'or'"))
	}
}

// NewAndStrategy creates a strategy that requires all child strategies to return true.
func NewAndStrategy(strategies ...Strategy) Strategy {
	return &CompositeStrategy{
		Strategies: strategies,
		Operator:   "and",
	}
}

// NewOrStrategy creates a strategy that requires at least one child strategy to return true.
func NewOrStrategy(strategies ...Strategy) Strategy {
	return &CompositeStrategy{
		Strategies: strategies,
		Operator:   "or",
	}
}
