package handler

import (
	"encoding/json"

	"github.com/starfederation/datastar-go/datastar"
)

// StreamContext extends Context with SSE streaming capabilities.
// It provides methods to send components and signals through an established SSE connection.
type StreamContext interface {
	Context

	// SendComponent sends a templ component with rendering options.
	// This is the primary method for updating UI elements via SSE.
	//
	// Example:
	//
	//	err := stream.SendComponent(
	//		templates.ChatMessage(msg),
	//		handler.WithTarget("#chat-messages"),
	//		handler.WithPatchMode(handler.PatchAppend),
	//	)
	SendComponent(component TemplComponent, opts ...TemplOption) error

	// SendMultiple sends multiple components in a single batch.
	// This is more efficient than calling SendComponent multiple times.
	//
	// Example:
	//
	//	err := stream.SendMultiple(
	//		handler.Patch(templates.UserCount(count), handler.WithTarget("#user-count")),
	//		handler.Patch(templates.NewNotification(msg), handler.WithTarget("#notifications")),
	//	)
	SendMultiple(patches ...TemplPatch) error

	// SendSignal updates a single frontend signal/state value.
	// Signals are used for reactive UI updates without replacing DOM elements.
	//
	// Example:
	//
	//	err := stream.SendSignal("isLoading", false)
	SendSignal(name string, value any) error

	// SendSignals updates multiple frontend signals at once.
	// This is more efficient than calling SendSignal multiple times.
	//
	// Example:
	//
	//	err := stream.SendSignals(map[string]any{
	//		"progress": 75,
	//		"status": "processing",
	//		"canSubmit": false,
	//	})
	SendSignals(signals map[string]any) error
}

// streamContext implements StreamContext by wrapping a base Context
// with SSE streaming capabilities.
type streamContext struct {
	Context
	sse *datastar.ServerSentEventGenerator
}

// SendComponent sends a single component through SSE.
func (c *streamContext) SendComponent(component TemplComponent, opts ...TemplOption) error {
	if c.sse == nil {
		return ErrSSENotInitialized
	}
	return c.sse.PatchElementTempl(component, opts...)
}

// SendMultiple sends multiple components efficiently.
func (c *streamContext) SendMultiple(patches ...TemplPatch) error {
	if c.sse == nil {
		return ErrSSENotInitialized
	}
	for _, patch := range patches {
		if err := c.sse.PatchElementTempl(patch.Component, patch.Options...); err != nil {
			return err
		}
	}
	return nil
}

// SendSignal updates a single signal value.
func (c *streamContext) SendSignal(name string, value any) error {
	if c.sse == nil {
		return ErrSSENotInitialized
	}
	data, err := json.Marshal(map[string]any{name: value})
	if err != nil {
		return err
	}
	return c.sse.PatchSignals(data)
}

// SendSignals updates multiple signals at once.
func (c *streamContext) SendSignals(signals map[string]any) error {
	if c.sse == nil {
		return ErrSSENotInitialized
	}
	data, err := json.Marshal(signals)
	if err != nil {
		return err
	}
	return c.sse.PatchSignals(data)
}
