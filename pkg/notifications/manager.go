package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/dmitrymomot/saaskit/pkg/logger"
)

// Manager orchestrates notification storage and delivery.
type Manager struct {
	storage   Storage
	deliverer Deliverer
	logger    *slog.Logger
}

// ManagerOption configures a Manager.
type ManagerOption func(*Manager)

// WithManagerLogger sets the logger for the Manager.
func WithManagerLogger(logger *slog.Logger) ManagerOption {
	return func(m *Manager) {
		m.logger = logger
	}
}

// NewManager creates a new notification manager.
func NewManager(storage Storage, deliverer Deliverer, opts ...ManagerOption) *Manager {
	if deliverer == nil {
		deliverer = &NoOpDeliverer{}
	}

	m := &Manager{
		storage:   storage,
		deliverer: deliverer,
		logger:    slog.Default(),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

func (m *Manager) Send(ctx context.Context, notif Notification) error {
	// Generate ID if not provided
	if notif.ID == "" {
		notif.ID = uuid.New().String()
	}

	// Set creation time if not provided
	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = time.Now()
	}

	// Store first to ensure persistence even if real-time delivery fails
	if err := m.storage.Create(ctx, notif); err != nil {
		return fmt.Errorf("failed to store notification: %w", err)
	}

	// Then attempt real-time delivery (best effort pattern)
	if m.deliverer != nil {
		// Log delivery failure but don't fail the entire operation
		// Notification is persisted and available for retrieval/retry
		if err := m.deliverer.Deliver(ctx, notif); err != nil {
			m.logger.LogAttrs(ctx, slog.LevelWarn, "Failed to deliver notification, but it was stored successfully",
				slog.String("notification_id", notif.ID),
				logger.UserID(notif.UserID),
				logger.Error(err),
			)
		}
	}

	return nil
}

func (m *Manager) SendToUsers(ctx context.Context, userIDs []string, template Notification) error {
	notifications := make([]Notification, 0, len(userIDs))

	for _, userID := range userIDs {
		notif := template
		notif.ID = uuid.New().String()
		notif.UserID = userID
		notif.CreatedAt = time.Now()

		// Store notification
		if err := m.storage.Create(ctx, notif); err != nil {
			return fmt.Errorf("failed to store notification for user %s: %w", userID, err)
		}

		notifications = append(notifications, notif)
	}

	// Attempt batch delivery after all notifications are persisted
	// Uses best-effort pattern - failures don't affect stored notifications
	if m.deliverer != nil {
		if err := m.deliverer.DeliverBatch(ctx, notifications); err != nil {
			m.logger.LogAttrs(ctx, slog.LevelWarn, "Failed to deliver notification batch, but they were stored successfully",
				slog.Int("notification_count", len(notifications)),
				logger.Error(err),
			)
		}
	}

	return nil
}

func (m *Manager) SendBatch(ctx context.Context, notifications []Notification) error {
	for i := range notifications {
		// Generate ID if not provided
		if notifications[i].ID == "" {
			notifications[i].ID = uuid.New().String()
		}

		// Set creation time if not provided
		if notifications[i].CreatedAt.IsZero() {
			notifications[i].CreatedAt = time.Now()
		}

		// Store notification
		if err := m.storage.Create(ctx, notifications[i]); err != nil {
			return fmt.Errorf("failed to store notification %s: %w", notifications[i].ID, err)
		}
	}

	// Attempt batch delivery after all notifications are persisted
	if m.deliverer != nil {
		if err := m.deliverer.DeliverBatch(ctx, notifications); err != nil {
			m.logger.LogAttrs(ctx, slog.LevelWarn, "Failed to deliver notification batch, but they were stored successfully",
				slog.Int("notification_count", len(notifications)),
				logger.Error(err),
			)
		}
	}

	return nil
}

func (m *Manager) Get(ctx context.Context, userID, notifID string) (*Notification, error) {
	return m.storage.Get(ctx, userID, notifID)
}

func (m *Manager) List(ctx context.Context, userID string, opts ListOptions) ([]Notification, error) {
	return m.storage.List(ctx, userID, opts)
}

func (m *Manager) MarkRead(ctx context.Context, userID string, notifIDs ...string) error {
	return m.storage.MarkRead(ctx, userID, notifIDs...)
}

// MarkAllRead marks all notifications as read for a user.
func (m *Manager) MarkAllRead(ctx context.Context, userID string) error {
	// Retrieve all unread notifications to get their IDs for bulk update
	notifications, err := m.storage.List(ctx, userID, ListOptions{
		OnlyUnread: true,
	})
	if err != nil {
		return err
	}

	// Extract notification IDs for batch marking
	ids := make([]string, len(notifications))
	for i, n := range notifications {
		ids[i] = n.ID
	}

	// Mark all as read
	if len(ids) > 0 {
		return m.storage.MarkRead(ctx, userID, ids...)
	}

	return nil
}

func (m *Manager) Delete(ctx context.Context, userID string, notifIDs ...string) error {
	return m.storage.Delete(ctx, userID, notifIDs...)
}

func (m *Manager) CountUnread(ctx context.Context, userID string) (int, error) {
	return m.storage.CountUnread(ctx, userID)
}

// Storage returns the underlying notification storage.
func (m *Manager) Storage() Storage {
	return m.storage
}

// Deliverer returns the underlying notification deliverer.
func (m *Manager) Deliverer() Deliverer {
	return m.deliverer
}
