package saaskit

import "net/http"

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
