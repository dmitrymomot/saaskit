package core_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	saaskit "github.com/dmitrymomot/saaskit/core"
)

func TestRedirect(t *testing.T) {
	t.Parallel()
	t.Run("regular redirect", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)

		resp := saaskit.Redirect("/users/123")
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/users/123", w.Header().Get("Location"))
	})

	t.Run("datastar redirect", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Accept", "text/event-stream")

		resp := saaskit.Redirect("/users/123")
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

		// Check SSE response body
		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "window.location.href")
		assert.Contains(t, body, "/users/123")
	})

	t.Run("datastar redirect with query param", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/?datastar={}", nil)

		resp := saaskit.Redirect("/users/123")
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

		// Check SSE response body
		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "/users/123")
	})
}

func TestRedirectWithCode(t *testing.T) {
	t.Parallel()
	t.Run("permanent redirect", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)

		resp := saaskit.RedirectWithCode("/new-location", http.StatusMovedPermanently)
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, http.StatusMovedPermanently, w.Code)
		assert.Equal(t, "/new-location", w.Header().Get("Location"))
	})

	t.Run("temporary redirect", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)

		resp := saaskit.RedirectWithCode("/temp-location", http.StatusTemporaryRedirect)
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
		assert.Equal(t, "/temp-location", w.Header().Get("Location"))
	})

	t.Run("datastar redirect ignores code", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Accept", "text/event-stream")

		resp := saaskit.RedirectWithCode("/new-location", http.StatusMovedPermanently)
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

		// DataStar uses SSE, not HTTP status codes
		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "/new-location")
	})
}

func TestRedirectBack(t *testing.T) {
	t.Parallel()
	t.Run("redirect to referer", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Host = "example.com"
		r.Header.Set("Referer", "http://example.com/previous")

		resp := saaskit.RedirectBack("/home")
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "http://example.com/previous", w.Header().Get("Location"))
	})

	t.Run("redirect to fallback when no referer", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Host = "example.com"

		resp := saaskit.RedirectBack("/home")
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/home", w.Header().Get("Location"))
	})

	t.Run("datastar redirect to referer", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Host = "example.com"
		r.Header.Set("Referer", "http://example.com/previous")
		r.Header.Set("Accept", "text/event-stream")

		resp := saaskit.RedirectBack("/home")
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "http://example.com/previous")
	})

	t.Run("invalid referer URL falls back", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Host = "example.com"
		r.Header.Set("Referer", "://invalid-url")

		resp := saaskit.RedirectBack("/home")
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/home", w.Header().Get("Location"))
	})

	t.Run("different host referer rejected", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Host = "example.com"
		r.Header.Set("Referer", "http://evil.com/phishing")

		resp := saaskit.RedirectBack("/home")
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/home", w.Header().Get("Location"))
	})

	t.Run("datastar redirect to fallback when referer invalid", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Host = "example.com"
		r.Header.Set("Referer", "http://evil.com/phishing")
		r.Header.Set("Accept", "text/event-stream")

		resp := saaskit.RedirectBack("/home")
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "/home") // Should use fallback, not the evil referer
	})
}

func TestRedirectBackWithCode(t *testing.T) {
	t.Parallel()
	t.Run("regular redirect with custom code", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Referer", "http://example.com/previous")
		r.Host = "example.com"

		resp := saaskit.RedirectBackWithCode("/home", http.StatusFound)
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "http://example.com/previous", w.Header().Get("Location"))
	})

	t.Run("datastar redirect back ignores code", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Referer", "http://example.com/previous")
		r.Host = "example.com"
		r.Header.Set("Accept", "text/event-stream")

		resp := saaskit.RedirectBackWithCode("/home", http.StatusFound)
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "http://example.com/previous")
	})
}
