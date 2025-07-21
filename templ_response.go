package saaskit

import (
	"context"
	"io"
	"net/http"

	"github.com/starfederation/datastar-go/datastar"
)

// TemplComponent represents a templ component interface.
// This matches github.com/a-h/templ.Component without importing it.
type TemplComponent interface {
	Render(ctx context.Context, w io.Writer) error
}

// TemplOption is an alias for datastar's PatchElementOption
type TemplOption = datastar.PatchElementOption

// WithTarget sets the target selector for where the component should be rendered
func WithTarget(selector string) TemplOption {
	return datastar.WithSelector(selector)
}

// WithPatchMode sets how the component should be merged into the DOM
func WithPatchMode(mode datastar.ElementPatchMode) TemplOption {
	return datastar.WithMode(mode)
}

// templResponse wraps a templ component to implement Response
type templResponse struct {
	component TemplComponent
	options   []datastar.PatchElementOption
}

// Render renders the templ component appropriately for the request type
func (t templResponse) Render(w http.ResponseWriter, r *http.Request) error {
	// For DataStar requests, use SSE
	if IsDataStar(r) {
		sse := datastar.NewSSE(w, r)
		return sse.PatchElementTempl(t.component, t.options...)
	}

	// For regular HTTP requests, render directly
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.component.Render(r.Context(), w)
}

// Templ creates a response from a templ component with optional configuration.
// For DataStar requests, it renders via SSE with optional target and patch mode.
// For regular HTTP requests, it renders directly to the response.
//
// Simple usage:
//
//	return saaskit.Templ(templates.UserProfile(req.UserID))
//
// With target selector:
//
//	return saaskit.Templ(
//		templates.UserCard(user),
//		saaskit.WithTarget("#user-info"),
//	)
//
// Appending to a list:
//
//	return saaskit.Templ(
//		templates.TodoItem(todo),
//		saaskit.WithTarget("#todo-list"),
//		saaskit.WithPatchMode(saaskit.PatchAppend),
//	)
func Templ(component TemplComponent, opts ...TemplOption) Response {
	return templResponse{
		component: component,
		options:   opts,
	}
}

// templPartialResponse conditionally renders partial or full component based on request type
type templPartialResponse struct {
	partial TemplComponent
	full    TemplComponent
	options []datastar.PatchElementOption
}

// Render renders either the partial component for DataStar requests or the full component
func (t templPartialResponse) Render(w http.ResponseWriter, r *http.Request) error {
	// For DataStar requests, render only the partial component via SSE
	if IsDataStar(r) {
		sse := datastar.NewSSE(w, r)
		return sse.PatchElementTempl(t.partial, t.options...)
	}

	// For regular HTTP requests, render the full component
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.full.Render(r.Context(), w)
}

// TemplPartial creates a response that renders differently for DataStar vs regular requests.
// For DataStar requests, it renders only the partial component via SSE for targeted updates.
// For regular requests, it renders the full component.
//
// Example:
//
//	handler := saaskit.HandlerFunc[saaskit.Context, EditUserRequest](
//		func(ctx saaskit.Context, req EditUserRequest) saaskit.Response {
//			user := getUser(req.UserID)
//			partial := templates.UserEditForm(user)
//			full := templates.UserEditPage(user)
//			return saaskit.TemplPartial(partial, full)
//		},
//	)
//
// With options:
//
//	return saaskit.TemplPartial(
//		templates.NotificationItem(notif),
//		templates.NotificationPage(notif),
//		saaskit.WithTarget("#notifications"),
//		saaskit.WithPatchMode(saaskit.PatchPrepend),
//	)
func TemplPartial(partial, full TemplComponent, opts ...TemplOption) Response {
	return templPartialResponse{
		partial: partial,
		full:    full,
		options: opts,
	}
}
