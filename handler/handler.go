package handler

import (
	"errors"
	"net/http"

	"github.com/dmitrymomot/saaskit/pkg/binder"
)

// HandlerFunc provides type-safe HTTP request handling with custom context support.
// C must implement the Context interface, R can be any request type.
//
// Example with standard context:
//
//	handler := saaskit.HandlerFunc[saaskit.Context, CreateUserRequest](
//		func(ctx saaskit.Context, req CreateUserRequest) saaskit.Response {
//			user := createUser(req.Name, req.Email)
//			return JSONResponse(user)
//		},
//	)
//
// Example with custom context:
//
//	handler := saaskit.HandlerFunc[AppContext, CreateUserRequest](
//		func(ctx AppContext, req CreateUserRequest) saaskit.Response {
//			userID := ctx.UserID() // Direct access to custom methods
//			return JSONResponse(user)
//		},
//	)
type HandlerFunc[C Context, R any] func(ctx C, req R) Response

// Response renders itself to an http.ResponseWriter.
// Implementations should set headers, status code, and write body.
// Errors are handled by the framework (returns 500).
type Response interface {
	Render(w http.ResponseWriter, r *http.Request) error
}

// Bind parses HTTP requests into typed values.
type Bind func(r *http.Request, v any) error

// ErrorHandler handles errors from binding or rendering.
type ErrorHandler[C Context] func(ctx C, err error)

// Decorator wraps a HandlerFunc to add cross-cutting functionality.
// Decorators are applied in order, with the first decorator in the list
// being the outermost wrapper.
//
// Example logger decorator:
//
//	func Logger[C Context, R any]() Decorator[C, R] {
//		return func(next HandlerFunc[C, R]) HandlerFunc[C, R] {
//			return func(ctx C, req R) Response {
//				log.Printf("Request: %+v", req)
//				resp := next(ctx, req)
//				log.Printf("Response complete")
//				return resp
//			}
//		}
//	}
type Decorator[C Context, R any] func(HandlerFunc[C, R]) HandlerFunc[C, R]

// WrapOption configures the Wrap function.
type WrapOption[C Context, R any] func(*wrapConfig[C, R])

// wrapConfig holds configuration for Wrap.
type wrapConfig[C Context, R any] struct {
	binders        []Bind
	errorHandler   ErrorHandler[C]
	contextFactory func(http.ResponseWriter, *http.Request) C
	decorators     []Decorator[C, R]
}

// WithBinder sets a custom request binder.
// For backward compatibility, this converts to a single-item binders slice.
func WithBinder[C Context, R any](b Bind) WrapOption[C, R] {
	return func(c *wrapConfig[C, R]) {
		if b != nil {
			c.binders = []Bind{b}
		}
	}
}

// WithBinders sets multiple request binders that will be applied in order.
// Each binder should process only its specific struct tags.
//
// Example:
//
//	http.HandleFunc("/users/:id", saaskit.Wrap(handler,
//		saaskit.WithBinders(
//			binder.Path(),   // processes path: tags
//			binder.Query(),  // processes query: tags
//			binder.Form(),   // processes form: tags
//			binder.File(),   // processes file: tags
//		),
//	))
func WithBinders[C Context, R any](binders ...Bind) WrapOption[C, R] {
	return func(c *wrapConfig[C, R]) {
		c.binders = append(c.binders, binders...)
	}
}

// WithErrorHandler sets a custom error handler.
func WithErrorHandler[C Context, R any](h ErrorHandler[C]) WrapOption[C, R] {
	return func(c *wrapConfig[C, R]) {
		if h != nil {
			c.errorHandler = h
		}
	}
}

// WithContextFactory sets a custom context factory.
func WithContextFactory[C Context, R any](f func(http.ResponseWriter, *http.Request) C) WrapOption[C, R] {
	return func(c *wrapConfig[C, R]) {
		if f != nil {
			c.contextFactory = f
		}
	}
}

// WithDecorators adds decorators to wrap the handler.
// Decorators are applied in order, with the first decorator being the outermost.
//
// Example:
//
//	http.HandleFunc("/users", saaskit.Wrap(handler,
//		saaskit.WithDecorators(
//			Logger[saaskit.Context, CreateUserRequest](),
//			RequireAuth[saaskit.Context, CreateUserRequest](),
//		),
//	))
func WithDecorators[C Context, R any](decorators ...Decorator[C, R]) WrapOption[C, R] {
	return func(c *wrapConfig[C, R]) {
		c.decorators = append(c.decorators, decorators...)
	}
}

// defaultErrorHandler provides standard HTTP error responses.
// It checks if the error is an HTTPError and uses its status code,
// otherwise defaults to 500 Internal Server Error.
func defaultErrorHandler[C Context](ctx C, err error) {
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		http.Error(ctx.ResponseWriter(), httpErr.Key, httpErr.Code)
		return
	}
	http.Error(ctx.ResponseWriter(), err.Error(), http.StatusInternalServerError)
}

// Wrap converts a typed HandlerFunc to http.HandlerFunc.
//
// Usage with standard context:
//
//	handler := saaskit.HandlerFunc[saaskit.Context, CreateUserRequest](...)
//	http.HandleFunc("/users", saaskit.Wrap(handler))
//
// Usage with custom context:
//
//	handler := saaskit.HandlerFunc[AppContext, CreateUserRequest](...)
//	http.HandleFunc("/users", saaskit.Wrap(handler,
//		saaskit.WithContextFactory(NewAppContext),
//	))
//
// With options:
//
//	http.HandleFunc("/users", saaskit.Wrap(handler,
//		saaskit.WithBinder(customBinder),
//		saaskit.WithErrorHandler(customErrorHandler),
//		saaskit.WithContextFactory(customContextFactory),
//		saaskit.WithDecorators(Logger(), RequireAuth()),
//	))
func Wrap[C Context, R any](h HandlerFunc[C, R], opts ...WrapOption[C, R]) http.HandlerFunc {
	// Initialize config with defaults
	cfg := &wrapConfig[C, R]{
		errorHandler: defaultErrorHandler[C],
	}

	// Set default context factory if none provided and C can be created with NewContext
	if cfg.contextFactory == nil {
		// Try to use NewContext as default factory
		cfg.contextFactory = func(w http.ResponseWriter, r *http.Request) C {
			ctx := NewContext(w, r)
			if c, ok := any(ctx).(C); ok {
				return c
			}
			// This will panic if C is not compatible with the default Context
			panic("cannot use default context factory with custom context type - provide WithContextFactory")
		}
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Apply decorators in reverse order so first decorator is outermost
	finalHandler := h
	for i := len(cfg.decorators) - 1; i >= 0; i-- {
		finalHandler = cfg.decorators[i](finalHandler)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := cfg.contextFactory(w, r)

		var req R

		// Apply binders in order - skip those that are not applicable
		for _, bind := range cfg.binders {
			if err := bind(r, &req); err != nil {
				// Skip binders that are not applicable to this request
				if errors.Is(err, binder.ErrBinderNotApplicable) {
					continue
				}
				cfg.errorHandler(ctx, err)
				return
			}
		}

		response := finalHandler(ctx, req)
		if response == nil {
			cfg.errorHandler(ctx, ErrNilResponse)
			return
		}
		if err := response.Render(w, r); err != nil {
			cfg.errorHandler(ctx, err)
		}
	}
}
