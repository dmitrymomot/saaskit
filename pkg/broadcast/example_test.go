package broadcast_test

import (
	"context"
	"fmt"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/broadcast"
)

func ExampleMemoryBroadcaster() {
	b := broadcast.NewMemoryBroadcaster[string](10)
	defer b.Close()

	ctx := context.Background()
	sub := b.Subscribe(ctx)

	go func() {
		for msg := range sub.Receive(ctx) {
			fmt.Println("Received:", msg.Data)
		}
	}()

	b.Broadcast(ctx, broadcast.Message[string]{Data: "Hello"})
	b.Broadcast(ctx, broadcast.Message[string]{Data: "World"})

	time.Sleep(10 * time.Millisecond)
}

func ExampleMemoryBroadcaster_multipleSubscribers() {
	type Notification struct {
		Type    string
		Message string
	}

	b := broadcast.NewMemoryBroadcaster[Notification](10)
	defer b.Close()

	ctx := context.Background()

	alice := b.Subscribe(ctx)
	go func() {
		for msg := range alice.Receive(ctx) {
			fmt.Printf("Alice received: %s - %s\n", msg.Data.Type, msg.Data.Message)
		}
	}()

	bob := b.Subscribe(ctx)
	go func() {
		for msg := range bob.Receive(ctx) {
			fmt.Printf("Bob received: %s - %s\n", msg.Data.Type, msg.Data.Message)
		}
	}()

	notification := broadcast.Message[Notification]{
		Data: Notification{
			Type:    "payment",
			Message: "Payment received",
		},
	}
	b.Broadcast(ctx, notification)

	time.Sleep(10 * time.Millisecond)
}

func ExampleMemoryBroadcaster_contextCancellation() {
	b := broadcast.NewMemoryBroadcaster[int](10)
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	sub := b.Subscribe(ctx)

	go func() {
		for msg := range sub.Receive(ctx) {
			fmt.Printf("Received: %d\n", msg.Data)
		}
		fmt.Println("Subscription ended")
	}()

	for i := range 5 {
		b.Broadcast(context.Background(), broadcast.Message[int]{Data: i})
		time.Sleep(30 * time.Millisecond)
	}
}

func ExampleMemoryBroadcaster_errorHandling() {
	// Demonstrate proper resource cleanup with defer
	b := broadcast.NewMemoryBroadcaster[string](10)
	defer b.Close() // Always close broadcaster when done

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Always cancel context when done

	sub := b.Subscribe(ctx)
	defer sub.Close() // Close subscriber explicitly (though context cancellation also cleans up)

	// Process messages until channel closes
	done := make(chan bool)
	go func() {
		for msg := range sub.Receive(ctx) {
			fmt.Println("Processing:", msg.Data)
		}
		done <- true
		fmt.Println("Channel closed, cleanup complete")
	}()

	// Send some messages
	b.Broadcast(ctx, broadcast.Message[string]{Data: "Message 1"})
	b.Broadcast(ctx, broadcast.Message[string]{Data: "Message 2"})

	// Simulate cleanup
	time.Sleep(10 * time.Millisecond)
	cancel() // This triggers subscriber cleanup
	<-done
}

func ExampleMemoryBroadcaster_closedBroadcaster() {
	b := broadcast.NewMemoryBroadcaster[string](10)
	ctx := context.Background()

	// Close the broadcaster
	b.Close()

	// Operations after close are safe but have no effect
	sub := b.Subscribe(ctx) // Returns a closed subscriber
	
	// Check if channel is closed immediately
	select {
	case _, ok := <-sub.Receive(ctx):
		if !ok {
			fmt.Println("Subscriber channel is closed")
		}
	default:
		fmt.Println("No message available")
	}

	// Broadcast after close has no effect
	err := b.Broadcast(ctx, broadcast.Message[string]{Data: "Won't be sent"})
	if err == nil {
		fmt.Println("Broadcast returned nil (no-op)")
	}
}

func ExampleMemoryBroadcaster_slowConsumer() {
	// Small buffer to demonstrate dropping
	b := broadcast.NewMemoryBroadcaster[int](2)
	defer b.Close()

	ctx := context.Background()
	sub := b.Subscribe(ctx)

	// Slow consumer that can't keep up
	received := make([]int, 0)
	done := make(chan bool)
	
	go func() {
		for msg := range sub.Receive(ctx) {
			received = append(received, msg.Data)
			// Simulate slow processing
			time.Sleep(50 * time.Millisecond)
		}
		done <- true
	}()

	// Send messages faster than consumer can process
	for i := range 10 {
		b.Broadcast(ctx, broadcast.Message[int]{Data: i})
		time.Sleep(5 * time.Millisecond) // Much faster than consumer
	}

	// Give consumer time to process remaining buffered messages
	time.Sleep(200 * time.Millisecond)
	sub.Close()
	<-done

	// Consumer will have missed some messages due to buffer overflow
	fmt.Printf("Sent 10 messages, received %d (some dropped due to slow consumer)\n", len(received))
}