package saaskit

import (
	"context"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

// Context wraps http.Request and http.ResponseWriter with context.Context.
// It embeds the request's context and provides access to HTTP components.
type Context interface {
	context.Context
	Request() *http.Request
	ResponseWriter() http.ResponseWriter
	SSE() *datastar.ServerSentEventGenerator
}

// NewContext creates a new Context from HTTP request and response writer.
func NewContext(w http.ResponseWriter, r *http.Request) Context {
	ctx := &httpContext{
		w: w,
		r: r,
	}

	// Initialize SSE if this is a DataStar request
	if IsDataStar(r) {
		ctx.sse = NewDataStarSSE(w, r)
	}

	return ctx
}

// httpContext is the default implementation of Context.
type httpContext struct {
	w   http.ResponseWriter
	r   *http.Request
	sse *datastar.ServerSentEventGenerator
}

func (c *httpContext) Request() *http.Request {
	return c.r
}

func (c *httpContext) ResponseWriter() http.ResponseWriter {
	return c.w
}

func (c *httpContext) SSE() *datastar.ServerSentEventGenerator {
	return c.sse
}

// Deadline returns the time when work done on behalf of this context
// should be canceled.
func (c *httpContext) Deadline() (deadline time.Time, ok bool) {
	return c.r.Context().Deadline()
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled.
func (c *httpContext) Done() <-chan struct{} {
	return c.r.Context().Done()
}

// Err returns a non-nil error value after Done is closed.
func (c *httpContext) Err() error {
	return c.r.Context().Err()
}

// Value returns the value associated with this context for key.
func (c *httpContext) Value(key any) any {
	return c.r.Context().Value(key)
}

// ContextKey is a key for context values.
// It should be created as a package-level variable.
type ContextKey struct{ name string }

// NewContextKey creates a new context key.
// The name should be unique within your application.
//
// Example:
//
//	var userKey = saaskit.NewContextKey("user")
func NewContextKey(name string) *ContextKey {
	return &ContextKey{name}
}

// ContextValue retrieves a typed value from the context.
// Returns the zero value of T if the key is not present or has a different type.
//
// Example:
//
//	var userKey = saaskit.NewContextKey("user")
//
//	// Set value
//	ctx = context.WithValue(ctx, userKey, &User{ID: 123})
//
//	// Get value
//	user := saaskit.ContextValue[*User](ctx, userKey)
//	if user != nil {
//		// Use user
//	}
func ContextValue[T any](ctx context.Context, key any) T {
	val, _ := ctx.Value(key).(T)
	return val
}
