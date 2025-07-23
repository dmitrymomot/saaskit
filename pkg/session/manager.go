package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/dmitrymomot/saaskit/pkg/cookie"
)

// FingerprintFunc generates a device fingerprint from the request
type FingerprintFunc func(r *http.Request) string

// Manager handles session operations
type Manager struct {
	store           Store
	transport       Transport
	config          Config
	fingerprintFunc FingerprintFunc
	cookieManager   *cookie.Manager
	cookieOptions   []cookie.Option
}

// New creates a new session manager with the given options
func New(opts ...Option) *Manager {
	m := &Manager{
		config: DefaultConfig(),
	}

	for _, opt := range opts {
		opt(m)
	}

	if m.store == nil {
		m.store = NewMemoryStore(m.config.CleanupInterval)
	}

	if m.transport == nil {
		if m.cookieManager == nil {
			// This should be configured by the user with proper secrets
			panic("session: cookie manager is required when using default cookie transport")
		}
		m.transport = NewCookieTransport(m.cookieManager, m.config.CookieName, m.cookieOptions...)
	}

	return m
}

// Ensure creates or retrieves a session
func (m *Manager) Ensure(ctx context.Context, w http.ResponseWriter, r *http.Request) (*Session, error) {
	session, err := m.Get(ctx, r)
	if err == nil {
		if err := m.validate(session, r); err == nil {
			if m.shouldUpdateActivity(session) {
				go m.updateActivity(ctx, session)
			}
			return session, nil
		}
		_ = m.transport.ClearToken(w)
	}

	session, err = m.createSession(ctx, nil, r)
	if err != nil {
		return nil, err
	}

	idle, _ := m.config.GetTimeouts(false)
	if err := m.transport.SetToken(w, session.Token, idle); err != nil {
		_ = m.store.Delete(ctx, session.Token)
		return nil, err
	}

	return session, nil
}

// Get retrieves an existing session
func (m *Manager) Get(ctx context.Context, r *http.Request) (*Session, error) {
	token, err := m.transport.GetToken(r)
	if err != nil {
		return nil, err
	}

	session, err := m.store.Get(ctx, token)
	if err != nil {
		return nil, err
	}

	if err := m.validate(session, r); err != nil {
		return nil, err
	}

	return session, nil
}

// Authenticate upgrades an anonymous session to authenticated
func (m *Manager) Authenticate(ctx context.Context, w http.ResponseWriter, r *http.Request, userID uuid.UUID) error {
	session, err := m.Get(ctx, r)
	if err != nil {
		// Create new authenticated session
		session, err = m.createSession(ctx, &userID, r)
		if err != nil {
			return err
		}
	} else {
		session.UserID = &userID

		newToken, err := generateToken()
		if err != nil {
			return err
		}

		_ = m.store.Delete(ctx, session.Token)

		session.Token = newToken
		idle, max := m.config.GetTimeouts(true)
		session.ExpiresAt = m.calculateExpiry(session.CreatedAt, time.Now(), idle, max)
		session.Touch()

		if err := m.store.Create(ctx, session); err != nil {
			return err
		}
	}

	idle, _ := m.config.GetTimeouts(true)
	return m.transport.SetToken(w, session.Token, idle)
}

// Destroy deletes the session
func (m *Manager) Destroy(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	token, err := m.transport.GetToken(r)
	if err == nil && token != "" {
		_ = m.store.Delete(ctx, token)
	}

	return m.transport.ClearToken(w)
}

// Set stores a value in the session
func (m *Manager) Set(ctx context.Context, w http.ResponseWriter, r *http.Request, key string, value any) error {
	session, err := m.Ensure(ctx, w, r)
	if err != nil {
		return err
	}

	session.Set(key, value)
	return m.store.Update(ctx, session)
}

// Get retrieves a value from the session
func (m *Manager) GetValue(ctx context.Context, r *http.Request, key string) (any, bool) {
	session, err := m.Get(ctx, r)
	if err != nil {
		return nil, false
	}

	return session.Get(key)
}

// Refresh updates the session expiry
func (m *Manager) Refresh(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	session, err := m.Get(ctx, r)
	if err != nil {
		return err
	}

	idle, max := m.config.GetTimeouts(session.IsAuthenticated())
	session.ExpiresAt = m.calculateExpiry(session.CreatedAt, time.Now(), idle, max)
	session.Touch()

	if err := m.store.Update(ctx, session); err != nil {
		return err
	}

	return m.transport.SetToken(w, session.Token, idle)
}

// createSession creates a new session
func (m *Manager) createSession(ctx context.Context, userID *uuid.UUID, r *http.Request) (*Session, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	idle, max := m.config.GetTimeouts(userID != nil)
	now := time.Now()

	var fingerprint string
	if m.fingerprintFunc != nil {
		fingerprint = m.fingerprintFunc(r)
	}

	session := NewSession(token, userID, fingerprint, m.calculateExpiry(now, now, idle, max).Sub(now))

	if err := m.store.Create(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// validate checks if the session is valid
func (m *Manager) validate(session *Session, r *http.Request) error {
	if session.IsExpired() {
		return ErrSessionExpired
	}

	if m.fingerprintFunc != nil && session.Fingerprint != "" {
		currentFingerprint := m.fingerprintFunc(r)
		if !session.ValidateFingerprint(currentFingerprint) {
			return ErrInvalidSession
		}
	}

	return nil
}

// shouldUpdateActivity checks if activity should be updated
func (m *Manager) shouldUpdateActivity(session *Session) bool {
	return time.Since(session.LastActivityAt) >= m.config.ActivityUpdateThreshold
}

// updateActivity updates the session's last activity time
func (m *Manager) updateActivity(ctx context.Context, session *Session) {
	now := time.Now()
	idle, max := m.config.GetTimeouts(session.IsAuthenticated())
	newExpiry := m.calculateExpiry(session.CreatedAt, now, idle, max)

	session.ExpiresAt = newExpiry
	session.LastActivityAt = now

	_ = m.store.UpdateActivity(ctx, session.Token, now)
}

// calculateExpiry returns the next expiry time (min of idle and max lifetime)
func (m *Manager) calculateExpiry(createdAt, now time.Time, idle, max time.Duration) time.Time {
	idleExpiry := now.Add(idle)
	maxExpiry := createdAt.Add(max)

	if maxExpiry.Before(idleExpiry) {
		return maxExpiry
	}
	return idleExpiry
}

// generateToken creates a cryptographically secure token
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", errors.Join(ErrTokenGeneration, err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
