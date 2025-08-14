package subscription

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type planIDCtxKey struct{}

func SetPlanIDToContext(ctx context.Context, planID string) context.Context {
	return context.WithValue(ctx, planIDCtxKey{}, planID)
}

func GetPlanIDFromContext(ctx context.Context) (string, bool) {
	planID, ok := ctx.Value(planIDCtxKey{}).(string)
	return planID, ok
}

// PlanIDContextResolver is the default resolver that retrieves plan ID from context.
// This resolver allows dynamic plan resolution without database lookups, useful for
// multi-tenant applications where plan ID is determined during request processing.
func PlanIDContextResolver(ctx context.Context, _ uuid.UUID) (string, error) {
	planID, ok := GetPlanIDFromContext(ctx)
	if !ok {
		return "", errors.Join(ErrPlanIDNotFound, ErrPlanIDNotInContext)
	}
	return planID, nil
}
