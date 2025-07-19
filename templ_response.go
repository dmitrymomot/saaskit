package saaskit

import (
	"context"
	"io"
	"net/http"
)

// TemplComponent represents a templ component interface.
// This matches github.com/a-h/templ.Component without importing it.
type TemplComponent interface {
	Render(ctx context.Context, w io.Writer) error
}

// ResponseModifier is a function that modifies the HTTP response
type ResponseModifier func(http.ResponseWriter, *http.Request)

// templResponse wraps a templ component to implement Response
type templResponse struct {
	component TemplComponent
	modifiers []ResponseModifier
}

// Render renders the templ component
func (t templResponse) Render(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Apply all response modifiers
	for _, mod := range t.modifiers {
		mod(w, r)
	}

	return t.component.Render(r.Context(), w)
}

// Templ creates a response from a templ component with optional response modifiers.
// The component must implement the TemplComponent interface (which matches templ.Component).
//
// Simple usage:
//
//	return saaskit.Templ(templates.UserProfile(req.UserID))
//
// With response modifiers (e.g., HTMX):
//
//	return saaskit.Templ(
//		templates.ErrorAlert("Invalid email"),
//		saaskit.ApplyHTMXModifiers(
//			saaskit.SetHTMXRetarget("#notifications"),
//			saaskit.SetHTMXReswapModifiers(
//				saaskit.SwapStrategy(saaskit.SwapAfterBegin),
//				saaskit.SwapSettle(500*time.Millisecond),
//				saaskit.SwapScrollTop(),
//			),
//			saaskit.SetHTMXTrigger("error-shown"),
//		),
//	)
func Templ(component TemplComponent, modifiers ...ResponseModifier) Response {
	return templResponse{
		component: component,
		modifiers: modifiers,
	}
}

// templPartialResponse conditionally renders partial or full component based on request type
type templPartialResponse struct {
	partial TemplComponent
	full    TemplComponent
}

// Render renders either the partial component for HTMX requests or the full component
func (t templPartialResponse) Render(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// For HTMX requests (but not boosted), render only the partial component
	// Boosted requests should receive the full component for proper navigation
	if IsHTMX(r) && !IsHTMXBoosted(r) {
		return t.partial.Render(r.Context(), w)
	}
	// Otherwise, render the full component (for regular requests and boosted HTMX)
	return t.full.Render(r.Context(), w)
}

// TemplPartial creates a response that renders differently for HTMX vs regular requests.
// For HTMX requests (non-boosted), it renders only the partial component for targeted updates.
// For regular requests and HTMX boosted requests, it renders the full component.
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
func TemplPartial(partial, full TemplComponent) Response {
	return templPartialResponse{
		partial: partial,
		full:    full,
	}
}
