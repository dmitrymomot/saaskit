# broadcast

A type-safe, generic broadcasting system for pub/sub messaging with storage persistence and Server-Sent Events (SSE) integration.

## Features

- Type-safe generic messaging with compile-time guarantees
- Hub-based architecture for managing channels and subscribers
- Message persistence with pluggable Storage interface
- Automatic slow consumer detection and cleanup
- Message replay for new subscribers
- Server-Sent Events (SSE) integration
- Graceful shutdown with configurable timeouts
- Thread-safe concurrent operations
- Metrics callbacks for monitoring

## Installation

```bash
go get github.com/dmitrymomot/saaskit/pkg/broadcast
```

## Usage

### Basic Pub/Sub

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/dmitrymomot/saaskit/pkg/broadcast"
)

func main() {
    // Create a hub for string messages
    hub := broadcast.NewHub[string](broadcast.HubConfig{
        DefaultBufferSize: 100,
    })
    defer hub.Close()

    ctx := context.Background()

    // Subscribe to a channel
    sub, err := hub.Subscribe(ctx, "news")
    if err != nil {
        log.Fatal(err)
    }
    defer sub.Close()

    // Start receiving messages in a goroutine
    go func() {
        for msg := range sub.Messages() {
            fmt.Printf("Received: %s at %v\n", msg.Payload, msg.Timestamp)
        }
    }()

    // Publish messages
    err = hub.Publish(ctx, "news", "Breaking news!")
    if err != nil {
        log.Fatal(err)
    }

    time.Sleep(100 * time.Millisecond)
}
```

### With Custom Message Types

```go
type ChatMessage struct {
    User    string `json:"user"`
    Content string `json:"content"`
    Room    string `json:"room"`
}

hub := broadcast.NewHub[ChatMessage](broadcast.HubConfig{
    DefaultBufferSize: 50,
})

msg := ChatMessage{
    User:    "alice",
    Content: "Hello everyone!",
    Room:    "general",
}

err := hub.Publish(ctx, "chat", msg,
    broadcast.WithMetadata(broadcast.Metadata{"priority": "high"}),
)
```

### With Storage Persistence

```go
// Implement the Storage interface
type MyStorage struct {
    // Your storage implementation
}

func (s *MyStorage) Store(ctx context.Context, message broadcast.Message[any]) error {
    // Store message logic
    return nil
}

func (s *MyStorage) Load(ctx context.Context, channel string, opts broadcast.LoadOptions) ([]broadcast.Message[any], error) {
    // Load messages logic
    return nil, nil
}

func (s *MyStorage) Delete(ctx context.Context, before time.Time) error {
    // Delete old messages logic
    return nil
}

func (s *MyStorage) Channels(ctx context.Context) ([]string, error) {
    // List channels logic
    return nil, nil
}

// Use with hub
hub := broadcast.NewHub[string](broadcast.HubConfig{
    Storage:             &MyStorage{},
    DefaultBufferSize:   100,
    CleanupInterval:     5 * time.Minute,
    SlowConsumerTimeout: 3 * time.Second,
})
```

### Message Replay

```go
// Subscribe with replay of last 10 messages
sub, err := hub.Subscribe(ctx, "events",
    broadcast.WithReplay(10),
    broadcast.WithBufferSize(200),
)
```

### SSE Integration

```go
func ChatHandler(hub broadcast.Hub[ChatMessage]) handler.HandlerFunc[handler.StreamContext, JoinRequest] {
    return func(ctx handler.StreamContext, req JoinRequest) handler.Response {
        sub, err := hub.Subscribe(ctx.Request().Context(), "chat")
        if err != nil {
            return handler.Error(err)
        }
        defer sub.Close()

        for msg := range sub.Messages() {
            if err := ctx.SendJSON(msg.Payload); err != nil {
                return handler.Error(err)
            }
        }
        return handler.OK()
    }
}
```

## Common Operations

### Monitoring Subscribers

```go
// Get active channels
channels := hub.Channels()
fmt.Printf("Active channels: %v\n", channels)

// Get subscriber count for a channel
count := hub.SubscriberCount("news")
fmt.Printf("Subscribers: %d\n", count)
```

### Error Handling with Callbacks

```go
sub, err := hub.Subscribe(ctx, "events",
    broadcast.WithErrorCallback(func(err error) {
        log.Printf("Subscription error: %v", err)
    }),
    broadcast.WithSlowConsumerCallback(func() {
        log.Println("Slow consumer detected")
    }),
)
```

## Error Handling

The package exports several specific error types:

```go
import "errors"

err := hub.Publish(ctx, "channel", "message")
if err != nil {
    switch {
    case errors.Is(err, broadcast.ErrHubClosed{}):
        // Hub is closed
    case errors.As(err, &broadcast.ErrStorageFailure{}):
        // Storage operation failed
    case errors.Is(err, broadcast.ErrShutdownTimeout{}):
        // Shutdown timeout exceeded
    default:
        // Other error
    }
}
```

## Configuration

### HubConfig Options

```go
config := broadcast.HubConfig{
    Storage:             nil,                    // Optional storage backend
    DefaultBufferSize:   100,                   // Default subscriber buffer size
    CleanupInterval:     5 * time.Minute,       // Channel cleanup frequency
    SlowConsumerTimeout: 5 * time.Second,       // Timeout for slow consumers
    ShutdownTimeout:     30 * time.Second,      // Graceful shutdown timeout
    ReplayTimeout:       10 * time.Second,      // Message replay timeout
    MetricsCallback: func(channel string, subscribers int) {
        // Handle metrics updates
    },
}

hub := broadcast.NewHub[MyMessageType](config)
```

### Subscribe Options

```go
sub, err := hub.Subscribe(ctx, "channel",
    broadcast.WithBufferSize(200),              // Custom buffer size
    broadcast.WithReplay(50),                   // Replay last 50 messages
    broadcast.WithErrorCallback(errorHandler),  // Error callback
    broadcast.WithSlowConsumerCallback(slowHandler), // Slow consumer callback
)
```

### Publish Options

```go
err := hub.Publish(ctx, "channel", payload,
    broadcast.WithPersistence(),                // Enable storage persistence
    broadcast.WithMetadata(metadata),           // Add message metadata
    broadcast.WithTimeout(5 * time.Second),     // Publish timeout
)
```

## Storage Interface

Implement the `Storage` interface for message persistence:

```go
type Storage interface {
    Store(ctx context.Context, message Message[any]) error
    Load(ctx context.Context, channel string, opts LoadOptions) ([]Message[any], error)
    Delete(ctx context.Context, before time.Time) error
    Channels(ctx context.Context) ([]string, error)
}

type LoadOptions struct {
    Limit  int        // Maximum messages to return
    After  *time.Time // Only messages after this time
    Before *time.Time // Only messages before this time
    LastID string     // For cursor-based pagination
}
```

### Storage Implementation Example

```go
type RedisStorage struct {
    client *redis.Client
}

func (r *RedisStorage) Store(ctx context.Context, message broadcast.Message[any]) error {
    data, err := json.Marshal(message)
    if err != nil {
        return err
    }
    
    key := fmt.Sprintf("broadcast:%s", message.Channel)
    return r.client.LPush(ctx, key, data).Err()
}

func (r *RedisStorage) Load(ctx context.Context, channel string, opts broadcast.LoadOptions) ([]broadcast.Message[any], error) {
    key := fmt.Sprintf("broadcast:%s", channel)
    
    limit := opts.Limit
    if limit <= 0 {
        limit = 100
    }
    
    results, err := r.client.LRange(ctx, key, 0, int64(limit-1)).Result()
    if err != nil {
        return nil, err
    }
    
    messages := make([]broadcast.Message[any], 0, len(results))
    for _, result := range results {
        var msg broadcast.Message[any]
        if err := json.Unmarshal([]byte(result), &msg); err != nil {
            continue
        }
        messages = append(messages, msg)
    }
    
    return messages, nil
}
```

## API Documentation

For detailed API documentation, run:

```bash
go doc -all ./pkg/broadcast
```

Or visit [pkg.go.dev](https://pkg.go.dev/github.com/dmitrymomot/saaskit/pkg/broadcast) for online documentation.

## Performance Considerations

- **Buffer Sizes**: Choose appropriate buffer sizes based on message volume. Larger buffers prevent blocking but use more memory.
- **Slow Consumers**: Configure `SlowConsumerTimeout` to automatically remove subscribers that can't keep up.
- **Cleanup**: Enable `CleanupInterval` to periodically remove empty channels and prevent memory leaks.
- **Storage**: Use efficient storage backends (Redis, PostgreSQL) for high-throughput scenarios.
- **Goroutine Management**: The hub manages goroutines automatically but ensure proper cleanup with `defer Close()`.

## Thread Safety

- All Hub operations are thread-safe and can be called concurrently
- Subscribers should only be used by a single goroutine for receiving messages
- Storage implementations must be thread-safe if used concurrently

## Notes

- Messages are delivered in order per subscriber
- Slow consumers are automatically detected and disconnected to prevent memory leaks
- Storage persistence is optional and can be enabled per message with `WithPersistence()`
- The package integrates seamlessly with the SaasKit handler package for SSE responses
- Always call `defer hub.Close()` and `defer sub.Close()` to ensure proper cleanup

## Contributing Guidelines

1. **Testing**: Maintain >80% test coverage for all new functionality
2. **Benchmarks**: Add benchmarks for performance-critical paths
3. **Documentation**: Update this README and add godoc comments for new public APIs
4. **Type Safety**: Preserve generic type safety in all operations
5. **Error Handling**: Use typed errors and proper error wrapping
6. **Thread Safety**: Ensure all public APIs remain thread-safe