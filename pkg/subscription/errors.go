package subscription

import "errors"

var (
	ErrPlanNotFound             = errors.New("subscription plan not found")
	ErrPlanIDNotFound           = errors.New("subscription plan ID not found")
	ErrPlanIDNotInContext       = errors.New("subscription plan ID not found in context")
	ErrInvalidPlanConfiguration = errors.New("invalid subscription plan configuration")

	ErrLimitExceeded        = errors.New("subscription limit exceeded")
	ErrInvalidResource      = errors.New("invalid subscription resource")
	ErrNoCounterRegistered  = errors.New("no usage counter registered for resource")
	ErrDowngradeNotPossible = errors.New("subscription downgrade not possible")

	ErrTrialExpired      = errors.New("subscription trial has expired")
	ErrTrialNotAvailable = errors.New("subscription trial not available")

	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
	ErrInvalidSubscriptionState  = errors.New("invalid subscription state")
	ErrProviderError             = errors.New("subscription provider error")

	ErrFailedToLoadPlans          = errors.New("failed to load subscription plans")
	ErrFailedToCountResourceUsage = errors.New("failed to count resource usage")
)
