package session

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryStore implements Store interface using in-memory storage
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	ticker   *time.Ticker
	done     chan struct{}
}

// NewMemoryStore creates a new in-memory session store
func NewMemoryStore(cleanupInterval time.Duration) *MemoryStore {
	store := &MemoryStore{
		sessions: make(map[string]*Session),
		done:     make(chan struct{}),
	}

	if cleanupInterval > 0 {
		store.ticker = time.NewTicker(cleanupInterval)
		go store.cleanupLoop()
	}

	return store
}

// Create stores a new session
func (m *MemoryStore) Create(ctx context.Context, session *Session) error {
	if session == nil || session.Token == "" {
		return ErrInvalidSession
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	sessionCopy := *session
	if session.Data != nil {
		sessionCopy.Data = make(map[string]any, len(session.Data))
		maps.Copy(sessionCopy.Data, session.Data)
	}

	m.sessions[session.Token] = &sessionCopy
	return nil
}

// Get retrieves a session by token
func (m *MemoryStore) Get(ctx context.Context, token string) (*Session, error) {
	m.mu.RLock()
	session, exists := m.sessions[token]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrSessionNotFound
	}

	if session.IsExpired() {
		m.mu.Lock()
		delete(m.sessions, token)
		m.mu.Unlock()
		return nil, ErrSessionExpired
	}

	sessionCopy := *session
	if session.Data != nil {
		sessionCopy.Data = make(map[string]any, len(session.Data))
		maps.Copy(sessionCopy.Data, session.Data)
	}

	return &sessionCopy, nil
}

// Update updates an existing session
func (m *MemoryStore) Update(ctx context.Context, session *Session) error {
	if session == nil || session.Token == "" {
		return ErrInvalidSession
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[session.Token]; !exists {
		return ErrSessionNotFound
	}

	sessionCopy := *session
	if session.Data != nil {
		sessionCopy.Data = make(map[string]any, len(session.Data))
		maps.Copy(sessionCopy.Data, session.Data)
	}

	m.sessions[session.Token] = &sessionCopy
	return nil
}

// UpdateActivity updates only the last activity time
func (m *MemoryStore) UpdateActivity(ctx context.Context, token string, lastActivity time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[token]
	if !exists {
		return ErrSessionNotFound
	}

	session.LastActivityAt = lastActivity
	return nil
}

// Delete removes a session by token
func (m *MemoryStore) Delete(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, token)
	return nil
}

// DeleteExpired removes all expired sessions
func (m *MemoryStore) DeleteExpired(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for token, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			delete(m.sessions, token)
		}
	}

	return nil
}

// DeleteByUserID removes all sessions for a specific user
func (m *MemoryStore) DeleteByUserID(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for token, session := range m.sessions {
		if session.UserID != nil && *session.UserID == uid {
			delete(m.sessions, token)
		}
	}

	return nil
}

// Close stops the cleanup goroutine
func (m *MemoryStore) Close() error {
	if m.ticker != nil {
		m.ticker.Stop()
		close(m.done)
	}
	return nil
}

// cleanupLoop runs periodic cleanup of expired sessions
func (m *MemoryStore) cleanupLoop() {
	for {
		select {
		case <-m.ticker.C:
			_ = m.DeleteExpired(context.Background())
		case <-m.done:
			return
		}
	}
}

// Stats returns memory store statistics
func (m *MemoryStore) Stats() (total, authenticated, anonymous int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total = len(m.sessions)
	for _, session := range m.sessions {
		if session.IsAuthenticated() {
			authenticated++
		} else {
			anonymous++
		}
	}
	return
}
