package limits

import "errors"

// Domain errors for limits operations
var (
	// Plan errors
	ErrPlanNotFound             = errors.New("limits.errors.plan_not_found")
	ErrPlanIDNotFound           = errors.New("limits.errors.plan_id_not_found")
	ErrPlanIDNotInContext       = errors.New("limits.errors.plan_id_not_in_context")
	ErrInvalidPlanConfiguration = errors.New("limits.errors.invalid_plan_configuration")

	// Resource limit errors
	ErrLimitExceeded        = errors.New("limits.errors.limit_exceeded")
	ErrInvalidResource      = errors.New("limits.errors.invalid_resource")
	ErrNoCounterRegistered  = errors.New("limits.errors.no_counter_registered")
	ErrDowngradeNotPossible = errors.New("limits.errors.downgrade_not_possible")

	// Trial errors
	ErrTrialExpired      = errors.New("limits.errors.trial_expired")
	ErrTrialNotAvailable = errors.New("limits.errors.trial_not_available")

	// System errors
	ErrFailedToLoadPlans          = errors.New("limits.errors.failed_to_load_plans")
	ErrFailedToCountResourceUsage = errors.New("limits.errors.failed_to_count_resource_usage")
)
