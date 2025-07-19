package saaskit

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HTMX header constants
const (
	// Request headers
	HXRequest        = "HX-Request"
	HXBoosted        = "HX-Boosted"
	HXHistoryRestore = "HX-History-Restore-Request"
	HXPrompt         = "HX-Prompt"
	HXTarget         = "HX-Target"
	HXTriggerName    = "HX-Trigger-Name"
	HXTrigger        = "HX-Trigger"
	HXCurrentURL     = "HX-Current-URL"

	// Response headers
	HXRedirect           = "HX-Redirect"
	HXRefresh            = "HX-Refresh"
	HXLocation           = "HX-Location"
	HXPushURL            = "HX-Push-Url"
	HXReplaceURL         = "HX-Replace-Url"
	HXReswap             = "HX-Reswap"
	HXRetarget           = "HX-Retarget"
	HXReselect           = "HX-Reselect"
	HXTriggerAfterSettle = "HX-Trigger-After-Settle"
	HXTriggerAfterSwap   = "HX-Trigger-After-Swap"
)

// HTMX swap strategies
const (
	// SwapInnerHTML replaces the inner html of the target element (default)
	SwapInnerHTML = "innerHTML"

	// SwapOuterHTML replaces the entire target element with the response
	SwapOuterHTML = "outerHTML"

	// SwapBeforeBegin inserts the response before the target element
	SwapBeforeBegin = "beforebegin"

	// SwapAfterBegin inserts the response before the first child of the target element
	SwapAfterBegin = "afterbegin"

	// SwapBeforeEnd inserts the response after the last child of the target element
	SwapBeforeEnd = "beforeend"

	// SwapAfterEnd inserts the response after the target element
	SwapAfterEnd = "afterend"

	// SwapDelete deletes the target element regardless of the response
	SwapDelete = "delete"

	// SwapNone does not append content from response (out of band items will still be processed)
	SwapNone = "none"
)

// IsHTMX checks if the request is an HTMX request
func IsHTMX(r *http.Request) bool {
	return r.Header.Get(HXRequest) == "true"
}

// IsHTMXBoosted checks if the request is an HTMX boosted request
func IsHTMXBoosted(r *http.Request) bool {
	return r.Header.Get(HXBoosted) == "true"
}

// IsHTMXHistoryRestore checks if this is an HTMX history restore request
func IsHTMXHistoryRestore(r *http.Request) bool {
	return r.Header.Get(HXHistoryRestore) == "true"
}

// GetHTMXTarget returns the id of the target element if it exists
func GetHTMXTarget(r *http.Request) string {
	return r.Header.Get(HXTarget)
}

// GetHTMXTrigger returns the id of the triggered element if it exists
func GetHTMXTrigger(r *http.Request) string {
	return r.Header.Get(HXTrigger)
}

// GetHTMXTriggerName returns the name of the triggered element if it exists
func GetHTMXTriggerName(r *http.Request) string {
	return r.Header.Get(HXTriggerName)
}

// GetHTMXPrompt returns the response to an hx-prompt if it exists
func GetHTMXPrompt(r *http.Request) string {
	return r.Header.Get(HXPrompt)
}

// GetHTMXCurrentURL returns the current URL of the browser
func GetHTMXCurrentURL(r *http.Request) string {
	return r.Header.Get(HXCurrentURL)
}

// HTMXModifier is a function that modifies response headers
type HTMXModifier func(w http.ResponseWriter)

// ApplyHTMXModifiers returns a function that applies all modifiers if it's an HTMX request
func ApplyHTMXModifiers(modifiers ...HTMXModifier) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !IsHTMX(r) {
			return
		}
		for _, mod := range modifiers {
			mod(w)
		}
	}
}

// SetHTMXRetarget creates a modifier that sets the HX-Retarget header
func SetHTMXRetarget(targetID string) HTMXModifier {
	return func(w http.ResponseWriter) {
		if targetID != "" {
			w.Header().Set(HXRetarget, targetID)
		}
	}
}

// SetHTMXReswap creates a modifier that sets the HX-Reswap header
func SetHTMXReswap(strategy string) HTMXModifier {
	return func(w http.ResponseWriter) {
		if strategy != "" {
			w.Header().Set(HXReswap, strategy)
		}
	}
}

// SetHTMXTrigger creates a modifier that sets the HX-Trigger header
func SetHTMXTrigger(events string) HTMXModifier {
	return func(w http.ResponseWriter) {
		if events != "" {
			w.Header().Set(HXTrigger, events)
		}
	}
}

// SetHTMXTriggerAfterSwap creates a modifier that sets the HX-Trigger-After-Swap header
func SetHTMXTriggerAfterSwap(events string) HTMXModifier {
	return func(w http.ResponseWriter) {
		if events != "" {
			w.Header().Set(HXTriggerAfterSwap, events)
		}
	}
}

// SetHTMXTriggerAfterSettle creates a modifier that sets the HX-Trigger-After-Settle header
func SetHTMXTriggerAfterSettle(events string) HTMXModifier {
	return func(w http.ResponseWriter) {
		if events != "" {
			w.Header().Set(HXTriggerAfterSettle, events)
		}
	}
}

// SetHTMXPushURL creates a modifier that sets the HX-Push-Url header
func SetHTMXPushURL(url string) HTMXModifier {
	return func(w http.ResponseWriter) {
		if url != "" {
			w.Header().Set(HXPushURL, url)
		}
	}
}

// SetHTMXReplaceURL creates a modifier that sets the HX-Replace-Url header
func SetHTMXReplaceURL(url string) HTMXModifier {
	return func(w http.ResponseWriter) {
		if url != "" {
			w.Header().Set(HXReplaceURL, url)
		}
	}
}

// SetHTMXReselect creates a modifier that sets the HX-Reselect header
func SetHTMXReselect(selector string) HTMXModifier {
	return func(w http.ResponseWriter) {
		if selector != "" {
			w.Header().Set(HXReselect, selector)
		}
	}
}

// SetHTMXRefresh creates a modifier that sets the HX-Refresh header
func SetHTMXRefresh() HTMXModifier {
	return func(w http.ResponseWriter) {
		w.Header().Set(HXRefresh, "true")
	}
}

// SwapModifier is a function that modifies a swap string builder
type SwapModifier func(*strings.Builder)

// SwapStrategy sets the base swap strategy (required first modifier)
func SwapStrategy(strategy string) SwapModifier {
	return func(b *strings.Builder) {
		b.WriteString(strategy)
	}
}

// SwapAfter sets the swap timing delay
func SwapAfter(d time.Duration) SwapModifier {
	return func(b *strings.Builder) {
		fmt.Fprintf(b, " swap:%dms", d.Milliseconds())
	}
}

// SwapSettle sets the settle timing delay
func SwapSettle(d time.Duration) SwapModifier {
	return func(b *strings.Builder) {
		fmt.Fprintf(b, " settle:%dms", d.Milliseconds())
	}
}

// SwapScrollTop scrolls to the top of the target element
func SwapScrollTop() SwapModifier {
	return func(b *strings.Builder) {
		b.WriteString(" scroll:top")
	}
}

// SwapScrollBottom scrolls to the bottom of the target element
func SwapScrollBottom() SwapModifier {
	return func(b *strings.Builder) {
		b.WriteString(" scroll:bottom")
	}
}

// SwapScrollTo scrolls to a specific element selector
func SwapScrollTo(selector string) SwapModifier {
	return func(b *strings.Builder) {
		b.WriteString(" scroll:" + selector)
	}
}

// SwapShowTop shows the top of the target element
func SwapShowTop() SwapModifier {
	return func(b *strings.Builder) {
		b.WriteString(" show:top")
	}
}

// SwapShowBottom shows the bottom of the target element
func SwapShowBottom() SwapModifier {
	return func(b *strings.Builder) {
		b.WriteString(" show:bottom")
	}
}

// SwapShowTo shows a specific element selector
func SwapShowTo(selector string) SwapModifier {
	return func(b *strings.Builder) {
		b.WriteString(" show:" + selector)
	}
}

// SwapFocusScroll enables or disables focus scrolling
func SwapFocusScroll(enabled bool) SwapModifier {
	return func(b *strings.Builder) {
		fmt.Fprintf(b, " focus-scroll:%t", enabled)
	}
}

// BuildSwap combines swap modifiers into a swap string
func BuildSwap(modifiers ...SwapModifier) string {
	var b strings.Builder
	for _, mod := range modifiers {
		mod(&b)
	}
	return b.String()
}

// SetHTMXReswapModifiers creates a modifier that sets the HX-Reswap header with modifiers
func SetHTMXReswapModifiers(modifiers ...SwapModifier) HTMXModifier {
	return func(w http.ResponseWriter) {
		swap := BuildSwap(modifiers...)
		if swap != "" {
			w.Header().Set(HXReswap, swap)
		}
	}
}
