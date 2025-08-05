// Package broadcast provides a type-safe, generic broadcasting system for pub/sub messaging.
//
// The package implements a hub-based architecture where publishers send messages to channels
// and subscribers receive them asynchronously. It supports both in-memory operation and
// pluggable storage backends for message persistence.
//
// # Core Components
//
// Hub: The central message broker that manages channels and subscribers
// Message: Generic message container with type-safe payloads
// Subscriber: Represents a subscription to receive messages from a channel
// Storage: Interface for message persistence (implementations provided separately)
//
// # Basic Usage
//
//	// Create a hub for string messages
//	hub := broadcast.NewHub[string](broadcast.HubConfig{
//	    DefaultBufferSize: 100,
//	})
//	defer hub.Close()
//
//	// Subscribe to a channel
//	ctx := context.Background()
//	sub, err := hub.Subscribe(ctx, "news",
//	    broadcast.WithBufferSize(50),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer sub.Close()
//
//	// Publish messages
//	err = hub.Publish(ctx, "news", "Breaking news!")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Receive messages
//	for msg := range sub.Messages() {
//	    fmt.Printf("Received: %s\n", msg.Payload)
//	}
//
// # Advanced Features
//
// Message Replay: New subscribers can receive recent messages
//
//	sub, err := hub.Subscribe(ctx, "events",
//	    broadcast.WithReplay(10), // Replay last 10 messages
//	)
//
// Slow Consumer Handling: Automatic detection and removal of slow consumers
//
//	hub := broadcast.NewHub[Event](broadcast.HubConfig{
//	    SlowConsumerTimeout: 5 * time.Second,
//	})
//
// Storage Integration: Persist messages for durability and replay
//
//	hub := broadcast.NewHub[Message](broadcast.HubConfig{
//	    Storage: myStorageImpl, // Implement broadcast.Storage interface
//	})
//
// # Thread Safety
//
// All hub operations are thread-safe and can be called concurrently.
// Subscribers should only be used by a single goroutine for receiving messages.
//
// # Integration with SSE
//
// The broadcast package integrates seamlessly with the handler package's SSE support:
//
//	func ChatHandler(hub broadcast.Hub[ChatMessage]) handler.HandlerFunc[handler.StreamContext, JoinRequest] {
//	    return func(ctx handler.StreamContext, req JoinRequest) handler.Response {
//	        sub, err := hub.Subscribe(ctx.Request().Context(), "chat")
//	        if err != nil {
//	            return handler.Error(err)
//	        }
//	        defer sub.Close()
//
//	        for msg := range sub.Messages() {
//	            if err := ctx.SendJSON(msg.Payload); err != nil {
//	                return handler.Error(err)
//	            }
//	        }
//	        return handler.OK()
//	    }
//	}
package broadcast
