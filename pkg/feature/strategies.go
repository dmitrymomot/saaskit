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

	// Extractors for retrieving data from context.
	userIDExtractor     UserIDExtractor
	userGroupsExtractor UserGroupsExtractor
}

// Evaluate determines if a feature should be enabled based on the context and criteria.
func (s *TargetedStrategy) Evaluate(ctx context.Context) (bool, error) {
	// Check for nil criteria
	if s.isEmptyCriteria() {
		return false, ErrInvalidStrategy
	}

	var userID string
	if s.userIDExtractor != nil {
		userID = s.userIDExtractor(ctx)
	}

	// Check deny list first (always takes precedence)
	if s.isInDenyList(userID) {
		return false, nil
	}

	// Check allow list (if a user is on the allow list, they get the feature)
	if s.isInAllowList(userID) {
		return true, nil
	}

	// Check for specific user IDs
	if s.isTargetedUser(userID) {
		return true, nil
	}

	// Check for groups
	if s.isInTargetedGroup(ctx) {
		return true, nil
	}

	// Check for percentage rollout
	if s.Criteria.Percentage != nil {
		return s.evaluatePercentage(userID)
	}

	// If we've gone through all criteria and nothing matched, return false
	return false, nil
}

// isEmptyCriteria checks if all criteria are nil
func (s *TargetedStrategy) isEmptyCriteria() bool {
	return s.Criteria.UserIDs == nil && s.Criteria.Groups == nil &&
		s.Criteria.Percentage == nil && s.Criteria.AllowList == nil &&
		s.Criteria.DenyList == nil
}

// isInDenyList checks if user is in the deny list
func (s *TargetedStrategy) isInDenyList(userID string) bool {
	if len(s.Criteria.DenyList) == 0 {
		return false
	}

	// If we can't determine the user ID and there's a deny list, fail safe
	if userID == "" {
		return true
	}

	return slices.Contains(s.Criteria.DenyList, userID)
}

// isInAllowList checks if user is in the allow list
func (s *TargetedStrategy) isInAllowList(userID string) bool {
	return len(s.Criteria.AllowList) > 0 && userID != "" &&
		slices.Contains(s.Criteria.AllowList, userID)
}

// isTargetedUser checks if user is in the targeted user IDs
func (s *TargetedStrategy) isTargetedUser(userID string) bool {
	return len(s.Criteria.UserIDs) > 0 && userID != "" &&
		slices.Contains(s.Criteria.UserIDs, userID)
}

// isInTargetedGroup checks if user belongs to any targeted group
func (s *TargetedStrategy) isInTargetedGroup(ctx context.Context) bool {
	if len(s.Criteria.Groups) == 0 || s.userGroupsExtractor == nil {
		return false
	}

	userGroups := s.userGroupsExtractor(ctx)
	if len(userGroups) == 0 {
		return false
	}

	// Check if any user group is in the targeted groups
	for _, userGroup := range userGroups {
		if slices.Contains(s.Criteria.Groups, userGroup) {
			return true
		}
	}

	return false
}

// evaluatePercentage checks if user falls within the percentage rollout
func (s *TargetedStrategy) evaluatePercentage(userID string) (bool, error) {
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
	if userID == "" {
		return false, nil
	}

	// Determine if this user is within the percentage
	hash := fnv.New32a()
	hash.Write([]byte(userID))
	hashValue := hash.Sum32() % 100
	return int(hashValue) < percentage, nil
}

// TargetedStrategyOption is a function that configures a TargetedStrategy.
type TargetedStrategyOption func(*TargetedStrategy)

// WithUserIDExtractor sets the user ID extractor for the strategy.
func WithUserIDExtractor(extractor UserIDExtractor) TargetedStrategyOption {
	return func(s *TargetedStrategy) {
		s.userIDExtractor = extractor
	}
}

// WithUserGroupsExtractor sets the user groups extractor for the strategy.
func WithUserGroupsExtractor(extractor UserGroupsExtractor) TargetedStrategyOption {
	return func(s *TargetedStrategy) {
		s.userGroupsExtractor = extractor
	}
}

// NewTargetedStrategy creates a strategy based on targeting criteria.
func NewTargetedStrategy(criteria TargetCriteria, opts ...TargetedStrategyOption) Strategy {
	s := &TargetedStrategy{
		Criteria: criteria,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// EnvironmentStrategy enables features based on the environment.
type EnvironmentStrategy struct {
	// EnabledEnvironments lists environments where the feature is enabled.
	EnabledEnvironments []string

	// Extractor for retrieving environment from context.
	environmentExtractor EnvironmentExtractor
}

// Evaluate checks if the feature should be enabled for the current environment.
func (s *EnvironmentStrategy) Evaluate(ctx context.Context) (bool, error) {
	if len(s.EnabledEnvironments) == 0 {
		return false, ErrInvalidStrategy
	}

	if s.environmentExtractor == nil {
		return false, nil
	}

	// Extract environment from context
	env := s.environmentExtractor(ctx)
	if env == "" {
		return false, nil
	}

	// Check if the environment is in the enabled list
	return slices.Contains(s.EnabledEnvironments, env), nil
}

// EnvironmentStrategyOption is a function that configures an EnvironmentStrategy.
type EnvironmentStrategyOption func(*EnvironmentStrategy)

// WithEnvironmentExtractor sets the environment extractor for the strategy.
func WithEnvironmentExtractor(extractor EnvironmentExtractor) EnvironmentStrategyOption {
	return func(s *EnvironmentStrategy) {
		s.environmentExtractor = extractor
	}
}

// NewEnvironmentStrategy creates a strategy that enables features in specific environments.
func NewEnvironmentStrategy(environments []string, opts ...EnvironmentStrategyOption) Strategy {
	s := &EnvironmentStrategy{
		EnabledEnvironments: environments,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
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
