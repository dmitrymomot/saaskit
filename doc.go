// Package saaskit provides a minimal, type-safe framework for building SaaS applications in Go.
//
// SaasKit is designed for solo developers who want to ship MVPs quickly without sacrificing quality.
// It focuses on explicitness, type safety, and convention with escape hatches.
//
// Key Features:
//
//   - Type-safe HTTP handlers using generics
//   - Extensible request binding and error handling
//   - Context management with typed values
//   - Zero runtime dependencies
//   - Router-agnostic design
//
// Basic Usage:
//
//	// Define your request type
//	type CreateUserRequest struct {
//		Name  string `json:"name"`
//		Email string `json:"email"`
//	}
//
//	// Create a type-safe handler with standard context
//	handler := saaskit.HandlerFunc[saaskit.Context, CreateUserRequest](func(ctx saaskit.Context, req CreateUserRequest) saaskit.Response {
//		// req is already parsed and typed
//		user := createUser(req.Name, req.Email)
//		return JSONResponse(user)
//	})
//
//	// Use with any router
//	http.Handle("/users", saaskit.Wrap(handler))
//
// Advanced Usage with Options:
//
//	http.Handle("/users", saaskit.Wrap(handler,
//		saaskit.WithBinder(customBinder),
//		saaskit.WithErrorHandler(customErrorHandler),
//		saaskit.WithContextFactory(customContextFactory),
//	))
//
// Custom Context Support:
//
// SaasKit supports custom context types for direct access to application-specific data:
//
//	// Define your custom context interface
//	type AppContext interface {
//		saaskit.Context
//		UserID() string
//		TenantID() string
//	}
//
//	// Implement the interface
//	type appContext struct {
//		saaskit.Context
//		userID   string
//		tenantID string
//	}
//
//	func (c *appContext) UserID() string   { return c.userID }
//	func (c *appContext) TenantID() string { return c.tenantID }
//
//	// Create a factory function
//	func NewAppContext(w http.ResponseWriter, r *http.Request) AppContext {
//		return &appContext{
//			Context:  saaskit.NewContext(w, r),
//			userID:   extractUserID(r),
//			tenantID: extractTenantID(r),
//		}
//	}
//
//	// Use in handlers with direct access to custom methods
//	handler := saaskit.HandlerFunc[AppContext, CreateUserRequest](
//		func(ctx AppContext, req CreateUserRequest) saaskit.Response {
//			userID := ctx.UserID()     // Direct access, no type assertion!
//			tenantID := ctx.TenantID() // Type-safe access to custom methods
//			// ... handle request
//		},
//	)
//
//	// Wrap with custom context factory
//	http.Handle("/users", saaskit.Wrap(handler,
//		saaskit.WithContextFactory(NewAppContext),
//	))
//
// Context Management:
//
// SaasKit provides a Context interface that embeds context.Context and adds HTTP-specific methods:
//
//	// Store typed values in context
//	userKey := saaskit.NewContextKey("user")
//	ctx = context.WithValue(ctx, userKey, &User{ID: 123})
//
//	// Retrieve typed values safely
//	user := saaskit.ContextValue[*User](ctx, userKey)
//	if user != nil {
//		// Use user
//	}
//
// The framework follows these principles:
//   - API design over implementation details
//   - Developer experience over clever code
//   - Real usage over theoretical completeness
//   - Performance over features
//   - Explicit over implicit
package saaskit
