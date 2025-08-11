package handler

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
	return strings.Contains(contentType, "application/x-datastar")
}

// NewSSE creates a Server-Sent Event generator for DataStar responses.
func NewSSE(w http.ResponseWriter, r *http.Request) *datastar.ServerSentEventGenerator {
	return datastar.NewSSE(w, r)
}
