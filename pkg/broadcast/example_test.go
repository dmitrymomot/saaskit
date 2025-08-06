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