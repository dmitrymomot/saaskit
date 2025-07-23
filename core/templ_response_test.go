package core_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/starfederation/datastar-go/datastar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	saaskit "github.com/dmitrymomot/saaskit/core"
)

// mockTemplComponent implements saaskit.TemplComponent for testing
type mockTemplComponent struct {
	content   string
	renderErr error
}

func (m mockTemplComponent) Render(ctx context.Context, w io.Writer) error {
	if m.renderErr != nil {
		return m.renderErr
	}
	_, err := w.Write([]byte(m.content))
	return err
}

func TestTempl(t *testing.T) {
	t.Parallel()
	t.Run("DataStar request without options", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		component := mockTemplComponent{content: "<div>Hello DataStar</div>"}
		resp := saaskit.Templ(component)

		err := resp.Render(w, req)
		require.NoError(t, err)

		// Check SSE response
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "<div>Hello DataStar</div>")
	})

	t.Run("DataStar request with target", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		component := mockTemplComponent{content: "<div>Targeted content</div>"}
		resp := saaskit.Templ(component, saaskit.WithTarget("#my-target"))

		err := resp.Render(w, req)
		require.NoError(t, err)

		// Check SSE response
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "<div>Targeted content</div>")
		assert.Contains(t, body, "#my-target")
	})

	t.Run("DataStar request with patch mode", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		component := mockTemplComponent{content: "<li>New item</li>"}
		resp := saaskit.Templ(component,
			saaskit.WithTarget("#list"),
			saaskit.WithPatchMode(saaskit.PatchAppend))

		err := resp.Render(w, req)
		require.NoError(t, err)

		// Check SSE response
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "<li>New item</li>")
		assert.Contains(t, body, "#list")
	})

	t.Run("Regular HTTP request", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/html")

		w := httptest.NewRecorder()
		component := mockTemplComponent{content: "<html><body>Full page</body></html>"}
		resp := saaskit.Templ(component)

		err := resp.Render(w, req)
		require.NoError(t, err)

		// Check regular HTML response
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
		assert.Equal(t, "<html><body>Full page</body></html>", w.Body.String())
	})

	t.Run("Component render error", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/html")

		w := httptest.NewRecorder()
		component := mockTemplComponent{renderErr: errors.New("render failed")}
		resp := saaskit.Templ(component)

		err := resp.Render(w, req)
		assert.Error(t, err)
		assert.Equal(t, "render failed", err.Error())
	})
}

func TestTemplPartial(t *testing.T) {
	t.Parallel()
	t.Run("DataStar request renders partial", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		partial := mockTemplComponent{content: "<div>Partial content</div>"}
		full := mockTemplComponent{content: "<html><body>Full page</body></html>"}
		resp := saaskit.TemplPartial(partial, full)

		err := resp.Render(w, req)
		require.NoError(t, err)

		// Should render only the partial via SSE
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "<div>Partial content</div>")
		assert.NotContains(t, body, "Full page")
	})

	t.Run("DataStar request with options", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		partial := mockTemplComponent{content: "<div>Notification</div>"}
		full := mockTemplComponent{content: "<html><body>Full notification page</body></html>"}
		resp := saaskit.TemplPartial(partial, full,
			saaskit.WithTarget("#notifications"),
			saaskit.WithPatchMode(saaskit.PatchPrepend))

		err := resp.Render(w, req)
		require.NoError(t, err)

		// Check SSE response with options
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		body := w.Body.String()
		assert.Contains(t, body, "datastar-patch-elements")
		assert.Contains(t, body, "<div>Notification</div>")
		assert.Contains(t, body, "#notifications")
	})

	t.Run("Regular HTTP request renders full", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/html")

		w := httptest.NewRecorder()
		partial := mockTemplComponent{content: "<div>Partial content</div>"}
		full := mockTemplComponent{content: "<html><body>Full page</body></html>"}
		resp := saaskit.TemplPartial(partial, full)

		err := resp.Render(w, req)
		require.NoError(t, err)

		// Should render the full component
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
		assert.Equal(t, "<html><body>Full page</body></html>", w.Body.String())
		assert.NotContains(t, w.Body.String(), "Partial content")
	})

	t.Run("Partial render error", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		partial := mockTemplComponent{renderErr: errors.New("partial failed")}
		full := mockTemplComponent{content: "<html><body>Full page</body></html>"}
		resp := saaskit.TemplPartial(partial, full)

		err := resp.Render(w, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "partial failed")
	})

	t.Run("Full render error", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/html")

		w := httptest.NewRecorder()
		partial := mockTemplComponent{content: "<div>Partial</div>"}
		full := mockTemplComponent{renderErr: errors.New("full failed")}
		resp := saaskit.TemplPartial(partial, full)

		err := resp.Render(w, req)
		assert.Error(t, err)
		assert.Equal(t, "full failed", err.Error())
	})
}

func TestTemplMulti(t *testing.T) {
	t.Parallel()
	t.Run("DataStar request with multiple patches", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		resp := saaskit.TemplMulti(
			saaskit.Patch(
				mockTemplComponent{content: "<div>Main content</div>"},
				saaskit.WithTarget("#main"),
			),
			saaskit.Patch(
				mockTemplComponent{content: "<span>Badge: 5</span>"},
				saaskit.WithTarget("#badge"),
				saaskit.WithPatchMode(saaskit.PatchOuter),
			),
			saaskit.Patch(
				mockTemplComponent{content: "<div>Success notification</div>"},
				saaskit.WithTarget("#notifications"),
				saaskit.WithPatchMode(saaskit.PatchPrepend),
			),
		)

		err := resp.Render(w, req)
		require.NoError(t, err)

		// Check multiple SSE patches
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		body := w.Body.String()

		// Should contain multiple patch events
		patchCount := strings.Count(body, "datastar-patch-elements")
		assert.Equal(t, 3, patchCount, "Should have 3 patch events")

		// Check all content is present
		assert.Contains(t, body, "<div>Main content</div>")
		assert.Contains(t, body, "<span>Badge: 5</span>")
		assert.Contains(t, body, "<div>Success notification</div>")

		// Check targets
		assert.Contains(t, body, "#main")
		assert.Contains(t, body, "#badge")
		assert.Contains(t, body, "#notifications")
	})

	t.Run("Regular HTTP request concatenates all", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/html")

		w := httptest.NewRecorder()
		resp := saaskit.TemplMulti(
			saaskit.Patch(mockTemplComponent{content: "<header>Header</header>"}),
			saaskit.Patch(mockTemplComponent{content: "<main>Main</main>"}),
			saaskit.Patch(mockTemplComponent{content: "<footer>Footer</footer>"}),
		)

		err := resp.Render(w, req)
		require.NoError(t, err)

		// Should concatenate all components
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
		expected := "<header>Header</header><main>Main</main><footer>Footer</footer>"
		assert.Equal(t, expected, w.Body.String())
	})

	t.Run("Empty patches list", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		resp := saaskit.TemplMulti()

		err := resp.Render(w, req)
		require.NoError(t, err)

		// Should handle empty list gracefully
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	})

	t.Run("Error in one patch stops processing", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		resp := saaskit.TemplMulti(
			saaskit.Patch(mockTemplComponent{content: "<div>First</div>"}),
			saaskit.Patch(mockTemplComponent{renderErr: errors.New("second failed")}),
			saaskit.Patch(mockTemplComponent{content: "<div>Third</div>"}),
		)

		err := resp.Render(w, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "second failed")

		// First patch might have been sent, but third should not
		body := w.Body.String()
		assert.NotContains(t, body, "<div>Third</div>")
	})

	t.Run("Regular HTTP error in concatenation", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/html")

		w := httptest.NewRecorder()
		resp := saaskit.TemplMulti(
			saaskit.Patch(mockTemplComponent{content: "<div>First</div>"}),
			saaskit.Patch(mockTemplComponent{renderErr: errors.New("render failed")}),
		)

		err := resp.Render(w, req)
		assert.Error(t, err)
		assert.Equal(t, "render failed", err.Error())
	})
}

func TestPatch(t *testing.T) {
	t.Parallel()
	t.Run("creates patch with component only", func(t *testing.T) {
		t.Parallel()
		component := mockTemplComponent{content: "<div>Test</div>"}
		patch := saaskit.Patch(component)

		assert.Equal(t, component, patch.Component)
		assert.Empty(t, patch.Options)
	})

	t.Run("creates patch with options", func(t *testing.T) {
		t.Parallel()
		component := mockTemplComponent{content: "<div>Test</div>"}
		patch := saaskit.Patch(component,
			saaskit.WithTarget("#target"),
			saaskit.WithPatchMode(saaskit.PatchAppend),
		)

		assert.Equal(t, component, patch.Component)
		assert.Len(t, patch.Options, 2)
	})
}

func TestWithTarget(t *testing.T) {
	t.Parallel()
	t.Run("creates selector option", func(t *testing.T) {
		t.Parallel()
		opt := saaskit.WithTarget("#my-element")
		// Since we can't inspect the internal option directly,
		// we test it by using it in a response
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		component := mockTemplComponent{content: "<div>Content</div>"}
		resp := saaskit.Templ(component, opt)

		err := resp.Render(w, req)
		require.NoError(t, err)

		body := w.Body.String()
		assert.Contains(t, body, "#my-element")
	})
}

func TestWithPatchMode(t *testing.T) {
	t.Parallel()
	t.Run("creates mode option", func(t *testing.T) {
		t.Parallel()
		opt := saaskit.WithPatchMode(datastar.ElementPatchModeAppend)
		// Test by using it in a response
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()
		component := mockTemplComponent{content: "<li>Item</li>"}
		resp := saaskit.Templ(component, opt)

		err := resp.Render(w, req)
		require.NoError(t, err)

		// The response should work without error
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	})
}

// Integration test with multiple components
func TestTemplResponseIntegration(t *testing.T) {
	t.Parallel()
	t.Run("complex multi-update scenario", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/update-order", nil)
		req.Header.Set("Accept", "text/event-stream")

		w := httptest.NewRecorder()

		// Simulate updating multiple parts of a page after an order update
		resp := saaskit.TemplMulti(
			// Update order details
			saaskit.Patch(
				mockTemplComponent{content: `<div class="order">Order #123 - Shipped</div>`},
				saaskit.WithTarget("#order-details"),
				saaskit.WithPatchMode(saaskit.PatchInner),
			),
			// Update cart badge
			saaskit.Patch(
				mockTemplComponent{content: `<span class="badge">0</span>`},
				saaskit.WithTarget("#cart-badge"),
				saaskit.WithPatchMode(saaskit.PatchOuter),
			),
			// Add success notification
			saaskit.Patch(
				mockTemplComponent{content: `<div class="alert success">Order placed successfully!</div>`},
				saaskit.WithTarget("#notifications"),
				saaskit.WithPatchMode(saaskit.PatchPrepend),
			),
		)

		err := resp.Render(w, req)
		require.NoError(t, err)

		body := w.Body.String()

		// Verify all updates are present
		assert.Contains(t, body, "Order #123 - Shipped")
		assert.Contains(t, body, `<span class="badge">0</span>`)
		assert.Contains(t, body, "Order placed successfully!")

		// Verify targets
		assert.Contains(t, body, "#order-details")
		assert.Contains(t, body, "#cart-badge")
		assert.Contains(t, body, "#notifications")
	})
}
