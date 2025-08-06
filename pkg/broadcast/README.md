# broadcast

Type-safe message broadcasting with subscriber management for one-to-many communication patterns.

## Features

- **Type-safe generics** - Compile-time type safety for all messages
- **Non-blocking broadcasts** - Never blocks on slow consumers, drops messages instead
- **Context-aware lifecycle** - Automatic cleanup when context cancels
- **Thread-safe operations** - All methods safe for concurrent use

## Installation

```go
import "github.com/dmitrymomot/saaskit/pkg/broadcast"
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/dmitrymomot/saaskit/pkg/broadcast"
)

func main() {
    // Create broadcaster with buffer size 10
    broadcaster := broadcast.NewMemoryBroadcaster[string](10)
    defer broadcaster.Close()

    ctx := context.Background()

    // Create subscriber
    subscriber := broadcaster.Subscribe(ctx)
    defer subscriber.Close()

    // Start receiving messages
    go func() {
        for msg := range subscriber.Receive(ctx) {
            fmt.Println("Received:", msg.Data)
        }
    }()

    // Broadcast messages
    broadcaster.Broadcast(ctx, broadcast.Message[string]{Data: "Hello"})
    broadcaster.Broadcast(ctx, broadcast.Message[string]{Data: "World"})
}
```

## Common Operations

### Typed Messages

```go
type Notification struct {
    Type    string
    UserID  string
    Message string
}

broadcaster := broadcast.NewMemoryBroadcaster[Notification](100)
broadcaster.Broadcast(ctx, broadcast.Message[Notification]{
    Data: Notification{
        Type:    "alert",
        UserID:  "user123",
        Message: "System update",
    },
})
```

### Multiple Subscribers

```go
// Each subscriber gets independent copy of messages
sub1 := broadcaster.Subscribe(ctx)
sub2 := broadcaster.Subscribe(ctx)

// Broadcast to all subscribers
broadcaster.Broadcast(ctx, broadcast.Message[string]{Data: "broadcast to all"})
```

### Context-Based Cleanup

```go
// Subscriber automatically removed when context cancels
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

sub := broadcaster.Subscribe(ctx)
// No manual cleanup needed - handled automatically
```

## Error Handling

```go
// Package errors:
var (
    ErrBroadcasterClosed = errors.New("broadcaster: closed")
    ErrSubscriberClosed  = errors.New("subscriber: closed")
)

// Usage:
if err := broadcaster.Broadcast(ctx, msg); err != nil {
    if errors.Is(err, broadcast.ErrBroadcasterClosed) {
        // handle closed broadcaster
    }
}
```

## API Documentation

```bash
# Full API documentation
go doc github.com/dmitrymomot/saaskit/pkg/broadcast

# Specific function or type
go doc github.com/dmitrymomot/saaskit/pkg/broadcast.NewMemoryBroadcaster
```

## Notes

- Messages are dropped for slow consumers to prevent blocking
- Buffer size determines how many messages can queue per subscriber
- All operations are idempotent and safe to call multiple times
