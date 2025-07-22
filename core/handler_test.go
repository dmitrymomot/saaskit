package core_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	saaskit "github.com/dmitrymomot/saaskit/core"
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

		wrapped := saaskit.Wrap(handler, saaskit.WithContextFactory[saaskit.Context, string](customFactory))

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

		customBinder := func(r *http.Request, v any) error {
			if req, ok := v.(*testRequest); ok {
				req.Name = "bound value"
			}
			return nil
		}

		handler := saaskit.HandlerFunc[saaskit.Context, testRequest](func(ctx saaskit.Context, req testRequest) saaskit.Response {
			assert.Equal(t, "bound value", req.Name)
			return mockResponse{statusCode: http.StatusOK, body: req.Name}
		})

		wrapped := saaskit.Wrap(handler, saaskit.WithBinder[saaskit.Context, testRequest](customBinder))

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "bound value", rec.Body.String())
	})

	t.Run("with multiple binders", func(t *testing.T) {
		type testRequest struct {
			Field1 string
			Field2 string
		}

		binder1 := func(r *http.Request, v any) error {
			if req, ok := v.(*testRequest); ok {
				req.Field1 = "value1"
			}
			return nil
		}

		binder2 := func(r *http.Request, v any) error {
			if req, ok := v.(*testRequest); ok {
				req.Field2 = "value2"
			}
			return nil
		}

		handler := saaskit.HandlerFunc[saaskit.Context, testRequest](func(ctx saaskit.Context, req testRequest) saaskit.Response {
			assert.Equal(t, "value1", req.Field1)
			assert.Equal(t, "value2", req.Field2)
			return mockResponse{statusCode: http.StatusOK, body: req.Field1 + "," + req.Field2}
		})

		wrapped := saaskit.Wrap(handler, saaskit.WithBinders[saaskit.Context, testRequest](binder1, binder2))

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "value1,value2", rec.Body.String())
	})

	t.Run("multiple binders execution order", func(t *testing.T) {
		type testRequest struct {
			Value string
		}

		var executionOrder []string

		binder1 := func(r *http.Request, v any) error {
			executionOrder = append(executionOrder, "binder1")
			if req, ok := v.(*testRequest); ok {
				req.Value = "first"
			}
			return nil
		}

		binder2 := func(r *http.Request, v any) error {
			executionOrder = append(executionOrder, "binder2")
			if req, ok := v.(*testRequest); ok {
				// This should overwrite the value set by binder1
				req.Value = "second"
			}
			return nil
		}

		handler := saaskit.HandlerFunc[saaskit.Context, testRequest](func(ctx saaskit.Context, req testRequest) saaskit.Response {
			return mockResponse{statusCode: http.StatusOK, body: req.Value}
		})

		wrapped := saaskit.Wrap(handler, saaskit.WithBinders[saaskit.Context, testRequest](binder1, binder2))

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, []string{"binder1", "binder2"}, executionOrder)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "second", rec.Body.String()) // Should have the value from the second binder
	})

	t.Run("multiple binders with error in chain", func(t *testing.T) {
		type testRequest struct {
			Field1 string
			Field2 string
		}

		binderErr := errors.New("binder2 failed")
		var binder1Called, binder2Called bool

		binder1 := func(r *http.Request, v any) error {
			binder1Called = true
			if req, ok := v.(*testRequest); ok {
				req.Field1 = "value1"
			}
			return nil
		}

		binder2 := func(r *http.Request, v any) error {
			binder2Called = true
			return binderErr // This binder fails
		}

		handler := saaskit.HandlerFunc[saaskit.Context, testRequest](func(ctx saaskit.Context, req testRequest) saaskit.Response {
			t.Fatal("handler should not be called when binder fails")
			return nil
		})

		var capturedErr error
		errorHandler := func(ctx saaskit.Context, err error) {
			capturedErr = err
			ctx.ResponseWriter().WriteHeader(http.StatusBadRequest)
			ctx.ResponseWriter().Write([]byte("binding error"))
		}

		wrapped := saaskit.Wrap(handler,
			saaskit.WithBinders[saaskit.Context, testRequest](binder1, binder2),
			saaskit.WithErrorHandler[saaskit.Context, testRequest](errorHandler),
		)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.True(t, binder1Called)
		assert.True(t, binder2Called)
		assert.Equal(t, binderErr, capturedErr)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "binding error", rec.Body.String())
	})

	t.Run("with binder error and custom error handler", func(t *testing.T) {
		binderErr := errors.New("binding failed")
		errorHandlerCalled := false

		customBinder := func(r *http.Request, v any) error {
			return binderErr
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
			saaskit.WithBinder[saaskit.Context, string](customBinder),
			saaskit.WithErrorHandler[saaskit.Context, string](customErrorHandler),
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
			saaskit.WithBinder[saaskit.Context, string](nil),
			saaskit.WithErrorHandler[saaskit.Context, string](nil),
			saaskit.WithContextFactory[saaskit.Context, string](nil),
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

		wrapped := saaskit.Wrap(handler, saaskit.WithErrorHandler[saaskit.Context, string](customErrorHandler))

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
			saaskit.WithContextFactory[customContext, string](newTestCustomContext),
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

		customBinder := func(r *http.Request, v any) error {
			// Note: Can't access custom context in binder anymore
			// This is a tradeoff for simpler API
			if req, ok := v.(*userRequest); ok {
				req.Name = "User test-user-123"
			}
			return nil
		}

		handler := saaskit.HandlerFunc[customContext, userRequest](func(ctx customContext, req userRequest) saaskit.Response {
			return mockResponse{statusCode: http.StatusOK, body: req.Name}
		})

		wrapped := saaskit.Wrap(handler,
			saaskit.WithContextFactory[customContext, userRequest](newTestCustomContext),
			saaskit.WithBinder[customContext, userRequest](customBinder),
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
			saaskit.WithContextFactory[saaskit.Context, string](customFactory),
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

func TestWrapWithDecorators(t *testing.T) {
	t.Run("single decorator", func(t *testing.T) {
		var decoratorCalled bool
		decorator := func(next saaskit.HandlerFunc[saaskit.Context, string]) saaskit.HandlerFunc[saaskit.Context, string] {
			return func(ctx saaskit.Context, req string) saaskit.Response {
				decoratorCalled = true
				return next(ctx, req)
			}
		}

		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			return mockResponse{statusCode: http.StatusOK, body: "success"}
		})

		wrapped := saaskit.Wrap(handler,
			saaskit.WithDecorators(decorator),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.True(t, decoratorCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "success", rec.Body.String())
	})

	t.Run("multiple decorators order", func(t *testing.T) {
		var order []string

		decorator1 := func(next saaskit.HandlerFunc[saaskit.Context, string]) saaskit.HandlerFunc[saaskit.Context, string] {
			return func(ctx saaskit.Context, req string) saaskit.Response {
				order = append(order, "decorator1-before")
				resp := next(ctx, req)
				order = append(order, "decorator1-after")
				return resp
			}
		}

		decorator2 := func(next saaskit.HandlerFunc[saaskit.Context, string]) saaskit.HandlerFunc[saaskit.Context, string] {
			return func(ctx saaskit.Context, req string) saaskit.Response {
				order = append(order, "decorator2-before")
				resp := next(ctx, req)
				order = append(order, "decorator2-after")
				return resp
			}
		}

		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			order = append(order, "handler")
			return mockResponse{statusCode: http.StatusOK, body: "success"}
		})

		wrapped := saaskit.Wrap(handler,
			saaskit.WithDecorators(decorator1, decorator2),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		// Verify order: decorator1 wraps decorator2 wraps handler
		expectedOrder := []string{
			"decorator1-before",
			"decorator2-before",
			"handler",
			"decorator2-after",
			"decorator1-after",
		}
		assert.Equal(t, expectedOrder, order)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("decorator modifying response", func(t *testing.T) {
		decorator := func(next saaskit.HandlerFunc[saaskit.Context, string]) saaskit.HandlerFunc[saaskit.Context, string] {
			return func(ctx saaskit.Context, req string) saaskit.Response {
				// Call original handler but ignore response
				_ = next(ctx, req)
				// Return modified response
				return mockResponse{statusCode: http.StatusAccepted, body: "modified"}
			}
		}

		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			return mockResponse{statusCode: http.StatusOK, body: "original"}
		})

		wrapped := saaskit.Wrap(handler,
			saaskit.WithDecorators(decorator),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusAccepted, rec.Code)
		assert.Equal(t, "modified", rec.Body.String())
	})

	t.Run("decorator short-circuiting", func(t *testing.T) {
		handlerCalled := false
		decorator := func(next saaskit.HandlerFunc[saaskit.Context, string]) saaskit.HandlerFunc[saaskit.Context, string] {
			return func(ctx saaskit.Context, req string) saaskit.Response {
				// Don't call next, return early
				return mockResponse{statusCode: http.StatusUnauthorized, body: "unauthorized"}
			}
		}

		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			handlerCalled = true
			return mockResponse{statusCode: http.StatusOK, body: "success"}
		})

		wrapped := saaskit.Wrap(handler,
			saaskit.WithDecorators(decorator),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "unauthorized", rec.Body.String())
	})

	t.Run("decorators with custom context", func(t *testing.T) {
		type userRequest struct {
			Name string
		}

		decorator := func(next saaskit.HandlerFunc[customContext, userRequest]) saaskit.HandlerFunc[customContext, userRequest] {
			return func(ctx customContext, req userRequest) saaskit.Response {
				// Access custom context method in decorator
				userID := ctx.UserID()
				req.Name = req.Name + " (user: " + userID + ")"
				return next(ctx, req)
			}
		}

		customBinder := func(r *http.Request, v any) error {
			if req, ok := v.(*userRequest); ok {
				req.Name = "John"
			}
			return nil
		}

		handler := saaskit.HandlerFunc[customContext, userRequest](func(ctx customContext, req userRequest) saaskit.Response {
			return mockResponse{statusCode: http.StatusOK, body: req.Name}
		})

		wrapped := saaskit.Wrap(handler,
			saaskit.WithContextFactory[customContext, userRequest](newTestCustomContext),
			saaskit.WithBinder[customContext, userRequest](customBinder),
			saaskit.WithDecorators(decorator),
		)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "John (user: test-user-123)", rec.Body.String())
	})

	t.Run("decorator error handling", func(t *testing.T) {
		decorator := func(next saaskit.HandlerFunc[saaskit.Context, string]) saaskit.HandlerFunc[saaskit.Context, string] {
			return func(ctx saaskit.Context, req string) saaskit.Response {
				return mockResponse{renderErr: errors.New("decorator error")}
			}
		}

		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			return mockResponse{statusCode: http.StatusOK, body: "success"}
		})

		wrapped := saaskit.Wrap(handler,
			saaskit.WithDecorators(decorator),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "decorator error")
	})

	t.Run("multiple WithDecorators calls", func(t *testing.T) {
		var order []string

		decorator1 := func(next saaskit.HandlerFunc[saaskit.Context, string]) saaskit.HandlerFunc[saaskit.Context, string] {
			return func(ctx saaskit.Context, req string) saaskit.Response {
				order = append(order, "decorator1")
				return next(ctx, req)
			}
		}

		decorator2 := func(next saaskit.HandlerFunc[saaskit.Context, string]) saaskit.HandlerFunc[saaskit.Context, string] {
			return func(ctx saaskit.Context, req string) saaskit.Response {
				order = append(order, "decorator2")
				return next(ctx, req)
			}
		}

		decorator3 := func(next saaskit.HandlerFunc[saaskit.Context, string]) saaskit.HandlerFunc[saaskit.Context, string] {
			return func(ctx saaskit.Context, req string) saaskit.Response {
				order = append(order, "decorator3")
				return next(ctx, req)
			}
		}

		handler := saaskit.HandlerFunc[saaskit.Context, string](func(ctx saaskit.Context, req string) saaskit.Response {
			order = append(order, "handler")
			return mockResponse{statusCode: http.StatusOK, body: "success"}
		})

		wrapped := saaskit.Wrap(handler,
			saaskit.WithDecorators(decorator1, decorator2),
			saaskit.WithDecorators(decorator3), // Multiple calls should append
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped(rec, req)

		// All decorators should be applied in order
		expectedOrder := []string{"decorator1", "decorator2", "decorator3", "handler"}
		assert.Equal(t, expectedOrder, order)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
