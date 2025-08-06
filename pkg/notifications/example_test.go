package notifications_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/notifications"
)

func ExampleManager_Send() {
	ctx := context.Background()

	// Create storage and deliverer
	storage := notifications.NewMemoryStorage()
	deliverer := notifications.NewBroadcastDeliverer(100)

	// Create manager
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
		log.Fatal(err)
	}

	fmt.Println("Notification sent successfully")
	// Output: Notification sent successfully
}

func ExampleManager_SendToUsers() {
	ctx := context.Background()

	// Setup
	storage := notifications.NewMemoryStorage()
	deliverer := notifications.NewBroadcastDeliverer(100)
	manager := notifications.NewManager(storage, deliverer)

	// Create a notification template
	template := notifications.Notification{
		Type:     notifications.TypeWarning,
		Priority: notifications.PriorityHigh,
		Title:    "System Maintenance",
		Message:  "The system will be unavailable from 2-3 AM UTC",
	}

	// Send to multiple users
	userIDs := []string{"user1", "user2", "user3"}
	err := manager.SendToUsers(ctx, userIDs, template)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Notification sent to %d users\n", len(userIDs))
	// Output: Notification sent to 3 users
}

func ExampleBroadcastDeliverer_Subscribe() {
	ctx := context.Background()

	// Create deliverer
	deliverer := notifications.NewBroadcastDeliverer(100)

	// Subscribe to user's notifications
	userID := "user123"
	subscriber := deliverer.Subscribe(ctx, userID)
	defer subscriber.Close()

	// Send a notification in a goroutine
	go func() {
		time.Sleep(10 * time.Millisecond) // Small delay to ensure subscriber is ready
		_ = deliverer.Deliver(ctx, notifications.Notification{
			ID:        "notif-1",
			UserID:    userID,
			Type:      notifications.TypeInfo,
			Title:     "Real-time notification",
			Message:   "This was delivered in real-time",
			CreatedAt: time.Now(),
		})
	}()

	// Receive the notification
	select {
	case msg := <-subscriber.Receive(ctx):
		fmt.Printf("Received: %s\n", msg.Data.Title)
	case <-time.After(100 * time.Millisecond):
		fmt.Println("No notification received")
	}

	// Output: Received: Real-time notification
}

func ExampleNotification_withActions() {
	// Create a notification with action buttons
	notif := notifications.Notification{
		ID:       "notif-123",
		UserID:   "user456",
		Type:     notifications.TypeInfo,
		Priority: notifications.PriorityNormal,
		Title:    "New Message",
		Message:  "You have received a new message from John Doe",
		Actions: []notifications.Action{
			{
				Label: "View Message",
				URL:   "/messages/789",
				Style: "primary",
			},
			{
				Label: "Mark as Read",
				URL:   "/api/messages/789/read",
				Style: "secondary",
			},
		},
		CreatedAt: time.Now(),
	}

	fmt.Printf("Notification has %d actions\n", len(notif.Actions))
	// Output: Notification has 2 actions
}

func ExampleManager_List() {
	ctx := context.Background()

	// Setup
	storage := notifications.NewMemoryStorage()
	manager := notifications.NewManager(storage, nil)

	// Create some notifications
	for i := 0; i < 5; i++ {
		_ = manager.Send(ctx, notifications.Notification{
			ID:      fmt.Sprintf("notif-%d", i),
			UserID:  "user123",
			Type:    notifications.TypeInfo,
			Title:   fmt.Sprintf("Notification %d", i),
			Message: "Test message",
		})
	}

	// List notifications with filtering
	notifs, err := manager.List(ctx, "user123", notifications.ListOptions{
		Limit:      3,
		OnlyUnread: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d unread notifications\n", len(notifs))
	// Output: Found 3 unread notifications
}

func ExampleManager_MarkRead() {
	ctx := context.Background()

	// Setup
	storage := notifications.NewMemoryStorage()
	manager := notifications.NewManager(storage, nil)

	// Create a notification
	_ = manager.Send(ctx, notifications.Notification{
		ID:      "notif-1",
		UserID:  "user123",
		Type:    notifications.TypeInfo,
		Title:   "Test",
		Message: "Test message",
	})

	// Mark as read
	err := manager.MarkRead(ctx, "user123", "notif-1")
	if err != nil {
		log.Fatal(err)
	}

	// Check unread count
	count, _ := manager.CountUnread(ctx, "user123")
	fmt.Printf("Unread notifications: %d\n", count)
	// Output: Unread notifications: 0
}

func ExampleMultiDeliverer() {
	ctx := context.Background()

	// Create multiple deliverers
	broadcastDeliverer := notifications.NewBroadcastDeliverer(100)
	// In real app, you'd have email, push, etc.
	noOpDeliverer := &notifications.NoOpDeliverer{}

	// Combine them
	multiDeliverer := notifications.NewMultiDeliverer([]notifications.Deliverer{
		broadcastDeliverer,
		noOpDeliverer,
	})

	// Use with manager
	storage := notifications.NewMemoryStorage()
	manager := notifications.NewManager(storage, multiDeliverer)

	// Send notification through all channels
	err := manager.Send(ctx, notifications.Notification{
		UserID:  "user123",
		Type:    notifications.TypeSuccess,
		Title:   "Multi-channel",
		Message: "Delivered through multiple channels",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Notification sent through multiple channels")
	// Output: Notification sent through multiple channels
}
