package handler_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/handler"
)

// Mock templ component for testing
type mockComponent struct {
	content string
}

func (m mockComponent) Render(ctx context.Context, w io.Writer) error {
	_, err := w.Write([]byte(m.content))
	return err
}

func TestSSE(t *testing.T) {
	t.Run("requires DataStar request", func(t *testing.T) {
		h := handler.SSE(func(stream handler.StreamContext) error {
			return nil
		})

		// Regular request without DataStar headers
		req := httptest.NewRequest("GET", "/events", nil)
		rec := httptest.NewRecorder()

		err := h.Render(rec, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SSE endpoint requires DataStar connection")
	})

	t.Run("accepts DataStar SSE request", func(t *testing.T) {
		executed := false
		h := handler.SSE(func(stream handler.StreamContext) error {
			executed = true
			return nil
		})

		// DataStar SSE request
		req := httptest.NewRequest("GET", "/events", nil)
		req.Header.Set("Accept", "text/event-stream")
		rec := httptest.NewRecorder()

		err := h.Render(rec, req)
		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("context cancellation stops handler", func(t *testing.T) {
		started := make(chan struct{})
		stopped := make(chan struct{})

		h := handler.SSE(func(stream handler.StreamContext) error {
			close(started)
			<-stream.Done()
			close(stopped)
			return nil
		})

		req := httptest.NewRequest("GET", "/events", nil)
		req.Header.Set("Accept", "text/event-stream")

		// Add cancellable context
		ctx, cancel := context.WithCancel(context.Background())
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()

		// Run handler in goroutine
		done := make(chan error)
		go func() {
			done <- h.Render(rec, req)
		}()

		// Wait for handler to start
		<-started

		// Cancel context
		cancel()

		// Verify handler stopped
		select {
		case <-stopped:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("handler did not stop on context cancellation")
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		expectedErr := assert.AnError
		h := handler.SSE(func(stream handler.StreamContext) error {
			return expectedErr
		})

		req := httptest.NewRequest("GET", "/events", nil)
		req.Header.Set("Accept", "text/event-stream")
		rec := httptest.NewRecorder()

		err := h.Render(rec, req)
		assert.Equal(t, expectedErr, err)
	})
}

func TestSSEWithHandlerFunc(t *testing.T) {
	type testRequest struct {
		Channel string `query:"channel"`
	}

	t.Run("integration with HandlerFunc", func(t *testing.T) {
		h := handler.HandlerFunc[handler.Context, testRequest](
			func(ctx handler.Context, req testRequest) handler.Response {
				return handler.SSE(func(stream handler.StreamContext) error {
					// Verify we have access to request data
					assert.Equal(t, "test", req.Channel)

					// Verify context methods work
					assert.NotNil(t, stream.Request())
					assert.NotNil(t, stream.ResponseWriter())

					return nil
				})
			},
		)

		// Create request with query parameter
		req := httptest.NewRequest("GET", "/events?channel=test", nil)
		req.Header.Set("Accept", "text/event-stream")
		rec := httptest.NewRecorder()

		// Wrap and execute
		wrapped := handler.Wrap(h, handler.WithBinders[handler.Context, testRequest](
			// Mock binder that sets Channel from query
			func(r *http.Request, v any) error {
				if req, ok := v.(*testRequest); ok {
					req.Channel = r.URL.Query().Get("channel")
				}
				return nil
			},
		))

		wrapped(rec, req)
	})
}

func ExampleSSE() {
	// Create an SSE handler for real-time notifications
	notificationHandler := handler.HandlerFunc[handler.Context, struct{}](
		func(ctx handler.Context, _ struct{}) handler.Response {
			return handler.SSE(func(stream handler.StreamContext) error {
				// Simulate receiving notifications
				notifications := make(chan string)
				go func() {
					notifications <- "New message from Alice"
					notifications <- "System update available"
					close(notifications)
				}()

				// Stream notifications to client
				for msg := range notifications {
					// In real usage, this would be a templ component
					err := stream.SendComponent(
						mockComponent{content: "<div>" + msg + "</div>"},
						handler.WithTarget("#notifications"),
						handler.WithPatchMode(handler.PatchAppend),
					)
					if err != nil {
						return err
					}
				}
				return nil
			})
		},
	)

	// The handler can be used with any router
	http.Handle("/notifications", handler.Wrap(notificationHandler))
}
