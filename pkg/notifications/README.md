# notifications

Transport-agnostic notification system with pluggable storage and delivery mechanisms.

## Features

- Transport-agnostic design for any transport layer (HTTP, WebSocket, gRPC)
- Pluggable storage interface for any database backend
- Real-time delivery via broadcast integration
- Type-safe notification handling with compile-time safety
- Priority-based routing and delivery
- Batch operations for efficient bulk processing

## Installation

```bash
go get github.com/dmitrymomot/saaskit/pkg/notifications
```

## Architecture

The package follows a layered architecture with clear separation of concerns:

- **Storage**: Handles notification persistence and CRUD operations
- **Deliverer**: Manages real-time notification delivery across channels
- **Manager**: Orchestrates storage and delivery operations

## Usage

```go
package main

import (
    "context"
    "github.com/dmitrymomot/saaskit/pkg/notifications"
)

func main() {
    ctx := context.Background()
    
    // Create storage (persistence layer)
    storage := notifications.NewMemoryStorage()
    
    // Create deliverer (real-time delivery)
    deliverer := notifications.NewBroadcastDeliverer(100)
    
    // Create manager (orchestration)
    manager := notifications.NewManager(storage, deliverer)
    
    // Send a notification
    err := manager.Send(ctx, notifications.Notification{
        UserID:   "user123",
        Type:     notifications.TypeInfo,
        Priority: notifications.PriorityNormal,
        Title:    "Welcome!",
        Message:  "Thanks for joining our platform",
    })
    if err != nil {
        panic(err)
    }
}
```

## Common Operations

### Notification Types and Priorities

```go
// Notification types
notifications.TypeInfo    // General information
notifications.TypeSuccess // Success confirmations  
notifications.TypeWarning // Warning messages
notifications.TypeError   // Error notifications

// Priority levels
notifications.PriorityLow    // Can be batched
notifications.PriorityNormal // Standard delivery
notifications.PriorityHigh   // Immediate delivery
notifications.PriorityUrgent // Critical, multi-channel
```

### Batch Operations

```go
// Send to multiple users
template := notifications.Notification{
    Type:    notifications.TypeWarning,
    Title:   "System Maintenance",
    Message: "Scheduled maintenance at 2 AM UTC",
}
err := manager.SendToUsers(ctx, []string{"user1", "user2"}, template)

// Mark all as read
err := manager.MarkAllRead(ctx, "user123")
```

### Notifications with Actions

```go
notif := notifications.Notification{
    UserID:  "user123",
    Type:    notifications.TypeInfo,
    Title:   "New Message",
    Message: "You have a new message from John",
    Actions: []notifications.Action{
        {Label: "View", URL: "/messages/123", Style: "primary"},
        {Label: "Dismiss", URL: "#", Style: "secondary"},
    },
}
```

### Filtering and Listing

```go
// Get recent notifications
notifs, err := manager.List(ctx, "user123", notifications.ListOptions{
    Limit:      10,
    OnlyUnread: true,
    Types:      []notifications.Type{notifications.TypeWarning},
})

// Count unread notifications
count, err := manager.CountUnread(ctx, "user123")
```

## Transport Integration

### HTTP Server-Sent Events (SSE)

```go
func NotificationStreamHandler(deliverer *notifications.BroadcastDeliverer) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := getUserID(r) // From auth middleware
        
        sub := deliverer.Subscribe(r.Context(), userID)
        defer sub.Close()
        
        // Setup SSE headers
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")
        
        // Stream notifications
        for {
            select {
            case <-r.Context().Done():
                return
            case notif := <-sub.Receive(r.Context()):
                data, _ := json.Marshal(notif.Data)
                fmt.Fprintf(w, "data: %s\n\n", data)
                w.(http.Flusher).Flush()
            }
        }
    }
}
```

### DataStar Integration

```go
import (
    "github.com/starfederation/datastar-go"
    "github.com/dmitrymomot/saaskit/handler"
)

func DataStarHandler(manager *notifications.Manager) handler.HandlerFunc[handler.Context, struct{}] {
    return func(ctx handler.Context, _ struct{}) handler.Response {
        userID := ctx.UserID()
        
        deliverer := manager.Deliverer().(*notifications.BroadcastDeliverer)
        sub := deliverer.Subscribe(ctx.Request().Context(), userID)
        defer sub.Close()
        
        sse := datastar.NewSSE(ctx.Writer(), ctx.Request())
        
        // Stream new notifications
        for notif := range sub.Receive(ctx.Request().Context()) {
            sse.Send(datastar.Fragment(renderNotification(notif.Data)))
        }
        
        return handler.Empty()
    }
}
```

### WebSocket Integration

```go
func WebSocketHandler(deliverer *notifications.BroadcastDeliverer) http.HandlerFunc {
    upgrader := websocket.Upgrader{}
    
    return func(w http.ResponseWriter, r *http.Request) {
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            return
        }
        defer conn.Close()
        
        userID := getUserID(r)
        sub := deliverer.Subscribe(r.Context(), userID)
        defer sub.Close()
        
        for notif := range sub.Receive(r.Context()) {
            if err := conn.WriteJSON(notif.Data); err != nil {
                return
            }
        }
    }
}
```

## Error Handling

```go
import "errors"

if errors.Is(err, notifications.ErrNotificationNotFound) {
    // Handle notification not found
}
```

## Custom Deliverers

### Email Deliverer Example

```go
type EmailDeliverer struct {
    emailClient *email.Client
    userRepo    UserRepository
}

func (e *EmailDeliverer) Deliver(ctx context.Context, notif notifications.Notification) error {
    // Only send emails for high priority
    if notif.Priority < notifications.PriorityHigh {
        return nil
    }
    
    user, err := e.userRepo.Get(ctx, notif.UserID)
    if err != nil {
        return err
    }
    
    return e.emailClient.Send(ctx, email.Message{
        To:      user.Email,
        Subject: notif.Title,
        Body:    notif.Message,
    })
}

func (e *EmailDeliverer) DeliverBatch(ctx context.Context, notifs []notifications.Notification) error {
    for _, n := range notifs {
        if err := e.Deliver(ctx, n); err != nil {
            continue // Log error but continue with remaining
        }
    }
    return nil
}
```

### Multi-Channel Delivery

```go
// Combine multiple delivery channels
multiDeliverer := notifications.NewMultiDeliverer([]notifications.Deliverer{
    broadcastDeliverer,
    emailDeliverer,
    pushDeliverer,
})

manager := notifications.NewManager(storage, multiDeliverer)
```

## Configuration

### With Logging

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// Configure manager with logger
manager := notifications.NewManager(storage, deliverer,
    notifications.WithManagerLogger(logger))

// Configure deliverer with logger
deliverer := notifications.NewBroadcastDeliverer(100,
    notifications.WithBroadcastLogger(logger))
```

## Storage Implementations

### PostgreSQL Example

```go
type PostgresStorage struct {
    db *sql.DB
}

func (s *PostgresStorage) Create(ctx context.Context, notif notifications.Notification) error {
    query := `
        INSERT INTO notifications (id, user_id, type, priority, title, message, read, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `
    
    _, err := s.db.ExecContext(ctx, query,
        notif.ID, notif.UserID, notif.Type, notif.Priority,
        notif.Title, notif.Message, notif.Read, notif.CreatedAt,
    )
    return err
}
```

## API Documentation

For detailed API documentation:

```bash
go doc -all ./pkg/notifications
```

## Notes

- Notifications are stored first, then delivered (store-and-forward pattern)
- Delivery is best-effort and non-blocking to ensure reliability
- Memory storage is for development only - use database storage in production
- All operations are safe for concurrent use
- BroadcastDeliverer creates separate broadcasters per user for isolation