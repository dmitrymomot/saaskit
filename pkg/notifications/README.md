# notifications

Transport-agnostic notification system with pluggable storage and delivery mechanisms.

## Features

- **Transport-agnostic** - Works with any transport layer (HTTP, WebSocket, gRPC)
- **Pluggable storage** - Interface-based design for any database backend
- **Real-time delivery** - Broadcast integration for live notifications
- **Type-safe** - Strongly typed notifications with compile-time safety
- **Priority levels** - Route notifications based on importance
- **Batch operations** - Efficient bulk sending and management

## Installation

```go
import "github.com/dmitrymomot/saaskit/pkg/notifications"
```

## Usage

### Basic Setup

```go
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
```

### Notification Types

```go
// Information
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

### With Actions

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

### Batch Operations

```go
// Send to multiple users
template := notifications.Notification{
    Type:    notifications.TypeInfo,
    Title:   "System Maintenance",
    Message: "Scheduled maintenance at 2 AM UTC",
}
err := manager.SendToUsers(ctx, []string{"user1", "user2", "user3"}, template)

// Mark all as read
err := manager.MarkAllRead(ctx, "user123")
```

### Listing and Filtering

```go
// Get recent notifications
notifs, err := manager.List(ctx, "user123", notifications.ListOptions{
    Limit:      10,
    OnlyUnread: true,
    Types:      []notifications.Type{notifications.TypeWarning, notifications.TypeError},
})

// Count unread
count, err := manager.CountUnread(ctx, "user123")
```

## Transport Integration Examples

### HTTP Server-Sent Events (SSE)

```go
func NotificationStreamHandler(deliverer *notifications.BroadcastDeliverer) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := getUserID(r) // From auth middleware
        
        // Subscribe to notifications
        sub := deliverer.Subscribe(r.Context(), userID)
        defer sub.Close()
        
        // Setup SSE
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

// Mount in your router
mux.HandleFunc("/notifications/stream", NotificationStreamHandler(deliverer))
```

### DataStar Integration

```go
import (
    "github.com/delaneyj/datastar"
    "github.com/dmitrymomot/saaskit/handler"
)

func DataStarNotificationHandler(manager *notifications.Manager) handler.HandlerFunc[handler.Context, struct{}] {
    return func(ctx handler.Context, _ struct{}) handler.Response {
        userID := ctx.UserID() // From auth context
        
        // Get broadcast deliverer
        deliverer := manager.Deliverer().(*notifications.BroadcastDeliverer)
        sub := deliverer.Subscribe(ctx.Request().Context(), userID)
        defer sub.Close()
        
        // First, send any unread notifications
        unread, _ := manager.List(ctx.Request().Context(), userID, notifications.ListOptions{
            OnlyUnread: true,
        })
        
        // Setup DataStar SSE
        sse := datastar.NewSSE(ctx.Writer(), ctx.Request())
        
        // Send unread
        for _, notif := range unread {
            sse.Send(datastar.Fragment(renderNotification(notif)))
        }
        
        // Stream new notifications
        for notif := range sub.Receive(ctx.Request().Context()) {
            sse.Send(datastar.Fragment(renderNotification(notif.Data)))
        }
        
        return handler.Empty()
    }
}

func renderNotification(n notifications.Notification) string {
    class := map[notifications.Type]string{
        notifications.TypeInfo:    "alert-info",
        notifications.TypeSuccess: "alert-success",
        notifications.TypeWarning: "alert-warning",
        notifications.TypeError:   "alert-error",
    }[n.Type]
    
    return fmt.Sprintf(`
        <div id="notif-%s" class="alert %s">
            <h4>%s</h4>
            <p>%s</p>
            <button data-action="click->notifications#dismiss" data-id="%s">×</button>
        </div>
    `, n.ID, class, n.Title, n.Message, n.ID)
}
```

### WebSocket Integration

```go
func WebSocketNotificationHandler(deliverer *notifications.BroadcastDeliverer) http.HandlerFunc {
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
        
        // Stream notifications over WebSocket
        for notif := range sub.Receive(r.Context()) {
            if err := conn.WriteJSON(notif.Data); err != nil {
                return
            }
        }
    }
}
```

### REST API Endpoints

```go
// List notifications
func ListNotifications(manager *notifications.Manager) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := getUserID(r)
        
        notifs, err := manager.List(r.Context(), userID, notifications.ListOptions{
            Limit:  20,
            Offset: getOffset(r),
        })
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        json.NewEncoder(w).Encode(notifs)
    }
}

// Mark as read
func MarkAsRead(manager *notifications.Manager) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := getUserID(r)
        notifID := chi.URLParam(r, "id")
        
        if err := manager.MarkRead(r.Context(), userID, notifID); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        w.WriteHeader(http.StatusNoContent)
    }
}

// Get unread count
func UnreadCount(manager *notifications.Manager) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := getUserID(r)
        
        count, err := manager.CountUnread(r.Context(), userID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        json.NewEncoder(w).Encode(map[string]int{"count": count})
    }
}
```

## Logging

The package supports structured logging with `slog` for error visibility:

```go
import "log/slog"

// Configure with custom logger
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// Add logger to BroadcastDeliverer
deliverer := notifications.NewBroadcastDeliverer(100, 
    notifications.WithBroadcastLogger(logger))

// Add logger to Manager
manager := notifications.NewManager(storage, deliverer,
    notifications.WithManagerLogger(logger))

// Add logger to MultiDeliverer
multiDeliverer := notifications.NewMultiDeliverer(
    []notifications.Deliverer{broadcastDeliverer, emailDeliverer},
    notifications.WithMultiDelivererLogger(logger))
```

Errors are logged when:
- Delivery fails in MultiDeliverer (logged as error)
- Delivery fails in Manager but storage succeeded (logged as warning)
- Broadcasting fails in BroadcastDeliverer (logged as error)
- Closing broadcasters fails (logged as error)

## Custom Deliverers

### Email Deliverer

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
            // Log error but continue
            continue
        }
    }
    return nil
}
```

### Push Notification Deliverer

```go
type PushDeliverer struct {
    fcmClient *fcm.Client
    deviceRepo DeviceRepository
}

func (p *PushDeliverer) Deliver(ctx context.Context, notif notifications.Notification) error {
    devices, err := p.deviceRepo.GetUserDevices(ctx, notif.UserID)
    if err != nil {
        return err
    }
    
    message := &fcm.Message{
        Notification: &fcm.Notification{
            Title: notif.Title,
            Body:  notif.Message,
        },
        Data: map[string]string{
            "notificationId": notif.ID,
            "type":          string(notif.Type),
        },
    }
    
    for _, device := range devices {
        message.Token = device.Token
        _, err := p.fcmClient.Send(ctx, message)
        // Continue on error
    }
    
    return nil
}
```

### Webhook Deliverer

```go
type WebhookDeliverer struct {
    httpClient *http.Client
    webhookURL string
}

func (w *WebhookDeliverer) Deliver(ctx context.Context, notif notifications.Notification) error {
    payload, err := json.Marshal(notif)
    if err != nil {
        return err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", w.webhookURL, bytes.NewReader(payload))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := w.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}
```

## Storage Implementations

### PostgreSQL Storage

```go
type PostgresStorage struct {
    db *sql.DB
}

func (s *PostgresStorage) Create(ctx context.Context, notif notifications.Notification) error {
    query := `
        INSERT INTO notifications (id, user_id, type, priority, title, message, data, read, created_at, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `
    
    data, _ := json.Marshal(notif.Data)
    _, err := s.db.ExecContext(ctx, query,
        notif.ID, notif.UserID, notif.Type, notif.Priority,
        notif.Title, notif.Message, data, notif.Read,
        notif.CreatedAt, notif.ExpiresAt,
    )
    return err
}

func (s *PostgresStorage) List(ctx context.Context, userID string, opts notifications.ListOptions) ([]notifications.Notification, error) {
    query := `
        SELECT id, type, priority, title, message, data, read, read_at, created_at, expires_at
        FROM notifications
        WHERE user_id = $1
    `
    
    args := []interface{}{userID}
    
    if opts.OnlyUnread {
        query += " AND read = false"
    }
    
    if len(opts.Types) > 0 {
        query += " AND type = ANY($2)"
        args = append(args, pq.Array(opts.Types))
    }
    
    query += " ORDER BY created_at DESC"
    
    if opts.Limit > 0 {
        query += fmt.Sprintf(" LIMIT %d OFFSET %d", opts.Limit, opts.Offset)
    }
    
    // Execute query and scan results...
}
```

## Testing

```go
func TestNotificationManager(t *testing.T) {
    ctx := context.Background()
    
    // Setup
    service := notifications.NewMemoryService()
    deliverer := notifications.NewBroadcastDeliverer(10)
    manager := notifications.NewManager(service, deliverer)
    
    // Test sending
    err := manager.Send(ctx, notifications.Notification{
        UserID:  "test-user",
        Type:    notifications.TypeInfo,
        Title:   "Test",
        Message: "Test message",
    })
    assert.NoError(t, err)
    
    // Test listing
    notifs, err := manager.List(ctx, "test-user", notifications.ListOptions{
        Limit: 10,
    })
    assert.NoError(t, err)
    assert.Len(t, notifs, 1)
    
    // Test real-time delivery
    sub := deliverer.Subscribe(ctx, "test-user")
    defer sub.Close()
    
    go func() {
        manager.Send(ctx, notifications.Notification{
            UserID:  "test-user",
            Type:    notifications.TypeSuccess,
            Title:   "Real-time",
            Message: "This is real-time",
        })
    }()
    
    select {
    case notif := <-sub.Receive(ctx):
        assert.Equal(t, "Real-time", notif.Data.Title)
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for notification")
    }
}
```

## Notes

- Notifications are stored first, then delivered (store-and-forward pattern)
- Delivery is best-effort and non-blocking
- The BroadcastDeliverer creates a separate broadcaster per user
- Memory storage is for development only - use a database in production
- All operations are safe for concurrent use