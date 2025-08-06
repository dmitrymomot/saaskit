# Broadcast Package

Type-safe, generic pub/sub broadcasting for Go applications with pluggable adapters.

## Features

- **Type Safety**: Generic interfaces prevent runtime type errors
- **Zero Config**: Works immediately with sensible defaults
- **Memory Safe**: Automatic cleanup of abandoned subscribers
- **Non-Blocking**: Drops slow consumers to prevent system blocking
- **Context Aware**: Respects context cancellation for graceful shutdown
- **Thread Safe**: Concurrent operations protected by RWMutex
- **Extensible**: Adapter pattern for future Redis/NATS implementations

## Installation

```bash
go get github.com/dmitrymomot/saaskit/pkg/broadcast
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/dmitrymomot/saaskit/pkg/broadcast"
)

func main() {
    // Create broadcaster with buffer size 10
    b := broadcast.NewMemoryBroadcaster[string](10)
    defer b.Close()

    ctx := context.Background()

    // Subscribe to messages
    sub := b.Subscribe(ctx)

    // Receive messages in goroutine
    go func() {
        for msg := range sub.Receive(ctx) {
            fmt.Println("Received:", msg.Data)
        }
    }()

    // Broadcast messages
    b.Broadcast(ctx, broadcast.Message[string]{Data: "Hello, World!"})
}
```

## Usage Patterns

### Typed Messages

```go
type Notification struct {
    Type    string
    UserID  string
    Message string
}

b := broadcast.NewMemoryBroadcaster[Notification](100)

// Type-safe broadcasting
b.Broadcast(ctx, broadcast.Message[Notification]{
    Data: Notification{
        Type:    "payment",
        UserID:  "user-123",
        Message: "Payment received",
    },
})
```

### Multiple Subscribers

```go
b := broadcast.NewMemoryBroadcaster[Event](100)

// Each subscriber gets their own copy
alice := b.Subscribe(ctx)
bob := b.Subscribe(ctx)

// Both receive the same message
b.Broadcast(ctx, broadcast.Message[Event]{Data: event})
```

### Context Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

sub := b.Subscribe(ctx)

// Subscription automatically cleaned up when context cancels
```

## Buffer Size Guide

Choosing the right buffer size depends on your use case:

- **Small (1-10)**: Low memory usage, quick dropping of slow consumers. Good for real-time updates where latest data matters most.
- **Medium (10-100)**: Balanced approach suitable for most applications. Handles brief processing delays without dropping messages.
- **Large (100-1000)**: High throughput scenarios with bursty traffic. More memory per subscriber but better tolerance for processing delays.
- **Very Large (1000+)**: Only for specific cases where message loss is critical and memory usage is not a concern.

Consider your:

- Message frequency
- Processing time per message
- Tolerance for message loss
- Available memory

## Performance

- **Broadcast**: ~82ns per operation, 0 allocations
- **Subscribe**: ~1.5μs per operation, 4 allocations
- **Memory**: Minimal overhead, messages not copied per subscriber

## Design Decisions

### Why Generics?

Type safety at compile time prevents runtime panics and makes refactoring safer.

### Why Drop Slow Consumers?

Prevents one slow subscriber from blocking the entire system. Better to drop messages than deadlock.

### Why Channel-Based?

Go-idiomatic pattern that works well with select statements and context cancellation.

## Future Adapters

The package is designed with an adapter pattern for future scaling:

```go
// Future Redis adapter (not yet implemented)
b := broadcast.NewRedisBroadcaster[Message](redisClient, config)

// Same interface, distributed broadcasting
sub := b.Subscribe(ctx)
b.Broadcast(ctx, msg)
```

## Thread Safety

All operations are thread-safe. The broadcaster uses RWMutex to protect concurrent access to the subscriber map.

## Troubleshooting

### Messages Not Being Received

**Symptom**: Subscriber's Receive() channel returns no messages.

**Common Causes**:

- Context was cancelled before messages were sent
- Subscriber was closed before receiving
- Broadcaster was closed

**Solution**:

```go
// Ensure context is not cancelled
ctx := context.Background() // or context.WithTimeout with sufficient time

// Check for closed channel
for msg := range sub.Receive(ctx) {
    // Process message
}
// Channel closed when loop exits
```

### Memory Leaks

**Symptom**: Goroutines or memory usage grows over time.

**Common Causes**:

- Not calling `broadcaster.Close()` when done
- Creating subscribers with long-lived contexts that never cancel

**Solution**:

```go
// Always close broadcaster when done
b := broadcast.NewMemoryBroadcaster[T](100)
defer b.Close() // Ensures cleanup

// Use context with timeout for subscribers
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
sub := b.Subscribe(ctx)
```

### Messages Being Dropped

**Symptom**: Some subscribers miss messages.

**Common Causes**:

- Buffer size too small for message rate
- Slow message processing blocking the receive channel

**Solution**:

```go
// Increase buffer size
b := broadcast.NewMemoryBroadcaster[T](1000) // Larger buffer

// Process messages quickly or offload to goroutine
for msg := range sub.Receive(ctx) {
    go processMessage(msg) // Non-blocking processing
}
```

### Performance Issues

**Symptom**: High CPU or slow broadcast operations.

**Common Causes**:

- Too many subscribers (1000+)
- Very large messages
- Inefficient message processing

**Solution**:

- Consider batching messages
- Reduce subscriber count or use multiple broadcasters
- Profile to identify bottlenecks
- Consider future Redis adapter for distributed load

## Limitations

- **In-Memory Only**: Current implementation doesn't persist messages
- **No Guaranteed Delivery**: Messages can be dropped if buffers are full
- **No Message History**: Late subscribers don't receive previous messages
- **Single Process**: Doesn't work across multiple application instances (use Redis adapter when available)

## Contributing

This package follows MLP (Minimum Lovable Product) principles. Features are intentionally limited to keep the implementation simple and maintainable.

Before adding features, consider:

- Does this solve a real problem for 80% of users?
- Can it be implemented without breaking existing code?
- Does it maintain the simplicity of the current API?

## License

Part of the SaasKit framework. See repository LICENSE for details.
