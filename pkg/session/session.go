package session

import (
	"time"

	"github.com/google/uuid"
)

// Session represents a user session with associated data
type Session struct {
	ID             uuid.UUID      `json:"id"`
	Token          string         `json:"token"`
	UserID         *uuid.UUID     `json:"user_id,omitempty"`
	Fingerprint    string         `json:"fingerprint,omitempty"`
	Data           map[string]any `json:"data,omitempty"`
	ExpiresAt      time.Time      `json:"expires_at"`
	LastActivityAt time.Time      `json:"last_activity_at"`
	CreatedAt      time.Time      `json:"created_at"`
}

// NewSession creates a new session with the given parameters
func NewSession(token string, userID *uuid.UUID, fingerprint string, ttl time.Duration) *Session {
	now := time.Now()
	return &Session{
		ID:             uuid.New(),
		Token:          token,
		UserID:         userID,
		Fingerprint:    fingerprint,
		Data:           make(map[string]any),
		ExpiresAt:      now.Add(ttl),
		LastActivityAt: now,
		CreatedAt:      now,
	}
}

// IsAuthenticated returns true if the session has a user ID
func (s *Session) IsAuthenticated() bool {
	return s != nil && s.UserID != nil
}

// IsExpired returns true if the session has expired
func (s *Session) IsExpired() bool {
	return s != nil && time.Now().After(s.ExpiresAt)
}

// Get retrieves a value from session data
func (s *Session) Get(key string) (any, bool) {
	if s == nil || s.Data == nil {
		return nil, false
	}
	val, ok := s.Data[key]
	return val, ok
}

// GetString retrieves a string value from session data
func (s *Session) GetString(key string) (string, bool) {
	val, ok := s.Get(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetInt retrieves an int value from session data
func (s *Session) GetInt(key string) (int, bool) {
	val, ok := s.Get(key)
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// GetBool retrieves a bool value from session data
func (s *Session) GetBool(key string) (bool, bool) {
	val, ok := s.Get(key)
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

// Set stores a value in session data
func (s *Session) Set(key string, value any) {
	if s == nil {
		return
	}
	if s.Data == nil {
		s.Data = make(map[string]any)
	}
	s.Data[key] = value
}

// Delete removes a value from session data
func (s *Session) Delete(key string) {
	if s == nil || s.Data == nil {
		return
	}
	delete(s.Data, key)
}

// Clear removes all data from the session
func (s *Session) Clear() {
	if s == nil {
		return
	}
	s.Data = make(map[string]any)
}

// Touch updates the last activity time
func (s *Session) Touch() {
	if s == nil {
		return
	}
	s.LastActivityAt = time.Now()
}

// ValidateFingerprint checks if the provided fingerprint matches the session's fingerprint
func (s *Session) ValidateFingerprint(fingerprint string) bool {
	if s == nil || s.Fingerprint == "" {
		return true
	}
	return constantTimeCompare(s.Fingerprint, fingerprint)
}

// constantTimeCompare performs a constant-time string comparison
func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
