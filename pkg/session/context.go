package session

import "context"

type sessionContextKey struct{}

// WithSession adds a session to the context
func WithSession(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, session)
}

// FromContext retrieves a session from the context
func FromContext(ctx context.Context) (*Session, bool) {
	session, ok := ctx.Value(sessionContextKey{}).(*Session)
	return session, ok
}

// MustFromContext retrieves a session from the context or panics
func MustFromContext(ctx context.Context) *Session {
	session, ok := FromContext(ctx)
	if !ok {
		panic("session: not found in context")
	}
	return session
}

// UserIDFromContext retrieves the user ID from the session in context
func UserIDFromContext(ctx context.Context) (string, bool) {
	session, ok := FromContext(ctx)
	if !ok || !session.IsAuthenticated() {
		return "", false
	}
	return session.UserID.String(), true
}
