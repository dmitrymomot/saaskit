package session

import (
	"net/http"
	"time"
)

// Transport defines how session tokens are transmitted between client and server
type Transport interface {
	// GetToken extracts the session token from the request
	GetToken(r *http.Request) (string, error)

	// SetToken sends the session token in the response
	SetToken(w http.ResponseWriter, token string, ttl time.Duration) error

	// ClearToken removes the session token from the response
	ClearToken(w http.ResponseWriter) error
}
