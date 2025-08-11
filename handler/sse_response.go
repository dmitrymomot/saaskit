package handler

import (
	"net/http"
)

// SSEHandler is a function that handles Server-Sent Events streaming.
// It receives a StreamContext with methods for sending components and signals.
//
// The handler should run for the lifetime of the SSE connection, typically
// using a loop that listens for events and sends updates. The connection
// will be closed when the handler returns or the client disconnects.
//
// Example:
//
//	handler.SSE(func(stream handler.StreamContext) error {
//		ticker := time.NewTicker(time.Second)
//		defer ticker.Stop()
//
//		for {
//			select {
//			case <-stream.Done():
//				return nil
//			case t := <-ticker.C:
//				err := stream.SendComponent(
//					templates.TimeDisplay(t),
//					handler.WithTarget("#time"),
//				)
//				if err != nil {
//					return err
//				}
//			}
//		}
//	})
type SSEHandler func(ctx StreamContext) error

// sseResponse implements Response for Server-Sent Events.
type sseResponse struct {
	handler SSEHandler
}

// Render validates DataStar connection and executes the SSE handler.
func (s sseResponse) Render(w http.ResponseWriter, r *http.Request) error {
	// Verify this is a DataStar SSE request
	if !IsDataStar(r) {
		return NewHTTPError(http.StatusBadRequest, "SSE endpoint requires DataStar connection")
	}

	// Create base context with SSE already initialized
	base := NewContext(w, r)
	if base.SSE() == nil {
		return ErrSSENotInitialized
	}

	// Wrap with streaming capabilities
	ctx := &streamContext{
		Context: base,
		sse:     base.SSE(),
	}

	// Run the handler with streaming context
	return s.handler(ctx)
}

// SSE creates a new SSE response that runs the given handler.
// The handler receives a StreamContext with methods for sending
// components and signals through the SSE connection.
//
// This response type is designed to work with DataStar's SSE
// connection that is established when the page loads. It allows
// handlers to push real-time updates to the client.
//
// Example usage in a handler:
//
//	handler.HandlerFunc[handler.Context, SubscribeRequest](
//		func(ctx handler.Context, req SubscribeRequest) handler.Response {
//			return handler.SSE(func(stream handler.StreamContext) error {
//				// Subscribe to events
//				events := eventBus.Subscribe(req.Channel)
//				defer eventBus.Unsubscribe(req.Channel)
//
//				// Stream events to client
//				for event := range events {
//					err := stream.SendComponent(
//						templates.EventCard(event),
//						handler.WithTarget("#events"),
//						handler.WithPatchMode(handler.PatchAppend),
//					)
//					if err != nil {
//						return err
//					}
//				}
//				return nil
//			})
//		},
//	)
func SSE(handler SSEHandler) Response {
	return sseResponse{handler: handler}
}
