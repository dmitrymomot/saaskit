// Package handler provides type-safe HTTP request handling for building SaaS applications in Go.
//
// The package offers a modern approach to HTTP handling with compile-time type safety,
// multiple response formats, and first-class support for real-time UI updates via DataStar.
// It's designed to reduce boilerplate while maintaining explicitness and flexibility.
//
// # Core Concepts
//
// The handler package centers around generic handler functions that bind HTTP requests
// to Go structs and return typed responses. This eliminates manual request parsing and
// response encoding while providing compile-time guarantees:
//
//	type CreateUserRequest struct {
//		Email    string `json:"email" validate:"required,email"`
//		Password string `json:"password" validate:"required,min=8"`
//	}
//
//	func createUser(ctx handler.Context, req CreateUserRequest) handler.Response {
//		user, err := userService.Create(req)
//		if err != nil {
//			return handler.JSONError(err)
//		}
//		return handler.JSON(user)
//	}
//
//	http.HandleFunc("/users", handler.Wrap(createUser))
//
// # Architecture
//
// The package uses a layered architecture:
//
// 1. HandlerFunc - Generic function type that accepts typed requests and returns responses
// 2. Response Interface - Common interface for all response types (JSON, HTML, redirects)
// 3. Context Interface - Enhanced context providing access to request, response, and SSE
// 4. Decorators - Middleware-like functions for cross-cutting concerns
// 5. Error Handlers - Customizable error response formatting
//
// # Response Types
//
// The package supports multiple response formats, automatically selected based on
// the request type:
//
// JSON responses for APIs:
//
//	handler.JSON(data)                    // 200 OK with data
//	handler.JSON(data, WithJSONStatus(201)) // Custom status
//	handler.JSONError(err)                // Error response
//
// Template responses for server-rendered HTML (using templ):
//
//	handler.Templ(component)              // Render single component
//	handler.TemplPartial(partial, full)   // Conditional rendering
//	handler.TemplMulti(patches...)        // Multiple components
//
// Redirect responses:
//
//	handler.Redirect("/success")          // 303 See Other
//	handler.RedirectBack("/fallback")     // Redirect to referrer
//
// SSE streaming responses:
//
//	handler.SSE(func(stream StreamContext) error {
//		// Long-lived connection for real-time updates
//		return stream.SendComponent(component, opts...)
//	})
//
// # DataStar Integration
//
// DataStar requests (identified by Accept: text/event-stream) automatically receive
// Server-Sent Events responses, enabling real-time UI updates without JavaScript:
//
//	if handler.IsDataStar(ctx.Request()) {
//		return handler.Templ(component,
//			handler.WithTarget("#list"),
//			handler.WithPatchMode(handler.PatchAppend))
//	}
//	return handler.JSON(data)
//
// # Error Handling
//
// The package provides structured error handling with i18n support:
//
//	// HTTP errors with translation keys
//	handler.ErrNotFound         // 404 with key "http.error.not_found"
//	handler.ErrUnauthorized     // 401 with key "http.error.unauthorized"
//
//	// Validation errors with field details
//	err := handler.NewValidationError()
//	err.Add("email", "Email is required")
//	err.Add("email", "Email format is invalid")
//	return handler.JSONError(err)  // 422 with field errors
//
// # Context Enhancement
//
// The Context interface extends standard context.Context with HTTP-specific methods:
//
//	ctx.Request()         // Access HTTP request
//	ctx.ResponseWriter()  // Access response writer
//	ctx.SSE()            // Get SSE generator for DataStar
//
// # Usage
//
// Basic handler registration with a router:
//
//	import "github.com/dmitrymomot/saaskit/handler"
//
//	// Define handler function
//	func createUser(ctx handler.Context, req CreateUserRequest) handler.Response {
//		// Implementation
//		return handler.JSON(result)
//	}
//
//	// Register with router
//	http.HandleFunc("/users", handler.Wrap(createUser))
//
// With custom options:
//
//	http.HandleFunc("/users", handler.Wrap(createUser,
//		handler.WithBinders(
//			binder.JSON(),          // Parse JSON body
//			binder.Validate(),      // Validate struct tags
//		),
//		handler.WithDecorators(
//			decorators.Logger(),    // Log requests
//			decorators.RequireAuth(), // Check authentication
//		),
//		handler.WithErrorHandler(customErrorHandler),
//	))
//
// # Performance Considerations
//
// The package is designed for minimal allocations:
//
// - Response interfaces avoid unnecessary allocations
// - Context reuses standard library types
// - No reflection in the hot path (only during initialization)
// - Efficient error handling without panics
//
// For maximum performance, pre-compile response templates and reuse handler instances.
package handler
