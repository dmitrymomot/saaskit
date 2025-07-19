package saaskit_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit"
)

func TestApplyHTMXModifiers(t *testing.T) {
	tests := []struct {
		name        string
		modifiers   []saaskit.HTMXModifier
		htmx        bool
		wantHeaders map[string]string
	}{
		{
			name: "single modifier with HTMX",
			modifiers: []saaskit.HTMXModifier{
				saaskit.SetHTMXRetarget("#notifications"),
			},
			htmx: true,
			wantHeaders: map[string]string{
				"HX-Retarget": "#notifications",
			},
		},
		{
			name: "multiple modifiers with HTMX",
			modifiers: []saaskit.HTMXModifier{
				saaskit.SetHTMXRetarget("#alerts"),
				saaskit.SetHTMXReswap(saaskit.SwapAfterBegin),
				saaskit.SetHTMXTrigger("alert-shown"),
			},
			htmx: true,
			wantHeaders: map[string]string{
				"HX-Retarget": "#alerts",
				"HX-Reswap":   "afterbegin",
				"HX-Trigger":  "alert-shown",
			},
		},
		{
			name: "modifiers ignored for non-HTMX",
			modifiers: []saaskit.HTMXModifier{
				saaskit.SetHTMXRetarget("#notifications"),
				saaskit.SetHTMXReswap(saaskit.SwapAfterBegin),
			},
			htmx:        false,
			wantHeaders: map[string]string{},
		},
		{
			name: "empty values are ignored",
			modifiers: []saaskit.HTMXModifier{
				saaskit.SetHTMXRetarget(""),
				saaskit.SetHTMXReswap(""),
				saaskit.SetHTMXTrigger("valid-trigger"),
			},
			htmx: true,
			wantHeaders: map[string]string{
				"HX-Trigger": "valid-trigger",
			},
		},
		{
			name: "all modifier types",
			modifiers: []saaskit.HTMXModifier{
				saaskit.SetHTMXRetarget("#target"),
				saaskit.SetHTMXReswap(saaskit.SwapInnerHTML),
				saaskit.SetHTMXTrigger("event1"),
				saaskit.SetHTMXTriggerAfterSwap("event2"),
				saaskit.SetHTMXTriggerAfterSettle("event3"),
				saaskit.SetHTMXPushURL("/new-url"),
				saaskit.SetHTMXReplaceURL("/replace-url"),
				saaskit.SetHTMXReselect(".new-selector"),
				saaskit.SetHTMXRefresh(),
			},
			htmx: true,
			wantHeaders: map[string]string{
				"HX-Retarget":             "#target",
				"HX-Reswap":               "innerHTML",
				"HX-Trigger":              "event1",
				"HX-Trigger-After-Swap":   "event2",
				"HX-Trigger-After-Settle": "event3",
				"HX-Push-Url":             "/new-url",
				"HX-Replace-Url":          "/replace-url",
				"HX-Reselect":             ".new-selector",
				"HX-Refresh":              "true",
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

			// Apply modifiers
			saaskit.ApplyHTMXModifiers(tt.modifiers...)(w, r)

			// Check headers
			for header, value := range tt.wantHeaders {
				assert.Equal(t, value, w.Header().Get(header), "Header %s", header)
			}

			// Ensure no headers are set for non-HTMX requests
			if !tt.htmx {
				assert.Empty(t, w.Header().Get("HX-Retarget"))
				assert.Empty(t, w.Header().Get("HX-Reswap"))
				assert.Empty(t, w.Header().Get("HX-Trigger"))
			}
		})
	}
}

func TestHTMXModifiersIndividually(t *testing.T) {
	tests := []struct {
		name     string
		modifier saaskit.HTMXModifier
		header   string
		value    string
	}{
		{
			name:     "SetHTMXRetarget",
			modifier: saaskit.SetHTMXRetarget("#container"),
			header:   "HX-Retarget",
			value:    "#container",
		},
		{
			name:     "SetHTMXReswap",
			modifier: saaskit.SetHTMXReswap(saaskit.SwapOuterHTML),
			header:   "HX-Reswap",
			value:    "outerHTML",
		},
		{
			name:     "SetHTMXTrigger",
			modifier: saaskit.SetHTMXTrigger("my-event"),
			header:   "HX-Trigger",
			value:    "my-event",
		},
		{
			name:     "SetHTMXRefresh",
			modifier: saaskit.SetHTMXRefresh(),
			header:   "HX-Refresh",
			value:    "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.modifier(w)
			assert.Equal(t, tt.value, w.Header().Get(tt.header))
		})
	}
}

func TestSwapModifiers(t *testing.T) {
	tests := []struct {
		name      string
		modifiers []saaskit.SwapModifier
		want      string
	}{
		{
			name: "single strategy",
			modifiers: []saaskit.SwapModifier{
				saaskit.SwapStrategy(saaskit.SwapInnerHTML),
			},
			want: "innerHTML",
		},
		{
			name: "strategy with settle",
			modifiers: []saaskit.SwapModifier{
				saaskit.SwapStrategy(saaskit.SwapAfterBegin),
				saaskit.SwapSettle(300 * time.Millisecond),
			},
			want: "afterbegin settle:300ms",
		},
		{
			name: "strategy with swap delay",
			modifiers: []saaskit.SwapModifier{
				saaskit.SwapStrategy(saaskit.SwapOuterHTML),
				saaskit.SwapAfter(150 * time.Millisecond),
			},
			want: "outerHTML swap:150ms",
		},
		{
			name: "with scroll modifiers",
			modifiers: []saaskit.SwapModifier{
				saaskit.SwapStrategy(saaskit.SwapBeforeEnd),
				saaskit.SwapScrollTop(),
			},
			want: "beforeend scroll:top",
		},
		{
			name: "with scroll to selector",
			modifiers: []saaskit.SwapModifier{
				saaskit.SwapStrategy(saaskit.SwapAfterEnd),
				saaskit.SwapScrollTo("#notifications"),
			},
			want: "afterend scroll:#notifications",
		},
		{
			name: "with show modifiers",
			modifiers: []saaskit.SwapModifier{
				saaskit.SwapStrategy(saaskit.SwapDelete),
				saaskit.SwapShowBottom(),
			},
			want: "delete show:bottom",
		},
		{
			name: "with focus scroll",
			modifiers: []saaskit.SwapModifier{
				saaskit.SwapStrategy(saaskit.SwapNone),
				saaskit.SwapFocusScroll(false),
			},
			want: "none focus-scroll:false",
		},
		{
			name: "complex combination",
			modifiers: []saaskit.SwapModifier{
				saaskit.SwapStrategy(saaskit.SwapInnerHTML),
				saaskit.SwapAfter(200 * time.Millisecond),
				saaskit.SwapSettle(500 * time.Millisecond),
				saaskit.SwapScrollTop(),
				saaskit.SwapShowTo("#header"),
				saaskit.SwapFocusScroll(true),
			},
			want: "innerHTML swap:200ms settle:500ms scroll:top show:#header focus-scroll:true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := saaskit.BuildSwap(tt.modifiers...)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSetHTMXReswapModifiers(t *testing.T) {
	modifiers := []saaskit.SwapModifier{
		saaskit.SwapStrategy(saaskit.SwapAfterBegin),
		saaskit.SwapSettle(250 * time.Millisecond),
		saaskit.SwapScrollBottom(),
	}

	modifier := saaskit.SetHTMXReswapModifiers(modifiers...)

	w := httptest.NewRecorder()
	modifier(w)

	assert.Equal(t, "afterbegin settle:250ms scroll:bottom", w.Header().Get("HX-Reswap"))
}
