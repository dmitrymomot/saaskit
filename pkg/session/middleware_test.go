package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/session"
)

func TestMiddleware(t *testing.T) {
	manager := setupManager(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := session.FromContext(r.Context())
		if ok {
			w.Header().Set("X-Session-ID", sess.ID.String())
			w.Header().Set("X-Session-Auth", "false")
			if sess.IsAuthenticated() {
				w.Header().Set("X-Session-Auth", "true")
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := manager.Middleware(handler)

	t.Run("adds session to context", func(t *testing.T) {
		// Create session first
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest("GET", "/", nil)
		sess1, err := manager.Ensure(r1.Context(), w1, r1)
		require.NoError(t, err)

		// Request with session cookie
		r2 := httptest.NewRequest("GET", "/", nil)
		for _, c := range w1.Result().Cookies() {
			r2.AddCookie(c)
		}
		w2 := httptest.NewRecorder()

		middleware.ServeHTTP(w2, r2)

		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Equal(t, sess1.ID.String(), w2.Header().Get("X-Session-ID"))
		assert.Equal(t, "false", w2.Header().Get("X-Session-Auth"))
	})

	t.Run("continues without session", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("X-Session-ID"))
		assert.Empty(t, w.Header().Get("X-Session-Auth"))
	})
}

func TestRequireAuth(t *testing.T) {
	manager := setupManager(t)
	userID := uuid.New()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := session.MustFromContext(r.Context())
		w.Header().Set("X-User-ID", sess.UserID.String())
		w.WriteHeader(http.StatusOK)
	})

	middleware := manager.RequireAuth(handler)

	t.Run("allows authenticated session", func(t *testing.T) {
		// Create authenticated session
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest("GET", "/", nil)
		err := manager.Authenticate(r1.Context(), w1, r1, userID)
		require.NoError(t, err)

		// Request with auth session
		r2 := httptest.NewRequest("GET", "/protected", nil)
		for _, c := range w1.Result().Cookies() {
			r2.AddCookie(c)
		}
		w2 := httptest.NewRecorder()

		middleware.ServeHTTP(w2, r2)

		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Equal(t, userID.String(), w2.Header().Get("X-User-ID"))
	})

	t.Run("blocks anonymous session", func(t *testing.T) {
		// Create anonymous session
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest("GET", "/", nil)
		_, err := manager.Ensure(r1.Context(), w1, r1)
		require.NoError(t, err)

		// Request with anon session
		r2 := httptest.NewRequest("GET", "/protected", nil)
		for _, c := range w1.Result().Cookies() {
			r2.AddCookie(c)
		}
		w2 := httptest.NewRecorder()

		middleware.ServeHTTP(w2, r2)

		assert.Equal(t, http.StatusUnauthorized, w2.Code)
	})

	t.Run("blocks no session", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestEnsureSession(t *testing.T) {
	manager := setupManager(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := session.MustFromContext(r.Context())
		w.Header().Set("X-Session-ID", sess.ID.String())
		w.WriteHeader(http.StatusOK)
	})

	middleware := manager.EnsureSession(handler)

	t.Run("creates session if missing", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("X-Session-ID"))

		// Check cookie was set
		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 1)
	})

	t.Run("uses existing session", func(t *testing.T) {
		// Create session first
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest("GET", "/", nil)
		sess1, err := manager.Ensure(r1.Context(), w1, r1)
		require.NoError(t, err)

		// Request with session
		r2 := httptest.NewRequest("GET", "/", nil)
		for _, c := range w1.Result().Cookies() {
			r2.AddCookie(c)
		}
		w2 := httptest.NewRecorder()

		middleware.ServeHTTP(w2, r2)

		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Equal(t, sess1.ID.String(), w2.Header().Get("X-Session-ID"))
	})
}

func TestContext(t *testing.T) {
	t.Run("WithSession and FromContext", func(t *testing.T) {
		ctx := httptest.NewRequest("GET", "/", nil).Context()
		sess := session.NewSession("token", nil, "", 1*time.Hour)

		// Add to context
		ctx = session.WithSession(ctx, sess)

		// Retrieve from context
		retrieved, ok := session.FromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, sess.ID, retrieved.ID)
	})

	t.Run("FromContext with no session", func(t *testing.T) {
		ctx := httptest.NewRequest("GET", "/", nil).Context()

		sess, ok := session.FromContext(ctx)
		assert.False(t, ok)
		assert.Nil(t, sess)
	})

	t.Run("MustFromContext", func(t *testing.T) {
		ctx := httptest.NewRequest("GET", "/", nil).Context()
		sess := session.NewSession("token", nil, "", 1*time.Hour)
		ctx = session.WithSession(ctx, sess)

		retrieved := session.MustFromContext(ctx)
		assert.Equal(t, sess.ID, retrieved.ID)
	})

	t.Run("MustFromContext panics", func(t *testing.T) {
		ctx := httptest.NewRequest("GET", "/", nil).Context()

		assert.Panics(t, func() {
			session.MustFromContext(ctx)
		})
	})

	t.Run("UserIDFromContext", func(t *testing.T) {
		ctx := httptest.NewRequest("GET", "/", nil).Context()

		// No session
		userID, ok := session.UserIDFromContext(ctx)
		assert.False(t, ok)
		assert.Empty(t, userID)

		// Anonymous session
		anonSess := session.NewSession("token", nil, "", 1*time.Hour)
		ctx = session.WithSession(ctx, anonSess)
		userID, ok = session.UserIDFromContext(ctx)
		assert.False(t, ok)
		assert.Empty(t, userID)

		// Authenticated session
		uid := uuid.New()
		authSess := session.NewSession("token", &uid, "", 1*time.Hour)
		ctx = session.WithSession(ctx, authSess)
		userID, ok = session.UserIDFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, uid.String(), userID)
	})
}
