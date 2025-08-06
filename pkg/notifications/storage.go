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

// ListOptions provides filtering and pagination options for listing notifications.
type ListOptions struct {
	Limit      int        // Maximum number of notifications to return (0 = no limit)
	Offset     int        // Number of notifications to skip for pagination
	OnlyUnread bool       // When true, only return unread notifications
	Types      []Type     // If specified, only return notifications of these types
	Since      *time.Time // If specified, only return notifications created after this time
}
