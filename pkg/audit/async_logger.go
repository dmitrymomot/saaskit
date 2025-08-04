package audit

import "context"

// NewAsyncLogger creates a logger optimized for high-throughput scenarios.
// Returns both the logger and a cleanup function that should be called during shutdown.
// BufferSize determines memory usage vs throughput tradeoff (typical: 1000-10000).
func NewAsyncLogger(bw batchWriter, bufferSize int, opts ...Option) (*Logger, func(context.Context) error) {
	asyncWriter, closeFunc := NewAsyncWriter(bw, AsyncOptions{
		BufferSize: bufferSize,
	})

	logger := NewLogger(asyncWriter, opts...)
	return logger, closeFunc
}
