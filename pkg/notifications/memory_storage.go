package notifications

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrNotificationNotFound is returned when a notification is not found.
	ErrNotificationNotFound = errors.New("notification not found")
)

// MemoryStorage is an in-memory implementation of the Storage interface.
// Suitable for development and testing.
type MemoryStorage struct {
	notifications map[string][]Notification // userID -> notifications
	mu            sync.RWMutex
}

// NewMemoryStorage creates a new in-memory notification storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		notifications: make(map[string][]Notification),
	}
}

func (s *MemoryStorage) Create(ctx context.Context, notif Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if notif.ID == "" {
		return errors.New("notification ID is required")
	}
	if notif.UserID == "" {
		return errors.New("user ID is required")
	}

	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = time.Now()
	}

	s.notifications[notif.UserID] = append(s.notifications[notif.UserID], notif)
	return nil
}

func (s *MemoryStorage) Get(ctx context.Context, userID, notifID string) (*Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	notifications, exists := s.notifications[userID]
	if !exists {
		return nil, ErrNotificationNotFound
	}

	for _, n := range notifications {
		if n.ID == notifID {
			// Return a copy to prevent external mutation of stored data
			notif := n
			return &notif, nil
		}
	}

	return nil, ErrNotificationNotFound
}

func (s *MemoryStorage) List(ctx context.Context, userID string, opts ListOptions) ([]Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	notifications, exists := s.notifications[userID]
	if !exists {
		return []Notification{}, nil
	}

	// Apply filters
	var filtered []Notification
	for _, n := range notifications {
		// Skip expired notifications
		if n.IsExpired() {
			continue
		}

		// Filter by unread status
		if opts.OnlyUnread && n.Read {
			continue
		}

		// Filter by types
		if len(opts.Types) > 0 {
			found := false
			for _, t := range opts.Types {
				if n.Type == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by time
		if opts.Since != nil && n.CreatedAt.Before(*opts.Since) {
			continue
		}

		filtered = append(filtered, n)
	}

	// Sort by created time (newest first) using bubble sort
	// O(n²) complexity acceptable for in-memory storage with small datasets
	// For production use, consider replacing MemoryStorage with database-backed implementation
	for i := 0; i < len(filtered)-1; i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[i].CreatedAt.Before(filtered[j].CreatedAt) {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	// Apply pagination
	start := opts.Offset
	if start > len(filtered) {
		return []Notification{}, nil
	}

	end := start + opts.Limit
	if opts.Limit == 0 || end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

func (s *MemoryStorage) MarkRead(ctx context.Context, userID string, notifIDs ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	notifications, exists := s.notifications[userID]
	if !exists {
		return nil
	}

	// Create lookup map for O(1) ID checking instead of O(n) slice iteration
	idMap := make(map[string]bool)
	for _, id := range notifIDs {
		idMap[id] = true
	}

	for i := range notifications {
		if idMap[notifications[i].ID] {
			notifications[i].MarkAsRead()
		}
	}

	s.notifications[userID] = notifications
	return nil
}

func (s *MemoryStorage) Delete(ctx context.Context, userID string, notifIDs ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	notifications, exists := s.notifications[userID]
	if !exists {
		return nil
	}

	// Create lookup map for O(1) ID checking instead of O(n) slice iteration
	idMap := make(map[string]bool)
	for _, id := range notifIDs {
		idMap[id] = true
	}

	var filtered []Notification
	for _, n := range notifications {
		if !idMap[n.ID] {
			filtered = append(filtered, n)
		}
	}

	s.notifications[userID] = filtered
	return nil
}

func (s *MemoryStorage) CountUnread(ctx context.Context, userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	notifications, exists := s.notifications[userID]
	if !exists {
		return 0, nil
	}

	count := 0
	for _, n := range notifications {
		if !n.Read && !n.IsExpired() {
			count++
		}
	}

	return count, nil
}
