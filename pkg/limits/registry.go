package limits

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CounterFunc returns the current usage for a tenant resource.
// Should be fast: cache or aggregate at repository level.
type CounterFunc func(ctx context.Context, tenantID uuid.UUID) (int64, error)

// CounterRegistry maps a Resource to its CounterFunc.
// Not thread-safe: register all counters at startup only.
type CounterRegistry map[Resource]CounterFunc

// NewRegistry returns a new, empty CounterRegistry.
func NewRegistry() CounterRegistry {
	return make(CounterRegistry)
}

// Register sets or replaces the CounterFunc for the given resource. Panics if fn is nil.
func (r CounterRegistry) Register(res Resource, fn CounterFunc) {
	if fn == nil {
		panic(fmt.Sprintf("limits: CounterFunc for resource %q cannot be nil", res))
	}
	r[res] = fn
}
