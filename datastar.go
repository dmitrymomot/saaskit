package saaskit

import (
	"net/http"
	"strings"

	"github.com/starfederation/datastar-go/datastar"
)

// DataStar detection constants
const (
	// DataStarAcceptHeader is the Accept header value that indicates a DataStar request
	DataStarAcceptHeader = "text/event-stream"

	// DataStarQueryParam is the query parameter used by DataStar for signals
	DataStarQueryParam = "datastar"
)

// Patch mode aliases for convenience
const (
	PatchOuter   = datastar.ElementPatchModeOuter   // Morphs element (default)
	PatchInner   = datastar.ElementPatchModeInner   // Replace inner HTML
	PatchReplace = datastar.ElementPatchModeReplace // Replace entire element
	PatchRemove  = datastar.ElementPatchModeRemove  // Remove element
	PatchAppend  = datastar.ElementPatchModeAppend  // Append inside element
	PatchPrepend = datastar.ElementPatchModePrepend // Prepend inside element
	PatchBefore  = datastar.ElementPatchModeBefore  // Insert before element
	PatchAfter   = datastar.ElementPatchModeAfter   // Insert after element
)

// IsDataStar checks if the request is a DataStar request.
// DataStar requests typically accept Server-Sent Events (SSE) and may include
// signals in the query parameter or request body.
func IsDataStar(r *http.Request) bool {
	// Check Accept header for SSE
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, DataStarAcceptHeader) {
		return true
	}

	// Check for DataStar query parameter
	if r.URL.Query().Has(DataStarQueryParam) {
		return true
	}

	// Check Content-Type for DataStar-specific types
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-datastar") {
		return true
	}

	return false
}

// NewDataStarSSE creates a new Server-Sent Event generator for DataStar responses.
// This is a wrapper around the DataStar SDK's NewSSE function.
func NewDataStarSSE(w http.ResponseWriter, r *http.Request) *datastar.ServerSentEventGenerator {
	return datastar.NewSSE(w, r)
}

// DataStarRedirect performs a redirect for DataStar requests using SSE.
// For non-DataStar requests, it falls back to standard HTTP redirect.
func DataStarRedirect(w http.ResponseWriter, r *http.Request, url string, code int) error {
	if IsDataStar(r) {
		// Use SSE for DataStar requests
		sse := datastar.NewSSE(w, r)
		return sse.Redirect(url)
	}

	// Standard HTTP redirect for non-DataStar requests
	http.Redirect(w, r, url, code)
	return nil
}
