package notifications

import (
	"context"
	"log/slog"
	"sync"

	"github.com/dmitrymomot/saaskit/pkg/broadcast"
	"github.com/dmitrymomot/saaskit/pkg/cache"
	"github.com/dmitrymomot/saaskit/pkg/logger"
)

// BroadcastDeliverer uses the broadcast package for real-time notification delivery.
type BroadcastDeliverer struct {
	userBroadcasters *cache.LRUCache[string, broadcast.Broadcaster[Notification]]
	bufferSize       int
	maxBroadcasters  int
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

// WithMaxBroadcasters sets the maximum number of user broadcasters.
// When this limit is reached, the least recently used broadcaster is evicted.
// Default is 10,000 if not specified.
func WithMaxBroadcasters(limit int) BroadcastDelivererOption {
	return func(b *BroadcastDeliverer) {
		if limit > 0 {
			b.maxBroadcasters = limit
		}
	}
}

// NewBroadcastDeliverer creates a new broadcast-based deliverer.
func NewBroadcastDeliverer(bufferSize int, opts ...BroadcastDelivererOption) *BroadcastDeliverer {
	b := &BroadcastDeliverer{
		bufferSize:      bufferSize,
		maxBroadcasters: 10000, // Default max broadcasters
		logger:          slog.Default(),
	}

	for _, opt := range opts {
		opt(b)
	}

	// Initialize LRU cache with the configured capacity
	b.userBroadcasters = cache.NewLRUCache[string, broadcast.Broadcaster[Notification]](b.maxBroadcasters)

	// Set eviction callback to close evicted broadcasters
	b.userBroadcasters.SetEvictCallback(func(userID string, broadcaster broadcast.Broadcaster[Notification]) {
		if err := broadcaster.Close(); err != nil {
			b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to close evicted broadcaster",
				logger.UserID(userID),
				logger.Error(err),
			)
		}
	})

	return b
}

func (d *BroadcastDeliverer) Deliver(ctx context.Context, notif Notification) error {
	// Get or create broadcaster for the user
	d.mu.Lock()
	b, exists := d.userBroadcasters.Get(notif.UserID)
	if !exists {
		b = broadcast.NewMemoryBroadcaster[Notification](d.bufferSize)
		// Put will evict LRU broadcaster if at capacity
		d.userBroadcasters.Put(notif.UserID, b)
	}
	d.mu.Unlock()

	return b.Broadcast(ctx, broadcast.Message[Notification]{Data: notif})
}

func (d *BroadcastDeliverer) DeliverBatch(ctx context.Context, notifs []Notification) error {
	// Group notifications by user to optimize broadcast operations
	// This reduces lock contention by batching operations per user
	userNotifs := make(map[string][]Notification)
	for _, n := range notifs {
		userNotifs[n.UserID] = append(userNotifs[n.UserID], n)
	}

	// Deliver to each user
	for userID, userNotifications := range userNotifs {
		// Get or create broadcaster for the user
		d.mu.Lock()
		b, exists := d.userBroadcasters.Get(userID)
		if !exists {
			b = broadcast.NewMemoryBroadcaster[Notification](d.bufferSize)
			// Put will evict LRU broadcaster if at capacity
			d.userBroadcasters.Put(userID, b)
		}
		d.mu.Unlock()

		for _, notif := range userNotifications {
			if err := b.Broadcast(ctx, broadcast.Message[Notification]{Data: notif}); err != nil {
				// Continue with remaining notifications even if one fails
				// This prevents a single bad notification from blocking the entire batch
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

	b, exists := d.userBroadcasters.Get(userID)
	if !exists {
		b = broadcast.NewMemoryBroadcaster[Notification](d.bufferSize)
		// Put will evict LRU broadcaster if at capacity
		d.userBroadcasters.Put(userID, b)
	}

	return b.Subscribe(ctx)
}

// Close closes all user broadcasters.
func (d *BroadcastDeliverer) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Clear will call the eviction callback for each broadcaster
	d.userBroadcasters.Clear()
	return nil
}
