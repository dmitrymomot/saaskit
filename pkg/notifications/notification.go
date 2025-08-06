package notifications

import (
	"time"
)

// Type represents the notification type/severity.
type Type string

const (
	TypeInfo    Type = "info"
	TypeSuccess Type = "success"
	TypeWarning Type = "warning"
	TypeError   Type = "error"
)

// Priority represents the notification priority level.
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityUrgent
)

// Action represents a call-to-action button in a notification.
type Action struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Style string `json:"style"` // primary, secondary, danger
}

// Notification is the core domain model for notifications.
type Notification struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Type      Type                   `json:"type"`
	Priority  Priority               `json:"priority"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`      // Custom payload
	Actions   []Action               `json:"actions,omitempty"`   // CTAs
	Read      bool                   `json:"read"`
	ReadAt    *time.Time             `json:"read_at,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
}

// IsExpired returns true if the notification has expired.
func (n *Notification) IsExpired() bool {
	if n.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*n.ExpiresAt)
}

// MarkAsRead marks the notification as read with the current timestamp.
func (n *Notification) MarkAsRead() {
	n.Read = true
	now := time.Now()
	n.ReadAt = &now
}