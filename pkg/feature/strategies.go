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
	Criteria TargetCriteria

	userIDExtractor     UserIDExtractor
	userGroupsExtractor UserGroupsExtractor
}

// Evaluate determines if a feature should be enabled based on the context and criteria.
// Evaluation order: deny list → allow list → user IDs → groups → percentage rollout
func (s *TargetedStrategy) Evaluate(ctx context.Context) (bool, error) {
	if s.isEmptyCriteria() {
		return false, ErrInvalidStrategy
	}

	var userID string
	if s.userIDExtractor != nil {
		userID = s.userIDExtractor(ctx)
	}

	// Deny list always takes precedence
	if s.isInDenyList(userID) {
		return false, nil
	}

	// Allow list overrides all other criteria except deny list
	if s.isInAllowList(userID) {
		return true, nil
	}

	if s.isTargetedUser(userID) {
		return true, nil
	}

	if s.isInTargetedGroup(ctx) {
		return true, nil
	}

	if s.Criteria.Percentage != nil {
		return s.evaluatePercentage(userID)
	}

	return false, nil
}

func (s *TargetedStrategy) isEmptyCriteria() bool {
	return s.Criteria.UserIDs == nil && s.Criteria.Groups == nil &&
		s.Criteria.Percentage == nil && s.Criteria.AllowList == nil &&
		s.Criteria.DenyList == nil
}

func (s *TargetedStrategy) isInDenyList(userID string) bool {
	if len(s.Criteria.DenyList) == 0 {
		return false
	}

	// Fail safe: if we can't determine user ID and there's a deny list, deny access
	if userID == "" {
		return true
	}

	return slices.Contains(s.Criteria.DenyList, userID)
}

func (s *TargetedStrategy) isInAllowList(userID string) bool {
	return len(s.Criteria.AllowList) > 0 && userID != "" &&
		slices.Contains(s.Criteria.AllowList, userID)
}

func (s *TargetedStrategy) isTargetedUser(userID string) bool {
	return len(s.Criteria.UserIDs) > 0 && userID != "" &&
		slices.Contains(s.Criteria.UserIDs, userID)
}

func (s *TargetedStrategy) isInTargetedGroup(ctx context.Context) bool {
	if len(s.Criteria.Groups) == 0 || s.userGroupsExtractor == nil {
		return false
	}

	userGroups := s.userGroupsExtractor(ctx)
	if len(userGroups) == 0 {
		return false
	}

	for _, userGroup := range userGroups {
		if slices.Contains(s.Criteria.Groups, userGroup) {
			return true
		}
	}

	return false
}

// evaluatePercentage uses consistent hashing to determine rollout eligibility
func (s *TargetedStrategy) evaluatePercentage(userID string) (bool, error) {
	percentage := *s.Criteria.Percentage
	if percentage < 0 || percentage > 100 {
		return false, errors.Join(ErrInvalidStrategy,
			errors.New("percentage must be between 0 and 100"))
	}

	if percentage == 0 {
		return false, nil
	}

	if percentage == 100 {
		return true, nil
	}

	if userID == "" {
		return false, nil
	}

	// Use consistent hashing to determine if user falls within percentage
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
	EnabledEnvironments  []string
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

	env := s.environmentExtractor(ctx)
	if env == "" {
		return false, nil
	}

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
