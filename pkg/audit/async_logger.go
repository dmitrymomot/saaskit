package audit

import "context"

// NewAsyncLogger creates a logger with async batch writer
// Requires a BatchWriter implementation for efficient batch operations
func NewAsyncLogger(bw batchWriter, bufferSize int, opts ...Option) (*Logger, func(context.Context) error) {
	asyncWriter, closeFunc := NewAsyncWriter(bw, AsyncOptions{
		BufferSize: bufferSize,
	})

	logger := NewLogger(asyncWriter, opts...)
	return logger, closeFunc
}
