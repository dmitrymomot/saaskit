package audit

type Option func(*Logger)

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
