package limits

import (
	"context"
	"maps"
	"slices"
	"sync"
)

// inMemSource implements the Source interface using an in-memory plan map.
type inMemSource struct {
	mu    sync.RWMutex
	plans map[string]Plan
}

// NewInMemSource returns an in-memory Source with a deep copy of the given plans.
func NewInMemSource(plans map[string]Plan) Source {
	plansCopy := make(map[string]Plan, len(plans))
	for id, plan := range plans {
		plansCopy[id] = Plan{
			ID:          plan.ID,
			Name:        plan.Name,
			Description: plan.Description,
			Limits:      maps.Clone(plan.Limits),
			Features:    slices.Clone(plan.Features),
			Public:      plan.Public,
			TrialDays:   plan.TrialDays,
		}
	}

	return &inMemSource{
		plans: plansCopy,
	}
}

// Load returns a copy of all available plans from memory.
// The returned map is not protected by the mutex after return.
func (s *inMemSource) Load(ctx context.Context) (map[string]Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plansCopy := make(map[string]Plan, len(s.plans))
	for id, plan := range s.plans {
		featuresCopy := slices.Clone(plan.Features)

		plansCopy[id] = Plan{
			ID:          plan.ID,
			Name:        plan.Name,
			Description: plan.Description,
			Limits:      maps.Clone(plan.Limits),
			Features:    featuresCopy,
			Public:      plan.Public,
			TrialDays:   plan.TrialDays,
		}
	}
	return plansCopy, nil
}
