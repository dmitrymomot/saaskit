package notifications

import (
	"context"
	"log/slog"
	"sync"

	"github.com/dmitrymomot/saaskit/pkg/broadcast"
	"github.com/dmitrymomot/saaskit/pkg/logger"
)

// BroadcastDeliverer uses the broadcast package for real-time notification delivery.
type BroadcastDeliverer struct {
	userBroadcasters map[string]broadcast.Broadcaster[Notification]
	bufferSize       int
	logger           *slog.Logger
	mu               sync.RWMutex
}

// BroadcastDelivererOption configures a BroadcastDeliverer.
type BroadcastDelivererOption func(*BroadcastDeliverer)

// WithBroadcastLogger sets the logger for the BroadcastDeliverer.
func WithBroadcastLogger(logger *slog.Logger) BroadcastDelivererOption {
	return func(b *BroadcastDeliverer) {
		b.logger = logger
	}
}

// NewBroadcastDeliverer creates a new broadcast-based deliverer.
func NewBroadcastDeliverer(bufferSize int, opts ...BroadcastDelivererOption) *BroadcastDeliverer {
	b := &BroadcastDeliverer{
		userBroadcasters: make(map[string]broadcast.Broadcaster[Notification]),
		bufferSize:       bufferSize,
		logger:           slog.Default(),
	}
	
	for _, opt := range opts {
		opt(b)
	}
	
	return b
}

// Deliver sends a notification to the user's broadcast channel.
func (d *BroadcastDeliverer) Deliver(ctx context.Context, notif Notification) error {
	d.mu.Lock()
	b, exists := d.userBroadcasters[notif.UserID]
	if !exists {
		b = broadcast.NewMemoryBroadcaster[Notification](d.bufferSize)
		d.userBroadcasters[notif.UserID] = b
	}
	d.mu.Unlock()

	return b.Broadcast(ctx, broadcast.Message[Notification]{Data: notif})
}

// DeliverBatch sends multiple notifications.
func (d *BroadcastDeliverer) DeliverBatch(ctx context.Context, notifs []Notification) error {
	// Group notifications by user
	userNotifs := make(map[string][]Notification)
	for _, n := range notifs {
		userNotifs[n.UserID] = append(userNotifs[n.UserID], n)
	}

	// Deliver to each user
	for userID, userNotifications := range userNotifs {
		d.mu.Lock()
		b, exists := d.userBroadcasters[userID]
		if !exists {
			b = broadcast.NewMemoryBroadcaster[Notification](d.bufferSize)
			d.userBroadcasters[userID] = b
		}
		d.mu.Unlock()

		for _, notif := range userNotifications {
			if err := b.Broadcast(ctx, broadcast.Message[Notification]{Data: notif}); err != nil {
				// Continue with other notifications on error
				d.logger.LogAttrs(ctx, slog.LevelError, "Failed to broadcast notification",
					slog.String("notification_id", notif.ID),
					logger.UserID(notif.UserID),
					logger.Error(err),
				)
				continue
			}
		}
	}

	return nil
}

// Subscribe returns a subscriber for user's notifications.
// This is used by transport layers (HTTP handlers, WebSocket, etc.) to receive real-time notifications.
func (d *BroadcastDeliverer) Subscribe(ctx context.Context, userID string) broadcast.Subscriber[Notification] {
	d.mu.Lock()
	defer d.mu.Unlock()

	b, exists := d.userBroadcasters[userID]
	if !exists {
		b = broadcast.NewMemoryBroadcaster[Notification](d.bufferSize)
		d.userBroadcasters[userID] = b
	}

	return b.Subscribe(ctx)
}

// Close closes all user broadcasters.
func (d *BroadcastDeliverer) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for userID, b := range d.userBroadcasters {
		if err := b.Close(); err != nil {
			// Continue closing others even if one fails
			d.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to close broadcaster",
				logger.UserID(userID),
				logger.Error(err),
			)
			continue
		}
	}

	// Clear the map
	d.userBroadcasters = make(map[string]broadcast.Broadcaster[Notification])
	return nil
}