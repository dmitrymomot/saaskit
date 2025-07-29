package session_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/session"
)

func TestTenantResolver(t *testing.T) {
	t.Parallel()

	t.Run("extracts tenant from session", func(t *testing.T) {
		t.Parallel()

		getSession := func(r *http.Request) (*session.Session, error) {
			s := &session.Session{
				ID:   uuid.New(),
				Data: map[string]any{"tenant_id": "session-tenant"},
			}
			return s, nil
		}

		resolver := session.NewTenantResolver(getSession)
		req := httptest.NewRequest("GET", "/test", nil)

		id, err := resolver.Resolve(req)
		require.NoError(t, err)
		assert.Equal(t, "session-tenant", id)
	})

	t.Run("returns empty for missing tenant in session", func(t *testing.T) {
		t.Parallel()

		getSession := func(r *http.Request) (*session.Session, error) {
			s := &session.Session{
				ID:   uuid.New(),
				Data: map[string]any{},
			}
			return s, nil
		}

		resolver := session.NewTenantResolver(getSession)
		req := httptest.NewRequest("GET", "/test", nil)

		id, err := resolver.Resolve(req)
		require.NoError(t, err)
		assert.Empty(t, id)
	})

	t.Run("returns error when GetSession is nil", func(t *testing.T) {
		t.Parallel()

		resolver := session.NewTenantResolver(nil)
		req := httptest.NewRequest("GET", "/test", nil)

		_, err := resolver.Resolve(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GetSession function not configured")
	})

	t.Run("propagates session retrieval error", func(t *testing.T) {
		t.Parallel()

		sessionErr := errors.New("session error")
		getSession := func(r *http.Request) (*session.Session, error) {
			return nil, sessionErr
		}

		resolver := session.NewTenantResolver(getSession)
		req := httptest.NewRequest("GET", "/test", nil)

		_, err := resolver.Resolve(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session resolver")
	})

	t.Run("handles nil session", func(t *testing.T) {
		t.Parallel()

		getSession := func(r *http.Request) (*session.Session, error) {
			return nil, nil
		}

		resolver := session.NewTenantResolver(getSession)
		req := httptest.NewRequest("GET", "/test", nil)

		id, err := resolver.Resolve(req)
		require.NoError(t, err)
		assert.Empty(t, id)
	})

	t.Run("returns tenant IDs without validation", func(t *testing.T) {
		t.Parallel()

		testCases := []string{
			"tenant123",
			"tenant-123",
			"tenant_123",
			"a1b2c3d4-e5f6-7890-1234-567890abcdef", // UUID format
			"invalid!@#$%",                         // Invalid characters - but resolver doesn't validate
		}

		for _, tenantID := range testCases {
			getSession := func(r *http.Request) (*session.Session, error) {
				s := &session.Session{
					ID:   uuid.New(),
					Data: map[string]any{"tenant_id": tenantID},
				}
				return s, nil
			}

			resolver := session.NewTenantResolver(getSession)
			req := httptest.NewRequest("GET", "/test", nil)

			id, err := resolver.Resolve(req)
			require.NoError(t, err, "tenant ID %s should be returned without validation", tenantID)
			assert.Equal(t, tenantID, id)
		}
	})

	t.Run("handles non-string values in session data", func(t *testing.T) {
		t.Parallel()

		getSession := func(r *http.Request) (*session.Session, error) {
			s := &session.Session{
				ID:   uuid.New(),
				Data: map[string]any{"tenant_id": 12345}, // numeric value
			}
			return s, nil
		}

		resolver := session.NewTenantResolver(getSession)
		req := httptest.NewRequest("GET", "/test", nil)

		id, err := resolver.Resolve(req)
		require.NoError(t, err)
		assert.Empty(t, id) // GetString returns empty for non-string values
	})
}
