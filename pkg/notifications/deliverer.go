package notifications

import (
	"context"
	"log/slog"

	"github.com/dmitrymomot/saaskit/pkg/logger"
)

// Deliverer handles real-time notification delivery.
type Deliverer interface {
	// Deliver sends notification to user through specific channel.
	Deliver(ctx context.Context, notif Notification) error

	// DeliverBatch sends multiple notifications.
	DeliverBatch(ctx context.Context, notifs []Notification) error
}

// MultiDeliverer combines multiple delivery channels.
type MultiDeliverer struct {
	deliverers []Deliverer
	logger     *slog.Logger
}

// MultiDelivererOption configures a MultiDeliverer.
type MultiDelivererOption func(*MultiDeliverer)

// WithMultiDelivererLogger sets the logger for the MultiDeliverer.
func WithMultiDelivererLogger(logger *slog.Logger) MultiDelivererOption {
	return func(m *MultiDeliverer) {
		m.logger = logger
	}
}

// NewMultiDeliverer creates a new multi-channel deliverer.
func NewMultiDeliverer(deliverers []Deliverer, opts ...MultiDelivererOption) *MultiDeliverer {
	m := &MultiDeliverer{
		deliverers: deliverers,
		logger:     slog.Default(),
	}
	
	for _, opt := range opts {
		opt(m)
	}
	
	return m
}

// Deliver sends notification through all configured channels.
func (m *MultiDeliverer) Deliver(ctx context.Context, notif Notification) error {
	for i, d := range m.deliverers {
		if err := d.Deliver(ctx, notif); err != nil {
			// Log but don't fail - best effort delivery
			m.logger.LogAttrs(ctx, slog.LevelError, "Failed to deliver notification",
				slog.String("notification_id", notif.ID),
				logger.UserID(notif.UserID),
				slog.Int("deliverer_index", i),
				logger.Error(err),
			)
			continue
		}
	}
	return nil
}

// DeliverBatch sends multiple notifications through all channels.
func (m *MultiDeliverer) DeliverBatch(ctx context.Context, notifs []Notification) error {
	for i, d := range m.deliverers {
		if err := d.DeliverBatch(ctx, notifs); err != nil {
			// Log but don't fail - best effort delivery
			m.logger.LogAttrs(ctx, slog.LevelError, "Failed to deliver notification batch",
				slog.Int("notification_count", len(notifs)),
				slog.Int("deliverer_index", i),
				logger.Error(err),
			)
			continue
		}
	}
	return nil
}

// NoOpDeliverer is a deliverer that does nothing.
// Useful for testing or when real-time delivery is not needed.
type NoOpDeliverer struct{}

// Deliver does nothing and returns nil.
func (n *NoOpDeliverer) Deliver(ctx context.Context, notif Notification) error {
	return nil
}

// DeliverBatch does nothing and returns nil.
func (n *NoOpDeliverer) DeliverBatch(ctx context.Context, notifs []Notification) error {
	return nil
}