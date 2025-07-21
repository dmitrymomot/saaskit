package saaskit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDataStar(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		query    string
		expected bool
	}{
		{
			name:     "SSE Accept header",
			headers:  map[string]string{"Accept": "text/event-stream"},
			expected: true,
		},
		{
			name:     "SSE Accept header with other values",
			headers:  map[string]string{"Accept": "text/html, text/event-stream, */*"},
			expected: true,
		},
		{
			name:     "DataStar query parameter",
			query:    "?datastar={\"count\":1}",
			expected: true,
		},
		{
			name:     "DataStar content type",
			headers:  map[string]string{"Content-Type": "application/x-datastar"},
			expected: true,
		},
		{
			name:     "Regular request",
			headers:  map[string]string{"Accept": "text/html"},
			expected: false,
		},
		{
			name:     "No headers",
			headers:  map[string]string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test"+tt.query, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := IsDataStar(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDataStarRedirect(t *testing.T) {
	t.Run("DataStar request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		err := DataStarRedirect(w, req, "/new-location", http.StatusSeeOther)
		require.NoError(t, err)

		// DataStar redirects use SSE, so check for event stream content type
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

		// The response should contain SSE data
		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "window.location.href")
		assert.Contains(t, body, "/new-location")
	})

	t.Run("Regular request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/html")

		w := httptest.NewRecorder()
		err := DataStarRedirect(w, req, "/new-location", http.StatusSeeOther)
		require.NoError(t, err)

		// Regular redirects use standard HTTP redirect
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/new-location", w.Header().Get("Location"))
	})
}

func TestRedirectResponseWithDataStar(t *testing.T) {
	t.Run("DataStar redirect", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/submit", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		resp := Redirect("/success")
		err := resp.Render(w, req)
		require.NoError(t, err)

		// Check for SSE response
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "/success")
	})

	t.Run("Regular redirect", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/submit", nil)

		w := httptest.NewRecorder()
		resp := Redirect("/success")
		err := resp.Render(w, req)
		require.NoError(t, err)

		// Check for standard HTTP redirect
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/success", w.Header().Get("Location"))
	})
}

func TestRedirectBackWithDataStar(t *testing.T) {
	t.Run("DataStar redirect back with referer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/delete", nil)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Referer", "/items")
		req.Host = "example.com"

		w := httptest.NewRecorder()
		resp := RedirectBack("/")
		err := resp.Render(w, req)
		require.NoError(t, err)

		// Check for SSE response
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "/items") // Should redirect to referer
	})

	t.Run("DataStar redirect back with fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/delete", nil)
		req.Header.Set("Accept", "text/event-stream")
		// No referer header

		w := httptest.NewRecorder()
		resp := RedirectBack("/home")
		err := resp.Render(w, req)
		require.NoError(t, err)

		// Check for SSE response
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "/home") // Should use fallback
	})

	t.Run("Regular redirect back", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/delete", nil)
		req.Header.Set("Referer", "/items")
		req.Host = "example.com"

		w := httptest.NewRecorder()
		resp := RedirectBack("/")
		err := resp.Render(w, req)
		require.NoError(t, err)

		// Check for standard HTTP redirect
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/items", w.Header().Get("Location"))
	})
}
