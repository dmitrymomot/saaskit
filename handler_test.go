package saaskit_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit"
)

// Mock response for testing
type mockResponse struct {
	statusCode int
	body       string
	renderErr  error
}

func (m mockResponse) Render(w http.ResponseWriter, r *http.Request) error {
	if m.renderErr != nil {
		return m.renderErr
	}
	w.WriteHeader(m.statusCode)
	w.Write([]byte(m.body))
	return nil
}

func TestWrap(t *testing.T) {
	t.Run("basic handler without options", func(t *testing.T) {
		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			assert.NotNil(t, ctx)
			assert.Equal(t, "", req) // zero value
			return mockResponse{statusCode: http.StatusOK, body: "success"}
		})

		wrapped := saaskit.Wrap(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "success", rec.Body.String())
	})

	t.Run("handler with render error", func(t *testing.T) {
		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			return mockResponse{renderErr: errors.New("render failed")}
		})

		wrapped := saaskit.Wrap(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "render failed")
	})

	t.Run("handler returns nil response", func(t *testing.T) {
		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			return nil
		})

		wrapped := saaskit.Wrap(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "handler returned nil response")
	})

	t.Run("with custom context factory", func(t *testing.T) {
		customContextCreated := false
		customFactory := func(w http.ResponseWriter, r *http.Request) saaskit.Context {
			customContextCreated = true
			return saaskit.NewContext(w, r)
		}

		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			return mockResponse{statusCode: http.StatusOK, body: "ok"}
		})

		wrapped := saaskit.Wrap(handler, saaskit.WithContextFactory(customFactory))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.True(t, customContextCreated)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("with custom binder", func(t *testing.T) {
		type testRequest struct {
			Name string
		}

		customBinder := &mockBinder[saaskit.Context]{
			bindFunc: func(ctx saaskit.Context, v any) error {
				if req, ok := v.(*testRequest); ok {
					req.Name = "bound value"
				}
				return nil
			},
		}

		handler := saaskit.HandlerFunc[saaskit.Context, testRequest](func(ctx saaskit.Context, req testRequest) saaskit.Response {
			assert.Equal(t, "bound value", req.Name)
			return mockResponse{statusCode: http.StatusOK, body: req.Name}
		})

		wrapped := saaskit.Wrap(handler, saaskit.WithBinder(customBinder))

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "bound value", rec.Body.String())
	})

	t.Run("with binder error and custom error handler", func(t *testing.T) {
		binderErr := errors.New("binding failed")
		errorHandlerCalled := false

		customBinder := &mockBinder[saaskit.Context]{
			bindFunc: func(ctx saaskit.Context, v any) error {
				return binderErr
			},
		}

		customErrorHandler := func(ctx saaskit.Context, err error) {
			errorHandlerCalled = true
			assert.Equal(t, binderErr, err)
			ctx.ResponseWriter().WriteHeader(http.StatusBadRequest)
			ctx.ResponseWriter().Write([]byte("custom error: " + err.Error()))
		}

		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			t.Fatal("handler should not be called on bind error")
			return nil
		})

		wrapped := saaskit.Wrap(handler,
			saaskit.WithBinder(customBinder),
			saaskit.WithErrorHandler(customErrorHandler),
		)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.True(t, errorHandlerCalled)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "custom error: binding failed", rec.Body.String())
	})

	t.Run("with nil options", func(t *testing.T) {
		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			return mockResponse{statusCode: http.StatusOK, body: "ok"}
		})

		// Should not panic with nil options
		wrapped := saaskit.Wrap(handler,
			saaskit.WithBinder[saaskit.Context](nil),
			saaskit.WithErrorHandler[saaskit.Context](nil),
			saaskit.WithContextFactory[saaskit.Context](nil),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		require.NotPanics(t, func() {
			wrapped(rec, req)
		})

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("nil response with custom error handler", func(t *testing.T) {
		var capturedErr error
		customErrorHandler := func(ctx saaskit.Context, err error) {
			capturedErr = err
			ctx.ResponseWriter().WriteHeader(http.StatusServiceUnavailable)
			ctx.ResponseWriter().Write([]byte("service unavailable"))
		}

		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			return nil
		})

		wrapped := saaskit.Wrap(handler, saaskit.WithErrorHandler(customErrorHandler))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Equal(t, "service unavailable", rec.Body.String())
		assert.NotNil(t, capturedErr)
		assert.Equal(t, "handler returned nil response", capturedErr.Error())
	})

	t.Run("handler returns HTTPError", func(t *testing.T) {
		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			return mockResponse{renderErr: saaskit.ErrNotFound}
		})

		wrapped := saaskit.Wrap(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "not_found")
	})

	t.Run("handler returns wrapped HTTPError", func(t *testing.T) {
		// Wrap the HTTPError to test errors.As functionality
		wrappedErr := fmt.Errorf("validation failed: %w", saaskit.ErrUnprocessableEntity)

		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			return mockResponse{renderErr: wrappedErr}
		})

		wrapped := saaskit.Wrap(handler)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "unprocessable_entity")
	})

	t.Run("handler returns non-HTTPError", func(t *testing.T) {
		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			return mockResponse{renderErr: errors.New("database connection failed")}
		})

		wrapped := saaskit.Wrap(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		// Should fallback to 500 with the actual error message
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "database connection failed")
	})
}

// mockBinder for testing
type mockBinder[C saaskit.Context] struct {
	bindFunc func(ctx C, v any) error
}

func (m *mockBinder[C]) Bind(ctx C, v any) error {
	return m.bindFunc(ctx, v)
}

// Custom context for testing
type customContext interface {
	saaskit.Context
	UserID() string
}

type testCustomContext struct {
	saaskit.Context
	userID string
}

func (c *testCustomContext) UserID() string {
	return c.userID
}

func newTestCustomContext(w http.ResponseWriter, r *http.Request) customContext {
	return &testCustomContext{
		Context: saaskit.NewContext(w, r),
		userID:  "test-user-123",
	}
}

func TestWrapWithCustomContext(t *testing.T) {
	t.Run("handler with custom context", func(t *testing.T) {
		handler := saaskit.HandlerFunc[customContext, string](func(ctx customContext, req string) saaskit.Response {
			// Direct access to custom context methods
			userID := ctx.UserID()
			assert.Equal(t, "test-user-123", userID)
			return mockResponse{statusCode: http.StatusOK, body: userID}
		})

		wrapped := saaskit.Wrap(handler,
			saaskit.WithContextFactory(newTestCustomContext),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "test-user-123", rec.Body.String())
	})

	t.Run("custom context with binder", func(t *testing.T) {
		type userRequest struct {
			Name string
		}

		customBinder := &mockBinder[customContext]{
			bindFunc: func(ctx customContext, v any) error {
				// Can access custom context in binder
				userID := ctx.UserID()
				assert.Equal(t, "test-user-123", userID)

				if req, ok := v.(*userRequest); ok {
					req.Name = "User " + userID
				}
				return nil
			},
		}

		handler := saaskit.HandlerFunc[customContext, userRequest](func(ctx customContext, req userRequest) saaskit.Response {
			return mockResponse{statusCode: http.StatusOK, body: req.Name}
		})

		wrapped := saaskit.Wrap(handler,
			saaskit.WithContextFactory(newTestCustomContext),
			saaskit.WithBinder(customBinder),
		)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "User test-user-123", rec.Body.String())
	})

	t.Run("custom context without factory panics", func(t *testing.T) {
		handler := saaskit.HandlerFunc[customContext, string](func(ctx customContext, req string) saaskit.Response {
			// This should never be called
			t.Fatal("handler should not be called")
			return nil
		})

		// Don't provide WithContextFactory - should panic
		wrapped := saaskit.Wrap(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		assert.Panics(t, func() {
			wrapped(rec, req)
		}, "should panic when custom context is used without factory")
	})
}

func TestDefaultContextFactory(t *testing.T) {
	t.Run("standard context uses default factory", func(t *testing.T) {
		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			// Verify we got a valid context
			assert.NotNil(t, ctx)
			assert.NotNil(t, ctx.Request())
			assert.NotNil(t, ctx.ResponseWriter())
			return mockResponse{statusCode: http.StatusOK, body: "ok"}
		})

		// No WithContextFactory provided - should use default
		wrapped := saaskit.Wrap(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})

	t.Run("overriding default factory works", func(t *testing.T) {
		customFactoryCalled := false
		customFactory := func(w http.ResponseWriter, r *http.Request) saaskit.Context {
			customFactoryCalled = true
			return saaskit.NewContext(w, r)
		}

		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			return mockResponse{statusCode: http.StatusOK, body: "ok"}
		})

		// Provide custom factory even for standard context
		wrapped := saaskit.Wrap(handler,
			saaskit.WithContextFactory(customFactory),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.True(t, customFactoryCalled, "custom factory should be called")
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("panic message contains helpful text", func(t *testing.T) {
		handler := saaskit.HandlerFunc[customContext, string](func(ctx customContext, req string) saaskit.Response {
			return nil
		})

		wrapped := saaskit.Wrap(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		// Capture panic message
		var panicMsg string
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicMsg = fmt.Sprintf("%v", r)
				}
			}()
			wrapped(rec, req)
		}()

		assert.Contains(t, panicMsg, "cannot use default context factory with custom context type")
		assert.Contains(t, panicMsg, "WithContextFactory")
	})
}
