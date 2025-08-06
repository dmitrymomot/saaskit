// Package broadcast provides type-safe message broadcasting with subscriber management.
//
// This package implements a generic publish-subscribe pattern that enables one-to-many
// communication with compile-time type safety through Go generics. It is designed for
// high-performance, non-blocking message distribution where slow consumers are
// automatically dropped to prevent system-wide blocking.
//
// # Architecture
//
// The package follows an interface-based design with pluggable implementations:
//
//   - Broadcaster[T]: Core interface for message distribution
//   - Subscriber[T]: Interface for message consumption
//   - Message[T]: Type-safe wrapper for broadcast data
//   - MemoryBroadcaster[T]: In-memory implementation with buffered channels
//
// The architecture supports future adapters (Redis, NATS, etc.) while maintaining
// the same API contract. All operations are thread-safe and optimized for minimal
// lock contention using RWMutex.
//
// # Usage
//
// The package provides a simple, type-safe API for broadcasting messages:
//
//	import "github.com/dmitrymomot/saaskit/pkg/broadcast"
//
//	// Create a broadcaster with buffer size 10
//	broadcaster := broadcast.NewMemoryBroadcaster[string](10)
//	defer broadcaster.Close()
//
//	// Subscribe to messages
//	ctx := context.Background()
//	subscriber := broadcaster.Subscribe(ctx)
//	defer subscriber.Close()
//
//	// Start receiving messages
//	go func() {
//		for msg := range subscriber.Receive(ctx) {
//			fmt.Println("Received:", msg.Data)
//		}
//	}()
//
//	// Broadcast messages to all subscribers
//	broadcaster.Broadcast(ctx, broadcast.Message[string]{Data: "hello"})
//	broadcaster.Broadcast(ctx, broadcast.Message[string]{Data: "world"})
//
// # Type Safety
//
// The package leverages Go generics to ensure type safety at compile time:
//
//	type Notification struct {
//		Type    string
//		UserID  string
//		Message string
//	}
//
//	// Type-safe broadcaster for Notification
//	broadcaster := broadcast.NewMemoryBroadcaster[Notification](100)
//
//	// Compile-time type checking
//	broadcaster.Broadcast(ctx, broadcast.Message[Notification]{
//		Data: Notification{
//			Type:    "alert",
//			UserID:  "user123",
//			Message: "System maintenance scheduled",
//		},
//	})
//
// # Non-Blocking Behavior
//
// The broadcaster never blocks on slow consumers. When a subscriber's buffer is full,
// messages are dropped for that subscriber and it is marked for removal:
//
//	// Subscriber with small buffer
//	sub := broadcaster.Subscribe(ctx) // buffer size from broadcaster config
//
//	// If subscriber doesn't consume fast enough, messages are dropped
//	// and subscriber is automatically removed to prevent memory leaks
//
// # Context-Aware Lifecycle
//
// Subscribers are automatically cleaned up when their context is cancelled:
//
//	// Subscriber with timeout
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	sub := broadcaster.Subscribe(ctx)
//	// Automatically removed after 5 seconds, no manual cleanup needed
//
// # Buffer Size Considerations
//
// Choose buffer sizes based on your use case:
//
//   - Small (1-10): Real-time updates, quick detection of slow consumers
//   - Medium (10-100): Balanced for most applications
//   - Large (100-1000): High throughput, tolerate burst traffic
//   - Very Large (1000+): When message loss is critical
//
// # Error Handling
//
// The package defines two error conditions:
//
//	var (
//		ErrBroadcasterClosed = errors.New("broadcaster: closed")
//		ErrSubscriberClosed  = errors.New("subscriber: closed")
//	)
//
// Operations on closed resources are safe and idempotent:
//
//	broadcaster.Close()
//	broadcaster.Close() // Safe, no error
//
//	// Broadcast after close is a no-op
//	err := broadcaster.Broadcast(ctx, msg) // Returns nil, no-op
//
//	// Subscribe after close returns closed subscriber
//	sub := broadcaster.Subscribe(ctx) // Returns already-closed subscriber
//
// # Performance Characteristics
//
// The memory implementation is optimized for high throughput:
//
//   - Broadcast: O(n) where n is subscriber count, non-blocking
//   - Subscribe: O(1) with minimal allocations
//   - Memory: O(n*m) where n is subscribers, m is buffer size
//   - Concurrency: RWMutex minimizes contention for reads
//
// Benchmarks show ~82ns per broadcast operation with zero allocations,
// and ~1.5μs per subscribe operation with 4 allocations.
//
// # Design Patterns
//
// The package implements several design patterns:
//
//   - Observer Pattern: Core publish-subscribe mechanism
//   - Adapter Pattern: Interface-based design for multiple implementations
//   - Resource Management: Automatic cleanup with contexts and WaitGroups
//   - Non-Blocking I/O: Select with default case for dropping messages
//
// # Thread Safety
//
// All public methods are safe for concurrent use. The implementation uses:
//
//   - RWMutex for subscriber map access (read-heavy optimization)
//   - Channel operations for message passing
//   - WaitGroup for graceful shutdown coordination
//   - Atomic operations through mutex-protected state
//
// # Examples
//
// See the example_test.go file for comprehensive usage patterns including:
//
//   - Multiple subscribers
//   - Context cancellation
//   - Error handling
//   - Slow consumer handling
//   - Type-safe custom types
package broadcast
