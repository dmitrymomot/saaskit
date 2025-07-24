package session

import (
	"net/http"
	"strings"
	"time"
)

// HeaderTransport implements Transport using HTTP headers
type HeaderTransport struct {
	headerName string
	prefix     string
}

// NewHeaderTransport creates a new header-based transport
func NewHeaderTransport(headerName string, opts ...HeaderOption) *HeaderTransport {
	t := &HeaderTransport{
		headerName: headerName,
		prefix:     "Bearer ",
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// HeaderOption is a functional option for HeaderTransport
type HeaderOption func(*HeaderTransport)

// WithHeaderPrefix sets a custom prefix for the header value
func WithHeaderPrefix(prefix string) HeaderOption {
	return func(t *HeaderTransport) {
		t.prefix = prefix
	}
}

// GetToken extracts the session token from the header
func (t *HeaderTransport) GetToken(r *http.Request) (string, error) {
	value := r.Header.Get(t.headerName)
	if value == "" {
		return "", ErrSessionNotFound
	}

	if t.prefix != "" && strings.HasPrefix(value, t.prefix) {
		value = strings.TrimPrefix(value, t.prefix)
	}

	return value, nil
}

// SetToken sends the session token in the response header
func (t *HeaderTransport) SetToken(w http.ResponseWriter, token string, ttl time.Duration) error {
	value := token
	if t.prefix != "" {
		value = t.prefix + token
	}
	w.Header().Set(t.headerName, value)

	if ttl > 0 {
		w.Header().Set(t.headerName+"-Expires", time.Now().Add(ttl).Format(time.RFC3339))
	}

	return nil
}

// ClearToken removes the session header from the response
func (t *HeaderTransport) ClearToken(w http.ResponseWriter) error {
	w.Header().Del(t.headerName)
	w.Header().Del(t.headerName + "-Expires")
	return nil
}
