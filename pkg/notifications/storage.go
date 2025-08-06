package notifications

import (
	"context"
	"time"
)

// Storage handles notification persistence and retrieval.
type Storage interface {
	// Create stores a new notification.
	Create(ctx context.Context, notif Notification) error

	// Get retrieves a single notification.
	Get(ctx context.Context, userID, notifID string) (*Notification, error)

	// List returns notifications for a user.
	List(ctx context.Context, userID string, opts ListOptions) ([]Notification, error)

	// MarkRead marks notification(s) as read.
	MarkRead(ctx context.Context, userID string, notifIDs ...string) error

	// Delete removes notification(s).
	Delete(ctx context.Context, userID string, notifIDs ...string) error

	// CountUnread returns unread count for user.
	CountUnread(ctx context.Context, userID string) (int, error)
}

// ListOptions provides filtering options for listing notifications.
type ListOptions struct {
	Limit      int        // Maximum number of notifications to return
	Offset     int        // Number of notifications to skip
	OnlyUnread bool       // Filter to only unread notifications
	Types      []Type     // Filter by notification types
	Since      *time.Time // Filter notifications created after this time
}