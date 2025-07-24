package session

import (
	"net/http"
	"time"
)

// CompositeTransport tries multiple transports in order
type CompositeTransport struct {
	transports []Transport
}

// NewCompositeTransport creates a new composite transport that tries multiple transports
func NewCompositeTransport(transports ...Transport) *CompositeTransport {
	return &CompositeTransport{
		transports: transports,
	}
}

// GetToken tries to extract the session token from each transport in order
func (t *CompositeTransport) GetToken(r *http.Request) (string, error) {
	for _, transport := range t.transports {
		token, err := transport.GetToken(r)
		if err == nil && token != "" {
			return token, nil
		}
	}
	return "", ErrSessionNotFound
}

// SetToken sends the session token using all configured transports
func (t *CompositeTransport) SetToken(w http.ResponseWriter, token string, ttl time.Duration) error {
	var lastErr error
	for _, transport := range t.transports {
		if err := transport.SetToken(w, token, ttl); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// ClearToken removes the session token from all configured transports
func (t *CompositeTransport) ClearToken(w http.ResponseWriter) error {
	var lastErr error
	for _, transport := range t.transports {
		if err := transport.ClearToken(w); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
