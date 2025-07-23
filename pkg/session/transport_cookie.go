package session

import (
	"net/http"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/cookie"
)

// CookieTransport implements Transport using cookies
type CookieTransport struct {
	cookieMgr  *cookie.Manager
	cookieName string
	options    []cookie.Option
}

// NewCookieTransport creates a new cookie-based transport
func NewCookieTransport(cookieMgr *cookie.Manager, cookieName string, opts ...cookie.Option) *CookieTransport {
	return &CookieTransport{
		cookieMgr:  cookieMgr,
		cookieName: cookieName,
		options:    opts,
	}
}

// GetToken extracts the session token from the cookie
func (t *CookieTransport) GetToken(r *http.Request) (string, error) {
	token, err := t.cookieMgr.GetEncrypted(r, t.cookieName)
	if err != nil {
		return "", ErrSessionNotFound
	}
	return token, nil
}

// SetToken stores the session token in a cookie
func (t *CookieTransport) SetToken(w http.ResponseWriter, token string, ttl time.Duration) error {
	opts := []cookie.Option{
		cookie.WithMaxAge(int(ttl.Seconds())),
		cookie.WithPath("/"),
		cookie.WithHTTPOnly(true),
	}
	opts = append(opts, t.options...)

	return t.cookieMgr.SetEncrypted(w, t.cookieName, token, opts...)
}

// ClearToken removes the session cookie
func (t *CookieTransport) ClearToken(w http.ResponseWriter) error {
	t.cookieMgr.Delete(w, t.cookieName)
	return nil
}
