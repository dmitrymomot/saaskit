package limits

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Plan ID context management
type planIDCtxKey struct{}

// SetPlanIDToContext stores the plan ID in the context for downstream access.
func SetPlanIDToContext(ctx context.Context, planID string) context.Context {
	return context.WithValue(ctx, planIDCtxKey{}, planID)
}

// GetPlanIDFromContext retrieves the plan ID from the context, if present.
func GetPlanIDFromContext(ctx context.Context) (string, bool) {
	planID, ok := ctx.Value(planIDCtxKey{}).(string)
	return planID, ok
}

// PlanIDContextResolver is the default resolver: gets plan ID from context or returns error.
func PlanIDContextResolver(ctx context.Context, _ uuid.UUID) (string, error) {
	planID, ok := GetPlanIDFromContext(ctx)
	if !ok {
		return "", errors.Join(ErrPlanIDNotFound, ErrPlanIDNotInContext)
	}
	return planID, nil
}
