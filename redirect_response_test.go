package saaskit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit"
)

func TestRedirect(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		htmx         bool
		boosted      bool
		wantStatus   int
		wantLocation string
		wantHeader   string
	}{
		{
			name:         "regular redirect",
			url:          "/users/123",
			htmx:         false,
			boosted:      false,
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/users/123",
		},
		{
			name:       "htmx redirect",
			url:        "/users/123",
			htmx:       true,
			boosted:    false,
			wantStatus: http.StatusOK,
			wantHeader: "/users/123",
		},
		{
			name:       "htmx boosted redirect",
			url:        "/users/123",
			htmx:       true,
			boosted:    true,
			wantStatus: http.StatusOK,
			wantHeader: "/users/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			if tt.htmx {
				r.Header.Set("HX-Request", "true")
			}
			if tt.boosted {
				r.Header.Set("HX-Boosted", "true")
			}

			resp := saaskit.Redirect(tt.url)
			err := resp.Render(w, r)

			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.htmx {
				assert.Equal(t, tt.wantHeader, w.Header().Get("HX-Redirect"))
			} else {
				assert.Equal(t, tt.wantLocation, w.Header().Get("Location"))
			}
		})
	}
}

func TestRedirectWithCode(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		code         int
		htmx         bool
		wantStatus   int
		wantLocation string
		wantHeader   string
	}{
		{
			name:         "permanent redirect",
			url:          "/new-location",
			code:         http.StatusMovedPermanently,
			htmx:         false,
			wantStatus:   http.StatusMovedPermanently,
			wantLocation: "/new-location",
		},
		{
			name:         "temporary redirect",
			url:          "/temp-location",
			code:         http.StatusTemporaryRedirect,
			htmx:         false,
			wantStatus:   http.StatusTemporaryRedirect,
			wantLocation: "/temp-location",
		},
		{
			name:       "htmx redirect ignores code",
			url:        "/new-location",
			code:       http.StatusMovedPermanently,
			htmx:       true,
			wantStatus: http.StatusOK,
			wantHeader: "/new-location",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			if tt.htmx {
				r.Header.Set("HX-Request", "true")
			}

			resp := saaskit.RedirectWithCode(tt.url, tt.code)
			err := resp.Render(w, r)

			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.htmx {
				assert.Equal(t, tt.wantHeader, w.Header().Get("HX-Redirect"))
			} else {
				assert.Equal(t, tt.wantLocation, w.Header().Get("Location"))
			}
		})
	}
}

func TestRedirectBack(t *testing.T) {
	tests := []struct {
		name         string
		referer      string
		fallback     string
		htmx         bool
		wantStatus   int
		wantLocation string
		wantHeader   string
	}{
		{
			name:         "redirect to referer",
			referer:      "http://example.com/previous",
			fallback:     "/home",
			htmx:         false,
			wantStatus:   http.StatusSeeOther,
			wantLocation: "http://example.com/previous",
		},
		{
			name:         "redirect to fallback when no referer",
			referer:      "",
			fallback:     "/home",
			htmx:         false,
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/home",
		},
		{
			name:       "htmx redirect to referer",
			referer:    "http://example.com/previous",
			fallback:   "/home",
			htmx:       true,
			wantStatus: http.StatusOK,
			wantHeader: "http://example.com/previous",
		},
		{
			name:         "invalid referer URL falls back",
			referer:      "://invalid-url",
			fallback:     "/home",
			htmx:         false,
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/home",
		},
		{
			name:         "different host referer rejected",
			referer:      "http://evil.com/phishing",
			fallback:     "/home",
			htmx:         false,
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/home",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			r.Host = "example.com"
			if tt.referer != "" {
				r.Header.Set("Referer", tt.referer)
			}
			if tt.htmx {
				r.Header.Set("HX-Request", "true")
			}

			resp := saaskit.RedirectBack(tt.fallback)
			err := resp.Render(w, r)

			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.htmx {
				assert.Equal(t, tt.wantHeader, w.Header().Get("HX-Redirect"))
			} else {
				assert.Equal(t, tt.wantLocation, w.Header().Get("Location"))
			}
		})
	}
}

func TestRedirectBackWithCode(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Referer", "http://example.com/previous")
	r.Host = "example.com"

	resp := saaskit.RedirectBackWithCode("/home", http.StatusFound)
	err := resp.Render(w, r)

	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "http://example.com/previous", w.Header().Get("Location"))
}
