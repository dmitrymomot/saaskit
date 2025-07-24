package session_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/session"
)

func TestNewSession(t *testing.T) {
	token := "test-token"
	userID := uuid.New()
	fingerprint := "test-fingerprint"
	ttl := 1 * time.Hour

	sess := session.NewSession(token, &userID, fingerprint, ttl)

	assert.NotNil(t, sess)
	assert.Equal(t, token, sess.Token)
	assert.Equal(t, &userID, sess.UserID)
	assert.Equal(t, fingerprint, sess.Fingerprint)
	assert.NotNil(t, sess.Data)
	assert.True(t, sess.ExpiresAt.After(time.Now()))
	assert.WithinDuration(t, time.Now(), sess.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, time.Now(), sess.LastActivityAt, 1*time.Second)
}

func TestSession_IsAuthenticated(t *testing.T) {
	tests := []struct {
		name     string
		session  *session.Session
		expected bool
	}{
		{
			name:     "nil session",
			session:  nil,
			expected: false,
		},
		{
			name:     "anonymous session",
			session:  session.NewSession("token", nil, "", 1*time.Hour),
			expected: false,
		},
		{
			name: "authenticated session",
			session: func() *session.Session {
				userID := uuid.New()
				return session.NewSession("token", &userID, "", 1*time.Hour)
			}(),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.session.IsAuthenticated())
		})
	}
}

func TestSession_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		session  *session.Session
		expected bool
	}{
		{
			name:     "nil session",
			session:  nil,
			expected: false,
		},
		{
			name:     "valid session",
			session:  session.NewSession("token", nil, "", 1*time.Hour),
			expected: false,
		},
		{
			name: "expired session",
			session: func() *session.Session {
				s := session.NewSession("token", nil, "", 1*time.Hour)
				s.ExpiresAt = time.Now().Add(-1 * time.Hour)
				return s
			}(),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.session.IsExpired())
		})
	}
}

func TestSession_DataOperations(t *testing.T) {
	sess := session.NewSession("token", nil, "", 1*time.Hour)

	t.Run("Set and Get", func(t *testing.T) {
		sess.Set("key1", "value1")
		sess.Set("key2", 42)
		sess.Set("key3", true)

		val, ok := sess.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", val)

		val, ok = sess.Get("key2")
		assert.True(t, ok)
		assert.Equal(t, 42, val)

		val, ok = sess.Get("key3")
		assert.True(t, ok)
		assert.Equal(t, true, val)

		val, ok = sess.Get("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("GetString", func(t *testing.T) {
		sess.Set("string", "hello")
		sess.Set("number", 123)

		str, ok := sess.GetString("string")
		assert.True(t, ok)
		assert.Equal(t, "hello", str)

		str, ok = sess.GetString("number")
		assert.False(t, ok)
		assert.Empty(t, str)

		str, ok = sess.GetString("nonexistent")
		assert.False(t, ok)
		assert.Empty(t, str)
	})

	t.Run("GetInt", func(t *testing.T) {
		sess.Set("int", 42)
		sess.Set("int64", int64(100))
		sess.Set("float64", float64(3.14))
		sess.Set("string", "not a number")

		num, ok := sess.GetInt("int")
		assert.True(t, ok)
		assert.Equal(t, 42, num)

		num, ok = sess.GetInt("int64")
		assert.True(t, ok)
		assert.Equal(t, 100, num)

		num, ok = sess.GetInt("float64")
		assert.True(t, ok)
		assert.Equal(t, 3, num)

		num, ok = sess.GetInt("string")
		assert.False(t, ok)
		assert.Equal(t, 0, num)
	})

	t.Run("GetBool", func(t *testing.T) {
		sess.Set("bool_true", true)
		sess.Set("bool_false", false)
		sess.Set("string", "not a bool")

		b, ok := sess.GetBool("bool_true")
		assert.True(t, ok)
		assert.True(t, b)

		b, ok = sess.GetBool("bool_false")
		assert.True(t, ok)
		assert.False(t, b)

		b, ok = sess.GetBool("string")
		assert.False(t, ok)
		assert.False(t, b)
	})

	t.Run("Delete", func(t *testing.T) {
		sess.Set("to_delete", "value")

		val, ok := sess.Get("to_delete")
		assert.True(t, ok)
		assert.Equal(t, "value", val)

		sess.Delete("to_delete")

		val, ok = sess.Get("to_delete")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("Clear", func(t *testing.T) {
		sess.Set("key1", "value1")
		sess.Set("key2", "value2")

		sess.Clear()

		val, ok := sess.Get("key1")
		assert.False(t, ok)
		assert.Nil(t, val)

		val, ok = sess.Get("key2")
		assert.False(t, ok)
		assert.Nil(t, val)
	})
}

func TestSession_Touch(t *testing.T) {
	sess := session.NewSession("token", nil, "", 1*time.Hour)
	originalTime := sess.LastActivityAt

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	sess.Touch()

	assert.True(t, sess.LastActivityAt.After(originalTime))
}

func TestSession_ValidateFingerprint(t *testing.T) {
	tests := []struct {
		name        string
		session     *session.Session
		fingerprint string
		expected    bool
	}{
		{
			name:        "nil session",
			session:     nil,
			fingerprint: "any",
			expected:    true,
		},
		{
			name:        "no fingerprint in session",
			session:     session.NewSession("token", nil, "", 1*time.Hour),
			fingerprint: "any",
			expected:    true,
		},
		{
			name:        "matching fingerprint",
			session:     session.NewSession("token", nil, "test-fp", 1*time.Hour),
			fingerprint: "test-fp",
			expected:    true,
		},
		{
			name:        "non-matching fingerprint",
			session:     session.NewSession("token", nil, "test-fp", 1*time.Hour),
			fingerprint: "different-fp",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.session.ValidateFingerprint(tt.fingerprint))
		})
	}
}

func TestSession_NilSafety(t *testing.T) {
	var sess *session.Session

	// These should not panic
	assert.NotPanics(t, func() {
		sess.Set("key", "value")
		sess.Get("key")
		sess.GetString("key")
		sess.GetInt("key")
		sess.GetBool("key")
		sess.Delete("key")
		sess.Clear()
		sess.Touch()
		sess.ValidateFingerprint("fp")
		sess.IsAuthenticated()
		sess.IsExpired()
	})
}

func TestConstantTimeCompare(t *testing.T) {
	sess1 := session.NewSession("token", nil, "fingerprint123", 1*time.Hour)
	sess2 := session.NewSession("token", nil, "fingerprint456", 1*time.Hour)

	// Same fingerprint should match
	assert.True(t, sess1.ValidateFingerprint("fingerprint123"))

	// Different fingerprint should not match
	assert.False(t, sess1.ValidateFingerprint("fingerprint456"))

	// Different lengths should not match
	assert.False(t, sess1.ValidateFingerprint("short"))
	assert.False(t, sess2.ValidateFingerprint("verylongfingerprint"))
}
