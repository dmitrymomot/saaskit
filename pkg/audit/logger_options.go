package audit

type Option func(*logger)

func WithTenantIDExtractor(fn contextExtractor) Option {
	return func(l *logger) {
		l.tenantIDExtractor = fn
	}
}

func WithUserIDExtractor(fn contextExtractor) Option {
	return func(l *logger) {
		l.userIDExtractor = fn
	}
}

func WithSessionIDExtractor(fn contextExtractor) Option {
	return func(l *logger) {
		l.sessionIDExtractor = fn
	}
}

func WithRequestIDExtractor(fn contextExtractor) Option {
	return func(l *logger) {
		l.requestIDExtractor = fn
	}
}

func WithIPExtractor(fn contextExtractor) Option {
	return func(l *logger) {
		l.ipExtractor = fn
	}
}

func WithUserAgentExtractor(fn contextExtractor) Option {
	return func(l *logger) {
		l.userAgentExtractor = fn
	}
}

func WithAsync(bufferSize int) Option {
	return func(l *logger) {
		l.asyncBufferSize = bufferSize
	}
}
