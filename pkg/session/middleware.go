package session

import (
	"net/http"
)

// Middleware provides session handling for HTTP requests
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.Get(r.Context(), r)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := WithSession(r.Context(), session)

		if m.shouldUpdateActivity(session) {
			go m.updateActivity(r.Context(), session)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth is a middleware that requires an authenticated session
func (m *Manager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.Get(r.Context(), r)
		if err != nil || !session.IsAuthenticated() {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := WithSession(r.Context(), session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// EnsureSession is a middleware that ensures a session exists
func (m *Manager) EnsureSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.Ensure(r.Context(), w, r)
		if err != nil {
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}

		ctx := WithSession(r.Context(), session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
