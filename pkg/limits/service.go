package limits

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

// LimitsService defines the public interface for interacting with tenant resource limits.
type LimitsService interface {
	// CanCreate checks if a tenant can create a new resource instance.
	CanCreate(ctx context.Context, tenantID uuid.UUID, res Resource) error

	// GetUsage returns the current usage and limit for a resource in a tenant.
	GetUsage(ctx context.Context, tenantID uuid.UUID, res Resource) (used, limit int64, err error)

	// GetUsageSafe is a convenience wrapper for UI dashboards. It returns zero values if usage cannot be obtained.
	GetUsageSafe(ctx context.Context, tenantID uuid.UUID, res Resource) (used, limit int64)

	// HasFeature quickly tells whether a feature is available for the current plan.
	HasFeature(ctx context.Context, tenantID uuid.UUID, feature Feature) bool

	// CheckTrial determines if a tenant's trial period is active for a specific plan.
	CheckTrial(ctx context.Context, tenantID uuid.UUID, startedAt time.Time) error

	// VerifyPlan checks if a plan ID is valid.
	// Takes context for symmetry with other methods, though not currently used.
	VerifyPlan(ctx context.Context, planID string) error

	// GetUsagePercentage returns usage as percentage (0-100, or -1 for unlimited).
	GetUsagePercentage(ctx context.Context, tenantID uuid.UUID, res Resource) int

	// CanDowngrade checks if downgrade is possible given current usage.
	CanDowngrade(ctx context.Context, tenantID uuid.UUID, targetPlanID string) error

	// GetAllUsage returns all resource usage for a tenant.
	GetAllUsage(ctx context.Context, tenantID uuid.UUID) (map[Resource]UsageInfo, error)
}

// Source defines how plans are loaded into the limits service.
type Source interface {
	Load(ctx context.Context) (map[string]Plan, error)
}

// PlanIDResolver resolves a plan ID for a given tenant.
type PlanIDResolver func(ctx context.Context, tenantID uuid.UUID) (string, error)

// service implements the LimitsService interface.
type service struct {
	// Note: These maps are treated as immutable after service initialization.
	// Thread-safety depends on this immutability assumption (no runtime modifications).
	plans          map[string]Plan
	counters       CounterRegistry
	planIDResolver PlanIDResolver
}

// NewLimitsService creates a new LimitsService with the given Source and CounterRegistry.
func NewLimitsService(ctx context.Context, src Source, counters CounterRegistry, planIDResolver PlanIDResolver) (LimitsService, error) {
	plans, err := src.Load(ctx)
	if err != nil {
		return nil, errors.Join(ErrFailedToLoadPlans, err)
	}

	if plans == nil {
		plans = make(map[string]Plan)
	}

	// Validate plan configurations
	if err := validatePlans(plans); err != nil {
		return nil, err
	}

	if counters == nil {
		counters = NewRegistry()
	}

	if planIDResolver == nil {
		planIDResolver = PlanIDContextResolver
	}

	return &service{
		plans:          plans,
		counters:       counters,
		planIDResolver: planIDResolver,
	}, nil
}

// CanCreate checks if a tenant can create a new resource instance.
func (s *service) CanCreate(ctx context.Context, tenantID uuid.UUID, res Resource) error {
	planID, err := s.planIDResolver(ctx, tenantID)
	if err != nil {
		return err
	}

	plan, exists := s.plans[planID]
	if !exists {
		return ErrPlanNotFound
	}

	limit, exists := plan.Limits[res]
	if !exists {
		return ErrInvalidResource
	}

	// -1 indicates unlimited usage
	if limit == Unlimited {
		return nil
	}

	counter, exists := s.counters[res]
	if !exists {
		return ErrNoCounterRegistered
	}

	current, err := counter(ctx, tenantID)
	if err != nil {
		return errors.Join(ErrFailedToCountResourceUsage, err)
	}

	if current >= limit {
		return ErrLimitExceeded
	}

	return nil
}

// GetUsage returns the current usage and limit for a resource in a tenant.
func (s *service) GetUsage(ctx context.Context, tenantID uuid.UUID, res Resource) (used, limit int64, err error) {
	planID, err := s.planIDResolver(ctx, tenantID)
	if err != nil {
		return 0, 0, err
	}

	plan, exists := s.plans[planID]
	if !exists {
		return 0, 0, ErrPlanNotFound
	}

	resourceLimit, exists := plan.Limits[res]
	if !exists {
		return 0, 0, ErrInvalidResource
	}

	counter, exists := s.counters[res]
	if !exists {
		return 0, 0, ErrNoCounterRegistered
	}

	current, err := counter(ctx, tenantID)
	if err != nil {
		return 0, 0, errors.Join(ErrFailedToCountResourceUsage, err)
	}

	return current, resourceLimit, nil
}

// GetUsageSafe is a convenience wrapper for UI dashboards. It returns zero values if usage cannot be obtained.
func (s *service) GetUsageSafe(ctx context.Context, tenantID uuid.UUID, res Resource) (used, limit int64) {
	used, limit, _ = s.GetUsage(ctx, tenantID, res)
	return used, limit
}

// HasFeature quickly tells whether a feature is available for the current plan.
func (s *service) HasFeature(ctx context.Context, tenantID uuid.UUID, feature Feature) bool {
	planID, err := s.planIDResolver(ctx, tenantID)
	if err != nil {
		return false
	}

	plan, exists := s.plans[planID]
	if !exists {
		return false
	}

	return slices.Contains(plan.Features, feature)
}

// CheckTrial determines if a tenant's trial period is active for a specific plan.
func (s *service) CheckTrial(ctx context.Context, tenantID uuid.UUID, startedAt time.Time) error {
	planID, err := s.planIDResolver(ctx, tenantID)
	if err != nil {
		return err
	}

	plan, exists := s.plans[planID]
	if !exists {
		return ErrPlanNotFound
	}

	if plan.TrialDays == 0 {
		return ErrTrialNotAvailable
	}

	if !plan.IsTrialActive(startedAt) {
		return ErrTrialExpired
	}

	return nil
}

// VerifyPlan checks if a plan ID is valid.
func (s *service) VerifyPlan(ctx context.Context, planID string) error {
	if _, exists := s.plans[planID]; !exists {
		return ErrPlanNotFound
	}
	return nil
}

// GetUsagePercentage returns usage as percentage (0-100, or -1 for unlimited).
func (s *service) GetUsagePercentage(ctx context.Context, tenantID uuid.UUID, res Resource) int {
	used, limit, err := s.GetUsage(ctx, tenantID, res)
	if err != nil {
		return 0
	}

	if limit == Unlimited {
		return -1
	}

	if limit == 0 {
		return 100
	}

	return min(int((used*100)/limit), 100)
}

// CanDowngrade checks if downgrade is possible given current usage.
func (s *service) CanDowngrade(ctx context.Context, tenantID uuid.UUID, targetPlanID string) error {
	// Verify target plan exists
	targetPlan, exists := s.plans[targetPlanID]
	if !exists {
		return ErrPlanNotFound
	}

	// Get current plan
	currentPlanID, err := s.planIDResolver(ctx, tenantID)
	if err != nil {
		return err
	}

	currentPlan, exists := s.plans[currentPlanID]
	if !exists {
		return ErrPlanNotFound
	}

	// Check each resource in the target plan
	for resource, targetLimit := range targetPlan.Limits {
		// If target is unlimited, no need to check
		if targetLimit == Unlimited {
			continue
		}

		// Get current limit
		currentLimit, hasResource := currentPlan.Limits[resource]

		// If current plan doesn't have this resource, skip
		if !hasResource {
			continue
		}

		// If current is unlimited but target is limited, need to check usage
		// If current limit is higher than target, need to check usage
		if currentLimit == Unlimited || currentLimit > targetLimit {
			// Check if current usage fits in target limit
			counter, exists := s.counters[resource]
			if !exists {
				// No counter means we can't verify, so we allow it
				continue
			}

			currentUsage, err := counter(ctx, tenantID)
			if err != nil {
				return errors.Join(ErrFailedToCountResourceUsage, err)
			}

			if currentUsage > targetLimit {
				return ErrDowngradeNotPossible
			}
		}
	}

	return nil
}

// GetAllUsage returns all resource usage for a tenant.
func (s *service) GetAllUsage(ctx context.Context, tenantID uuid.UUID) (map[Resource]UsageInfo, error) {
	planID, err := s.planIDResolver(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	plan, exists := s.plans[planID]
	if !exists {
		return nil, ErrPlanNotFound
	}

	result := make(map[Resource]UsageInfo, len(plan.Limits))

	for resource, limit := range plan.Limits {
		usage := UsageInfo{
			Current: 0,
			Limit:   limit,
		}

		// Try to get current usage if counter exists
		if counter, exists := s.counters[resource]; exists {
			if current, err := counter(ctx, tenantID); err == nil {
				usage.Current = current
			}
			// We don't fail on counter errors, just leave usage as 0
		}

		result[resource] = usage
	}

	return result, nil
}

// validatePlans checks plan configurations for validity.
func validatePlans(plans map[string]Plan) error {
	for planID, plan := range plans {
		if plan.TrialDays < 0 {
			return errors.Join(ErrInvalidPlanConfiguration,
				fmt.Errorf("plan %s has negative trial days: %d", planID, plan.TrialDays))
		}
	}
	return nil
}
