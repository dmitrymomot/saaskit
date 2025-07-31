package feature

import (
	"context"
	"errors"
	"hash/fnv"
	"slices"
)

type AlwaysStrategy struct {
	Value bool
}

func (s *AlwaysStrategy) Evaluate(ctx context.Context) (bool, error) {
	return s.Value, nil
}

func NewAlwaysOnStrategy() Strategy {
	return &AlwaysStrategy{Value: true}
}

func NewAlwaysOffStrategy() Strategy {
	return &AlwaysStrategy{Value: false}
}

type TargetedStrategy struct {
	Criteria TargetCriteria

	userIDExtractor     UserIDExtractor
	userGroupsExtractor UserGroupsExtractor
}

// Evaluate determines feature enablement using a strict precedence hierarchy:
// 1. DenyList (highest) - blocks access regardless of other criteria
// 2. AllowList - grants access overriding user/group/percentage rules
// 3. UserIDs - direct user targeting
// 4. Groups - group membership targeting
// 5. Percentage - consistent hash-based rollout (lowest precedence)
func (s *TargetedStrategy) Evaluate(ctx context.Context) (bool, error) {
	if s.isEmptyCriteria() {
		return false, ErrInvalidStrategy
	}

	var userID string
	if s.userIDExtractor != nil {
		userID = s.userIDExtractor(ctx)
	}

	if s.isInDenyList(userID) {
		return false, nil
	}

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

	// Fail-safe: deny access when user identity is unknown but deny list exists
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

// evaluatePercentage uses FNV-1a hash for consistent user bucketing.
// Same user always gets same result, ensuring stable feature rollouts.
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

	// Hash userID to get consistent 0-99 bucket assignment
	hash := fnv.New32a()
	hash.Write([]byte(userID))
	hashValue := hash.Sum32() % 100
	return int(hashValue) < percentage, nil
}

type TargetedStrategyOption func(*TargetedStrategy)

func WithUserIDExtractor(extractor UserIDExtractor) TargetedStrategyOption {
	return func(s *TargetedStrategy) {
		s.userIDExtractor = extractor
	}
}

func WithUserGroupsExtractor(extractor UserGroupsExtractor) TargetedStrategyOption {
	return func(s *TargetedStrategy) {
		s.userGroupsExtractor = extractor
	}
}

func NewTargetedStrategy(criteria TargetCriteria, opts ...TargetedStrategyOption) Strategy {
	s := &TargetedStrategy{
		Criteria: criteria,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type EnvironmentStrategy struct {
	EnabledEnvironments  []string
	environmentExtractor EnvironmentExtractor
}

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

type EnvironmentStrategyOption func(*EnvironmentStrategy)

func WithEnvironmentExtractor(extractor EnvironmentExtractor) EnvironmentStrategyOption {
	return func(s *EnvironmentStrategy) {
		s.environmentExtractor = extractor
	}
}

func NewEnvironmentStrategy(environments []string, opts ...EnvironmentStrategyOption) Strategy {
	s := &EnvironmentStrategy{
		EnabledEnvironments: environments,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type CompositeStrategy struct {
	Strategies []Strategy
	Operator   string // "and" or "or"
}

// Evaluate combines multiple strategies with short-circuit evaluation.
// "and": returns false on first false result, "or": returns true on first true result.
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

func NewAndStrategy(strategies ...Strategy) Strategy {
	return &CompositeStrategy{
		Strategies: strategies,
		Operator:   "and",
	}
}

func NewOrStrategy(strategies ...Strategy) Strategy {
	return &CompositeStrategy{
		Strategies: strategies,
		Operator:   "or",
	}
}
