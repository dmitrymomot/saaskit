package audit

// WithResource sets the resource type and ID
func WithResource(resource, id string) EventOption {
	return func(e *Event) {
		e.Resource = resource
		e.ResourceID = id
	}
}

// WithMetadata adds metadata to the event
func WithMetadata(key string, value any) EventOption {
	return func(e *Event) {
		if e.Metadata == nil {
			e.Metadata = make(map[string]any)
		}
		e.Metadata[key] = value
	}
}

// WithResult sets the event result
func WithResult(result Result) EventOption {
	return func(e *Event) {
		e.Result = result
	}
}
