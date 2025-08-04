package audit

import "time"

type Option func(*logger)

// AsyncOptions configures async storage behavior
type AsyncOptions struct {
	BatchSize      int           // Number of events to batch before flushing
	BatchTimeout   time.Duration // Duration to wait before flushing a partial batch
	StorageTimeout time.Duration // Timeout for storing events to the underlying storage
}

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

func WithAsyncOptions(opts AsyncOptions) Option {
	return func(l *logger) {
		l.asyncOptions = opts
	}
}
