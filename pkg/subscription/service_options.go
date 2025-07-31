package subscription

// ServiceOption configures a Service instance.
type ServiceOption func(*service)

// WithPlanIDResolver sets a custom plan ID resolver.
// Default resolver (PlanIDContextResolver) expects plan ID in context.
// Use this to implement database-backed plan resolution or other strategies.
func WithPlanIDResolver(resolver PlanIDResolver) ServiceOption {
	return func(s *service) {
		if resolver != nil {
			s.planIDResolver = resolver
		}
	}
}

// WithCounter registers a counter function for a specific resource.
// Counter functions must be fast as they're called on every creation attempt.
// Panics if a counter for the same resource has already been registered
// to prevent accidental overwrites and ensure explicit configuration.
func WithCounter(resource Resource, fn ResourceCounterFunc) ServiceOption {
	return func(s *service) {
		if fn == nil {
			return
		}
		if s.counters == nil {
			s.counters = make(map[Resource]ResourceCounterFunc)
		}
		if _, exists := s.counters[resource]; exists {
			panic("subscription: counter for resource " + string(resource) + " already registered")
		}
		s.counters[resource] = fn
	}
}
