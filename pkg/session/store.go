package session

import (
	"context"
	"time"
)

// Store defines the interface for session persistence
type Store interface {
	// Create stores a new session
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by token
	Get(ctx context.Context, token string) (*Session, error)

	// Update updates an existing session
	Update(ctx context.Context, session *Session) error

	// UpdateActivity updates only the last activity time
	UpdateActivity(ctx context.Context, token string, lastActivity time.Time) error

	// Delete removes a session by token
	Delete(ctx context.Context, token string) error

	// DeleteExpired removes all expired sessions
	DeleteExpired(ctx context.Context) error
}

// StoreWithCleanup is an optional interface for stores that support user session cleanup
type StoreWithCleanup interface {
	Store
	// DeleteByUserID removes all sessions for a specific user
	DeleteByUserID(ctx context.Context, userID string) error
}
