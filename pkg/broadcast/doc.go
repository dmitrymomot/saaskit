// Package broadcast provides type-safe message broadcasting with subscriber management.
// It enables one-to-many communication patterns with automatic cleanup and buffering.
//
// The package uses Go generics to provide type safety at compile time, ensuring
// messages are strongly typed throughout the broadcasting system.
//
// Basic usage:
//
//	broadcaster := broadcast.NewMemoryBroadcaster[string](10)
//	defer broadcaster.Close()
//
//	ctx := context.Background()
//	subscriber := broadcaster.Subscribe(ctx)
//	defer subscriber.Close()
//
//	// Broadcast a message
//	broadcaster.Broadcast(ctx, broadcast.Message[string]{Data: "hello"})
//
//	// Receive messages
//	for msg := range subscriber.Receive(ctx) {
//		fmt.Println(msg.Data)
//	}
//
// The memory implementation automatically handles subscriber cleanup when:
// - The subscriber's context is cancelled
// - The subscriber's buffer is full (drops slow subscribers)
// - The broadcaster is closed
package broadcast