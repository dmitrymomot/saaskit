// Package notifications provides a transport-agnostic notification system
// with pluggable storage and delivery mechanisms.
//
// The package is designed to be used as a utility library that can be integrated
// with any transport layer (HTTP, WebSocket, gRPC, etc.) while maintaining
// clean separation of concerns.
//
// # Architecture
//
// The package follows a layered architecture:
//
//   - Storage: Handles persistence and CRUD operations
//   - Deliverer: Manages real-time notification delivery
//   - Manager: Orchestrates service and deliverer
//
// # Basic Usage
//
//	// Create storage (persistence layer)
//	storage := notifications.NewMemoryStorage()
//
//	// Create deliverer (real-time delivery)
//	deliverer := notifications.NewBroadcastDeliverer(100)
//
//	// Create manager (orchestration)
//	manager := notifications.NewManager(storage, deliverer)
//
//	// Send a notification
//	err := manager.Send(ctx, notifications.Notification{
//	    UserID:  "user123",
//	    Type:    notifications.TypeInfo,
//	    Title:   "Welcome!",
//	    Message: "Thanks for joining our platform",
//	})
//
// # Transport Integration
//
// The package is transport-agnostic. You can integrate it with any transport layer:
//
//	// HTTP SSE Handler
//	func NotificationHandler(deliverer *notifications.BroadcastDeliverer) http.HandlerFunc {
//	    return func(w http.ResponseWriter, r *http.Request) {
//	        userID := getUserID(r)
//	        sub := deliverer.Subscribe(r.Context(), userID)
//	        defer sub.Close()
//
//	        // Set up SSE
//	        w.Header().Set("Content-Type", "text/event-stream")
//
//	        // Stream notifications
//	        for notif := range sub.Receive(r.Context()) {
//	            data, _ := json.Marshal(notif.Data)
//	            fmt.Fprintf(w, "data: %s\n\n", data)
//	            w.(http.Flusher).Flush()
//	        }
//	    }
//	}
//
// # Custom Deliverers
//
// You can implement custom deliverers for different channels:
//
//	type EmailDeliverer struct {
//	    emailClient *email.Client
//	}
//
//	func (e *EmailDeliverer) Deliver(ctx context.Context, notif notifications.Notification) error {
//	    // Send email for high-priority notifications
//	    if notif.Priority >= notifications.PriorityHigh {
//	        return e.emailClient.Send(...)
//	    }
//	    return nil
//	}
//
// # Storage Implementations
//
// The package includes a memory-based storage for development.
// For production, implement the Storage interface with your database:
//
//	type PostgresStorage struct {
//	    db *sql.DB
//	}
//
//	func (s *PostgresStorage) Create(ctx context.Context, notif Notification) error {
//	    // Store in PostgreSQL
//	}
//
// # Notification Types
//
// Four notification types are provided:
//   - TypeInfo: Informational messages
//   - TypeSuccess: Success confirmations
//   - TypeWarning: Warning messages
//   - TypeError: Error notifications
//
// # Priority Levels
//
// Notifications can have different priority levels:
//   - PriorityLow: Non-urgent, can be batched
//   - PriorityNormal: Standard notifications
//   - PriorityHigh: Important, should be delivered immediately
//   - PriorityUrgent: Critical, may trigger additional channels
package notifications