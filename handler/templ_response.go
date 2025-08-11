package handler

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

// TemplPatch represents a component with its own rendering options
type TemplPatch struct {
	Component TemplComponent
	Options   []datastar.PatchElementOption
}

// Patch creates a TemplPatch with options for use with TemplMulti
func Patch(component TemplComponent, opts ...TemplOption) TemplPatch {
	return TemplPatch{
		Component: component,
		Options:   opts,
	}
}

// templResponse wraps a templ component to implement Response
type templResponse struct {
	component TemplComponent
	options   []datastar.PatchElementOption
}

// Render outputs component via SSE for DataStar or HTML for regular requests
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

// Render outputs partial for DataStar SSE or full component for regular HTML
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

// templMultiResponse renders multiple components to different targets
type templMultiResponse struct {
	patches []TemplPatch
}

// Render sends multiple SSE patches for DataStar or concatenated HTML
func (t templMultiResponse) Render(w http.ResponseWriter, r *http.Request) error {
	// For DataStar requests, send multiple SSE patches
	if IsDataStar(r) {
		sse := datastar.NewSSE(w, r)
		for _, patch := range t.patches {
			if err := sse.PatchElementTempl(patch.Component, patch.Options...); err != nil {
				return err
			}
		}
		return nil
	}

	// For regular HTTP requests, concatenate all components
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	for _, patch := range t.patches {
		if err := patch.Component.Render(r.Context(), w); err != nil {
			return err
		}
	}
	return nil
}

// TemplMulti renders multiple components to different targets.
// For DataStar requests, each component is sent as a separate SSE patch with its own options.
// For regular HTTP requests, all components are concatenated in order.
//
// Example usage:
//
//	// Update main content and add a notification
//	return saaskit.TemplMulti(
//		saaskit.Patch(templates.UpdatedUserProfile(user),
//			saaskit.WithTarget("#user-profile")),
//		saaskit.Patch(templates.SuccessNotification("Profile updated!"),
//			saaskit.WithTarget("#notifications"),
//			saaskit.WithPatchMode(saaskit.PatchPrepend)),
//	)
//
// Complex example:
//
//	// Update multiple parts of the page
//	return saaskit.TemplMulti(
//		// Replace main content
//		saaskit.Patch(templates.OrderDetails(order),
//			saaskit.WithTarget("#order-details"),
//			saaskit.WithPatchMode(saaskit.PatchInner)),
//		// Update cart badge
//		saaskit.Patch(templates.CartBadge(itemCount),
//			saaskit.WithTarget("#cart-badge"),
//			saaskit.WithPatchMode(saaskit.PatchOuter)),
//		// Add order confirmation notification
//		saaskit.Patch(templates.OrderConfirmation(order.ID),
//			saaskit.WithTarget("#notifications"),
//			saaskit.WithPatchMode(saaskit.PatchPrepend)),
//	)
func TemplMulti(patches ...TemplPatch) Response {
	return templMultiResponse{
		patches: patches,
	}
}
