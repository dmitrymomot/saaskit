package subscription

import (
	"context"
	"maps"
	"slices"
	"sync"
)

type inMemSource struct {
	mu    sync.RWMutex
	plans map[string]Plan
}

// NewInMemSource returns an in-memory Source with a deep copy of the given plans.
// Panics if no plans are provided to ensure the service always has at least one valid plan.
// Deep copying prevents external modifications from affecting the source's state.
func NewInMemSource(plans ...Plan) PlansListSource {
	plansLen := len(plans)
	if plansLen < 1 {
		panic("at least one plan is required")
	}
	plansCopy := make(map[string]Plan, plansLen)
	for _, plan := range plans {
		plansCopy[plan.ID] = Plan{
			ID:          plan.ID,
			Name:        plan.Name,
			Description: plan.Description,
			Limits:      maps.Clone(plan.Limits),
			Features:    slices.Clone(plan.Features),
			Public:      plan.Public,
			TrialDays:   plan.TrialDays,
			Price:       plan.Price,
			Interval:    plan.Interval,
		}
	}

	return &inMemSource{
		plans: plansCopy,
	}
}

// Load returns a copy of all available plans from memory.
// Deep copying prevents callers from modifying the source's internal state.
func (s *inMemSource) Load(ctx context.Context) (map[string]Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plansCopy := make(map[string]Plan, len(s.plans))
	for id, plan := range s.plans {
		plansCopy[id] = Plan{
			ID:          plan.ID,
			Name:        plan.Name,
			Description: plan.Description,
			Limits:      maps.Clone(plan.Limits),
			Features:    slices.Clone(plan.Features),
			Public:      plan.Public,
			TrialDays:   plan.TrialDays,
			Price:       plan.Price,
			Interval:    plan.Interval,
		}
	}
	return plansCopy, nil
}
