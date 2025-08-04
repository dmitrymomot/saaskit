package audit

func WithResource(resource, id string) EventOption {
	return func(e *Event) {
		e.Resource = resource
		e.ResourceID = id
	}
}

// WithMetadata initializes metadata map if needed before adding key-value pair.
// Safe to call multiple times to build up metadata incrementally.
func WithMetadata(key string, value any) EventOption {
	return func(e *Event) {
		if e.Metadata == nil {
			e.Metadata = make(map[string]any)
		}
		e.Metadata[key] = value
	}
}

// WithResult overrides the default result set by Log/LogError methods.
// Typically used to mark a successful action as a failure based on business logic.
func WithResult(result Result) EventOption {
	return func(e *Event) {
		e.Result = result
	}
}
