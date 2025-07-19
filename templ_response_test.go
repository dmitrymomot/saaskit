package saaskit_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit"
)

// mockTemplComponent is a test implementation of TemplComponent
type mockTemplComponent struct {
	content string
	err     error
}

func (m mockTemplComponent) Render(ctx context.Context, w io.Writer) error {
	if m.err != nil {
		return m.err
	}
	_, err := w.Write([]byte(m.content))
	return err
}

func TestTempl(t *testing.T) {
	tests := []struct {
		name      string
		component saaskit.TemplComponent
		wantBody  string
		wantErr   bool
	}{
		{
			name:      "successful render",
			component: mockTemplComponent{content: "<h1>Hello World</h1>"},
			wantBody:  "<h1>Hello World</h1>",
			wantErr:   false,
		},
		{
			name:      "render error",
			component: mockTemplComponent{err: assert.AnError},
			wantBody:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)

			resp := saaskit.Templ(tt.component)
			err := resp.Render(w, r)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestTemplWithModifiers(t *testing.T) {
	component := mockTemplComponent{content: "<div>Alert</div>"}

	tests := []struct {
		name        string
		modifiers   []saaskit.ResponseModifier
		htmx        bool
		wantHeaders map[string]string
	}{
		{
			name: "with retarget for HTMX",
			modifiers: []saaskit.ResponseModifier{
				saaskit.ApplyHTMXModifiers(
					saaskit.SetHTMXRetarget("#notifications"),
				),
			},
			htmx: true,
			wantHeaders: map[string]string{
				"HX-Retarget": "#notifications",
			},
		},
		{
			name: "with reswap for HTMX",
			modifiers: []saaskit.ResponseModifier{
				saaskit.ApplyHTMXModifiers(
					saaskit.SetHTMXReswapModifiers(
						saaskit.SwapStrategy(saaskit.SwapAfterBegin),
					),
				),
			},
			htmx: true,
			wantHeaders: map[string]string{
				"HX-Reswap": "afterbegin",
			},
		},
		{
			name: "with trigger for HTMX",
			modifiers: []saaskit.ResponseModifier{
				saaskit.ApplyHTMXModifiers(
					saaskit.SetHTMXTrigger("notification-shown"),
				),
			},
			htmx: true,
			wantHeaders: map[string]string{
				"HX-Trigger": "notification-shown",
			},
		},
		{
			name: "with multiple HTMX modifiers",
			modifiers: []saaskit.ResponseModifier{
				saaskit.ApplyHTMXModifiers(
					saaskit.SetHTMXRetarget("#alerts"),
					saaskit.SetHTMXReswapModifiers(
						saaskit.SwapStrategy(saaskit.SwapBeforeEnd),
					),
					saaskit.SetHTMXTrigger("alert-added"),
				),
			},
			htmx: true,
			wantHeaders: map[string]string{
				"HX-Retarget": "#alerts",
				"HX-Reswap":   "beforeend",
				"HX-Trigger":  "alert-added",
			},
		},
		{
			name: "modifiers ignored for non-HTMX requests",
			modifiers: []saaskit.ResponseModifier{
				saaskit.ApplyHTMXModifiers(
					saaskit.SetHTMXRetarget("#notifications"),
					saaskit.SetHTMXReswapModifiers(
						saaskit.SwapStrategy(saaskit.SwapAfterBegin),
					),
				),
			},
			htmx:        false,
			wantHeaders: map[string]string{},
		},
		{
			name: "with complex swap modifiers",
			modifiers: []saaskit.ResponseModifier{
				saaskit.ApplyHTMXModifiers(
					saaskit.SetHTMXReswapModifiers(
						saaskit.SwapStrategy(saaskit.SwapInnerHTML),
						saaskit.SwapSettle(500*time.Millisecond),
						saaskit.SwapScrollTop(),
						saaskit.SwapFocusScroll(true),
					),
				),
			},
			htmx: true,
			wantHeaders: map[string]string{
				"HX-Reswap": "innerHTML settle:500ms scroll:top focus-scroll:true",
			},
		},
		{
			name: "custom response modifier",
			modifiers: []saaskit.ResponseModifier{
				func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Custom-Header", "custom-value")
				},
			},
			htmx: false,
			wantHeaders: map[string]string{
				"X-Custom-Header": "custom-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			if tt.htmx {
				r.Header.Set("HX-Request", "true")
			}

			resp := saaskit.Templ(component, tt.modifiers...)
			err := resp.Render(w, r)

			require.NoError(t, err)
			assert.Equal(t, "<div>Alert</div>", w.Body.String())

			// Check expected headers
			for header, value := range tt.wantHeaders {
				assert.Equal(t, value, w.Header().Get(header), "Header %s", header)
			}

			// Check that unwanted headers are not set
			if !tt.htmx {
				assert.Empty(t, w.Header().Get("HX-Retarget"))
				assert.Empty(t, w.Header().Get("HX-Reswap"))
				assert.Empty(t, w.Header().Get("HX-Trigger"))
			}
		})
	}
}

func TestTemplPartial(t *testing.T) {
	partialComponent := mockTemplComponent{content: "<div>Partial Content</div>"}
	fullComponent := mockTemplComponent{content: "<html><body><div>Partial Content</div></body></html>"}

	tests := []struct {
		name     string
		htmx     bool
		boosted  bool
		wantBody string
	}{
		{
			name:     "regular request renders full component",
			htmx:     false,
			boosted:  false,
			wantBody: "<html><body><div>Partial Content</div></body></html>",
		},
		{
			name:     "htmx request renders partial only",
			htmx:     true,
			boosted:  false,
			wantBody: "<div>Partial Content</div>",
		},
		{
			name:     "htmx boosted request renders full component",
			htmx:     true,
			boosted:  true,
			wantBody: "<html><body><div>Partial Content</div></body></html>",
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

			resp := saaskit.TemplPartial(partialComponent, fullComponent)
			err := resp.Render(w, r)

			require.NoError(t, err)
			assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
			assert.Equal(t, tt.wantBody, w.Body.String())
		})
	}
}

func TestTemplPartial_Error(t *testing.T) {
	tests := []struct {
		name             string
		partialComponent saaskit.TemplComponent
		fullComponent    saaskit.TemplComponent
		htmx             bool
		wantErr          bool
	}{
		{
			name:             "partial render error on htmx request",
			partialComponent: mockTemplComponent{err: assert.AnError},
			fullComponent:    mockTemplComponent{content: "full"},
			htmx:             true,
			wantErr:          true,
		},
		{
			name:             "full render error on regular request",
			partialComponent: mockTemplComponent{content: "partial"},
			fullComponent:    mockTemplComponent{err: assert.AnError},
			htmx:             false,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			if tt.htmx {
				r.Header.Set("HX-Request", "true")
			}

			resp := saaskit.TemplPartial(tt.partialComponent, tt.fullComponent)
			err := resp.Render(w, r)

			assert.Error(t, err)
		})
	}
}
