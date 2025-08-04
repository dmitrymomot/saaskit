package audit

// Option configures Logger behavior during initialization
type Option func(*Logger)

// Context extractors enable automatic population of audit events from request context.
// These functions attempt to extract values and return (value, found) to indicate success.
// If extraction fails, the corresponding event field will remain empty.

func WithTenantIDExtractor(fn contextExtractor) Option {
	return func(l *Logger) {
		l.tenantIDExtractor = fn
	}
}

func WithUserIDExtractor(fn contextExtractor) Option {
	return func(l *Logger) {
		l.userIDExtractor = fn
	}
}

func WithSessionIDExtractor(fn contextExtractor) Option {
	return func(l *Logger) {
		l.sessionIDExtractor = fn
	}
}

func WithRequestIDExtractor(fn contextExtractor) Option {
	return func(l *Logger) {
		l.requestIDExtractor = fn
	}
}

func WithIPExtractor(fn contextExtractor) Option {
	return func(l *Logger) {
		l.ipExtractor = fn
	}
}

func WithUserAgentExtractor(fn contextExtractor) Option {
	return func(l *Logger) {
		l.userAgentExtractor = fn
	}
}
