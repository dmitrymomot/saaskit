# handler

Type-safe HTTP handler abstraction with built-in support for JSON, HTML templates, and DataStar responses.

## Overview

The handler package provides a framework-agnostic approach to building HTTP handlers with strong typing, flexible response formats, and seamless integration with DataStar for reactive web applications. It simplifies request handling while maintaining type safety and composability.

## Internal Usage

This package is internal to the project and provides HTTP handler abstractions for building web endpoints with type-safe request/response handling and multiple response formats.

## Features

- Type-safe request handling with automatic binding
- Multiple response formats (JSON, HTML, redirects)
- Built-in DataStar/SSE support for reactive UIs
- Real-time streaming with SSE response type
- Context abstraction with custom extensions
- Decorator pattern for cross-cutting concerns
- Comprehensive HTTP error types with i18n support

## Usage

### Basic Example

```go
import "github.com/dmitrymomot/saaskit/handler"

// Define your request type
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Create a typed handler
h := handler.HandlerFunc[handler.Context, CreateUserRequest](
    func(ctx handler.Context, req CreateUserRequest) handler.Response {
        user := createUser(req.Name, req.Email)
        return handler.JSON(user) // Returns JSON response
    },
)

// Wrap for standard http.HandlerFunc
http.HandleFunc("/users", handler.Wrap(h))
```

### Real-time SSE Streaming

```go
// Create an SSE handler for real-time updates
chatHandler := handler.HandlerFunc[handler.Context, SubscribeRequest](
    func(ctx handler.Context, req SubscribeRequest) handler.Response {
        return handler.SSE(func(stream handler.StreamContext) error {
            // Subscribe to chat messages
            messages := chatRoom.Subscribe(req.RoomID, stream.Done())
            defer chatRoom.Unsubscribe(req.RoomID)
            
            // Stream messages to client
            for msg := range messages {
                err := stream.SendComponent(
                    templates.ChatMessage(msg),
                    handler.WithTarget("#chat-messages"),
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

// Mount the SSE endpoint
http.HandleFunc("/chat/:roomId/subscribe", handler.Wrap(chatHandler))
```

### Additional Usage Scenarios

```go
// Custom context with authentication
type AppContext interface {
    handler.Context
    UserID() string
}

// Handler with custom context
authHandler := handler.HandlerFunc[AppContext, UpdateRequest](
    func(ctx AppContext, req UpdateRequest) handler.Response {
        userID := ctx.UserID() // Access custom context methods

        // Return different response types
        if req.Format == "html" {
            return handler.Templ(templates.UserProfile(user))
        }
        return handler.JSON(user)
    },
)

// Use with custom context factory
http.HandleFunc("/profile", handler.Wrap(authHandler,
    handler.WithContextFactory(NewAppContext),
))

// Multiple response patches for DataStar
complexHandler := handler.HandlerFunc[handler.Context, DeleteRequest](
    func(ctx handler.Context, req DeleteRequest) handler.Response {
        deleteItem(req.ID)

        // Update multiple UI sections
        return handler.TemplMulti(
            handler.Patch(templates.ItemList(items),
                handler.WithTarget("#item-list")),
            handler.Patch(templates.SuccessNotification("Item deleted"),
                handler.WithTarget("#notifications"),
                handler.WithPatchMode(handler.PatchPrepend)),
        )
    },
)
```

### Error Handling

```go
// Handler that returns errors
handler := handler.HandlerFunc[handler.Context, LoginRequest](
    func(ctx handler.Context, req LoginRequest) handler.Response {
        if req.Email == "" {
            // Return validation error
            ve := handler.NewValidationError()
            ve.Add("email", "Email is required")
            return handler.JSONError(ve)
        }

        user, err := findUser(req.Email)
        if err != nil {
            // Return HTTP error
            return handler.JSONError(handler.ErrNotFound)
        }

        return handler.JSON(user)
    },
)

// Custom error handler
customErrorHandler := func(ctx handler.Context, err error) {
    // Log error
    log.Printf("Handler error: %v", err)

    // Send custom response
    handler.JSONError(err).Render(ctx.ResponseWriter(), ctx.Request())
}

http.HandleFunc("/login", handler.Wrap(handler,
    handler.WithErrorHandler(customErrorHandler),
))
```

## Best Practices

### Integration Guidelines

- Use typed handlers for all endpoints to ensure compile-time safety
- Leverage custom contexts for request-scoped data like authentication
- Choose appropriate response types based on client capabilities
- Implement proper error handling with meaningful error codes

### Project-Specific Considerations

- Always check for DataStar requests when building interactive UIs
- Use ValidationError for field-level validation feedback
- Leverage decorators for common concerns like logging and auth
- Keep handlers focused on single responsibilities

## API Reference

### Configuration Variables

```go
// DataStar constants
const DataStarAcceptHeader = "text/event-stream"
const DataStarQueryParam = "datastar"

// Patch mode aliases
const PatchOuter = datastar.ElementPatchModeOuter
const PatchInner = datastar.ElementPatchModeInner
const PatchReplace = datastar.ElementPatchModeReplace
const PatchRemove = datastar.ElementPatchModeRemove
const PatchAppend = datastar.ElementPatchModeAppend
const PatchPrepend = datastar.ElementPatchModePrepend
const PatchBefore = datastar.ElementPatchModeBefore
const PatchAfter = datastar.ElementPatchModeAfter

// Package errors
var ErrNilResponse = errors.New("handler returned nil response")
var ErrSSENotInitialized = errors.New("SSE not initialized for this request")

// HTTP errors (4xx)
var ErrBadRequest = HTTPError{Code: 400, Key: "bad_request"}
var ErrUnauthorized = HTTPError{Code: 401, Key: "unauthorized"}
var ErrForbidden = HTTPError{Code: 403, Key: "forbidden"}
var ErrNotFound = HTTPError{Code: 404, Key: "not_found"}
// ... and more

// HTTP errors (5xx)
var ErrInternalServerError = HTTPError{Code: 500, Key: "internal_server_error"}
var ErrServiceUnavailable = HTTPError{Code: 503, Key: "service_unavailable"}
// ... and more
```

### Types

```go
// Core handler type with generics
type HandlerFunc[C Context, R any] func(ctx C, req R) Response

// Response interface for all response types
type Response interface {
    Render(w http.ResponseWriter, r *http.Request) error
}

// Context combines http.Request, ResponseWriter, and context.Context
type Context interface {
    context.Context
    Request() *http.Request
    ResponseWriter() http.ResponseWriter
    SSE() *datastar.ServerSentEventGenerator
}

// Request binding function
type Bind func(r *http.Request, v any) error

// Error handling function
type ErrorHandler[C Context] func(ctx C, err error)

// Handler decorator for middleware
type Decorator[C Context, R any] func(HandlerFunc[C, R]) HandlerFunc[C, R]

// Configuration option for Wrap
type WrapOption[C Context, R any] func(*wrapConfig[C, R])

// JSON response structure
type JSONResponse struct {
    Data  any            `json:"data,omitempty"`
    Meta  map[string]any `json:"meta,omitempty"`
    Error *ErrorDetail   `json:"error,omitempty"`
}

// Error details for JSON responses
type ErrorDetail struct {
    Code    string              `json:"code,omitempty"`
    Message string              `json:"message,omitempty"`
    Details map[string][]string `json:"details,omitempty"`
}

// HTTP error with status code and i18n key
type HTTPError struct {
    Code int    // HTTP status code
    Key  string // Translation key
}

// Field validation errors
type ValidationError url.Values

// Context key for type-safe context values
type ContextKey struct{ name string }

// Templ component interface (matches github.com/a-h/templ)
type TemplComponent interface {
    Render(ctx context.Context, w io.Writer) error
}

// Templ rendering options
type TemplOption = datastar.PatchElementOption

// Component with rendering options
type TemplPatch struct {
    Component TemplComponent
    Options   []datastar.PatchElementOption
}

// JSON response configuration
type JSONOption func(*jsonResponse)

// SSE handler function
type SSEHandler func(ctx StreamContext) error

// SSE streaming context
type StreamContext interface {
    Context
    SendComponent(component TemplComponent, opts ...TemplOption) error
    SendMultiple(patches ...TemplPatch) error
    SendSignal(name string, value any) error
    SendSignals(signals map[string]any) error
}
```

### Functions

```go
// Core handler wrapping
func Wrap[C Context, R any](h HandlerFunc[C, R], opts ...WrapOption[C, R]) http.HandlerFunc

// Wrap options
func WithBinder[C Context, R any](b Bind) WrapOption[C, R]
func WithBinders[C Context, R any](binders ...Bind) WrapOption[C, R]
func WithErrorHandler[C Context, R any](h ErrorHandler[C]) WrapOption[C, R]
func WithContextFactory[C Context, R any](f func(http.ResponseWriter, *http.Request) C) WrapOption[C, R]
func WithDecorators[C Context, R any](decorators ...Decorator[C, R]) WrapOption[C, R]

// Context creation and utilities
func NewContext(w http.ResponseWriter, r *http.Request) Context
func NewContextKey(name string) *ContextKey
func ContextValue[T any](ctx context.Context, key any) T
func ContextValueOK[T any](ctx context.Context, key any) (T, bool)

// JSON responses
func JSON(v any, opts ...JSONOption) Response
func JSONError(err any, opts ...JSONOption) Response
func WithJSONStatus(status int) JSONOption
func WithJSONMeta(meta map[string]any) JSONOption

// Redirect responses
func Redirect(url string) Response
func RedirectWithCode(url string, code int) Response
func RedirectBack(fallback string) Response
func RedirectBackWithCode(fallback string, code int) Response

// Template responses
func Templ(component TemplComponent, opts ...TemplOption) Response
func TemplPartial(partial, full TemplComponent, opts ...TemplOption) Response
func TemplMulti(patches ...TemplPatch) Response
func Patch(component TemplComponent, opts ...TemplOption) TemplPatch
func WithTarget(selector string) TemplOption
func WithPatchMode(mode datastar.ElementPatchMode) TemplOption

// DataStar/SSE utilities
func IsDataStar(r *http.Request) bool
func NewSSE(w http.ResponseWriter, r *http.Request) *datastar.ServerSentEventGenerator

// SSE streaming
func SSE(handler SSEHandler) Response

// Error creation
func NewHTTPError(code int, key string) HTTPError
func NewValidationError() ValidationError
```

### Methods

```go
// HTTPError methods
func (e HTTPError) Error() string

// ValidationError methods
func (e ValidationError) Error() string
func (e ValidationError) Add(field, message string)
func (e ValidationError) Get(field string) string
func (e ValidationError) Has(field string) bool
func (e ValidationError) IsEmpty() bool

// ContextKey methods
func (c *ContextKey) String() string

// Context implementation methods
func (c *httpContext) Request() *http.Request
func (c *httpContext) ResponseWriter() http.ResponseWriter
func (c *httpContext) SSE() *datastar.ServerSentEventGenerator
func (c *httpContext) Deadline() (deadline time.Time, ok bool)
func (c *httpContext) Done() <-chan struct{}
func (c *httpContext) Err() error
func (c *httpContext) Value(key any) any
```

### Error Types

```go
// Package errors
var ErrNilResponse = errors.New("handler returned nil response")

// 4xx Client errors
var ErrBadRequest = HTTPError{Code: 400, Key: "bad_request"}
var ErrUnauthorized = HTTPError{Code: 401, Key: "unauthorized"}
var ErrForbidden = HTTPError{Code: 403, Key: "forbidden"}
var ErrNotFound = HTTPError{Code: 404, Key: "not_found"}
var ErrUnprocessableEntity = HTTPError{Code: 422, Key: "unprocessable_entity"}
var ErrTooManyRequests = HTTPError{Code: 429, Key: "too_many_requests"}

// 5xx Server errors
var ErrInternalServerError = HTTPError{Code: 500, Key: "internal_server_error"}
var ErrServiceUnavailable = HTTPError{Code: 503, Key: "service_unavailable"}
```
