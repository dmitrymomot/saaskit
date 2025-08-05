package broadcast

import (
	"context"
	"time"
)

// Message represents a broadcast message with generic payload
type Message[T any] struct {
	ID        string    `json:"id"`
	Channel   string    `json:"channel"`
	Payload   T         `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
	Metadata  Metadata  `json:"metadata,omitempty"`
}

// Metadata holds optional message metadata
type Metadata map[string]any

// Hub manages message broadcasting to subscribers
type Hub[T any] interface {
	// Subscribe creates a new subscription to a channel
	Subscribe(ctx context.Context, channel string, opts ...SubscribeOption) (Subscriber[T], error)

	// Publish sends a message to all subscribers of a channel
	Publish(ctx context.Context, channel string, payload T, opts ...PublishOption) error

	// PublishMessage sends a pre-built message
	PublishMessage(ctx context.Context, message Message[T]) error

	// Channels returns a list of active channels
	Channels() []string

	// SubscriberCount returns the number of subscribers for a channel
	SubscriberCount(channel string) int

	// Close gracefully shuts down the hub
	Close() error
}

// Subscriber represents a subscription to receive messages
type Subscriber[T any] interface {
	// Messages returns a channel to receive messages
	Messages() <-chan Message[T]

	// Channel returns the subscribed channel name
	Channel() string

	// ID returns the unique subscriber ID
	ID() string

	// Close unsubscribes and cleans up resources
	Close() error
}

// Storage interface for message persistence
type Storage interface {
	// Store saves a message
	Store(ctx context.Context, message Message[any]) error

	// Load retrieves messages for a channel
	Load(ctx context.Context, channel string, opts LoadOptions) ([]Message[any], error)

	// Delete removes messages older than the given time
	Delete(ctx context.Context, before time.Time) error

	// Channels returns all known channels
	Channels(ctx context.Context) ([]string, error)
}

// LoadOptions configures message loading
type LoadOptions struct {
	Limit  int        // Maximum messages to return
	After  *time.Time // Only messages after this time
	Before *time.Time // Only messages before this time
	LastID string     // For cursor-based pagination
}

// SubscribeOption configures subscription behavior
type SubscribeOption func(*subscribeConfig)

type subscribeConfig struct {
	bufferSize     int
	replay         bool
	replayLimit    int
	errorCallback  func(error)
	onSlowConsumer func()
}

// WithBufferSize sets the message buffer size for a subscriber
func WithBufferSize(size int) SubscribeOption {
	return func(c *subscribeConfig) {
		c.bufferSize = size
	}
}

// WithReplay enables replay of recent messages
func WithReplay(limit int) SubscribeOption {
	return func(c *subscribeConfig) {
		c.replay = true
		c.replayLimit = limit
	}
}

// WithErrorCallback sets a callback for subscription errors
func WithErrorCallback(fn func(error)) SubscribeOption {
	return func(c *subscribeConfig) {
		c.errorCallback = fn
	}
}

// WithSlowConsumerCallback sets a callback when consumer is slow
func WithSlowConsumerCallback(fn func()) SubscribeOption {
	return func(c *subscribeConfig) {
		c.onSlowConsumer = fn
	}
}

// PublishOption configures publish behavior
type PublishOption func(*publishConfig)

type publishConfig struct {
	persist  bool
	metadata Metadata
	timeout  time.Duration
}

// WithPersistence enables message persistence
func WithPersistence() PublishOption {
	return func(c *publishConfig) {
		c.persist = true
	}
}

// WithMetadata adds metadata to the message
func WithMetadata(metadata Metadata) PublishOption {
	return func(c *publishConfig) {
		c.metadata = metadata
	}
}

// WithTimeout sets publish timeout
func WithTimeout(timeout time.Duration) PublishOption {
	return func(c *publishConfig) {
		c.timeout = timeout
	}
}

// HubConfig configures hub behavior
type HubConfig struct {
	Storage             Storage
	DefaultBufferSize   int
	CleanupInterval     time.Duration
	SlowConsumerTimeout time.Duration
	ShutdownTimeout     time.Duration // Timeout for graceful shutdown
	ReplayTimeout       time.Duration // Timeout for replaying messages
	MetricsCallback     func(channel string, subscribers int)
}

// NewHub creates a new broadcasting hub
func NewHub[T any](config HubConfig) Hub[T] {
	return newHub[T](config)
}
