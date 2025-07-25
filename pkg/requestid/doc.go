// Package requestid provides HTTP middleware and helper utilities for working with
// request correlation identifiers (also known as request IDs).
//
// A request ID is a short opaque string that uniquely identifies an incoming HTTP
// request. Propagating the same ID through the request – via headers, context,
// and structured logs – makes it easy to correlate log records belonging to the
// same user interaction when troubleshooting distributed systems.
//
// # Overview
//
// The package offers:
//
//   - HTTP Middleware (see Middleware) that attaches a request ID to every
//     request. If the client supplies an "X-Request-ID" header its value is
//     validated and reused; otherwise a new UUIDv4 string is generated. The
//     chosen ID is stored in the request context and echoed back to the client
//     in the response header.
//
//   - Context helpers WithContext and FromContext for storing and extracting
//     request IDs from a context.Context.
//
//   - LoggerExtractor that integrates with the slog structured-logging package
//     so the request ID can be injected into log attributes effortlessly.
//
// # Usage
//
//	import (
//		"net/http"
//
//		"github.com/dmitrymomot/saaskit/pkg/requestid"
//	)
//
//	mux := http.NewServeMux()
//	mux.Handle("/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		id := requestid.FromContext(r.Context())
//		w.Write([]byte("hello, your request id is " + id))
//	}))
//
//	http.ListenAndServe(":8080", requestid.Middleware(mux))
//
// # Logger integration
//
//	import (
//		"log/slog"
//
//		"github.com/dmitrymomot/saaskit/pkg/requestid"
//	)
//
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	logger = logger.With(requestid.LoggerExtractor())
//
// # Constants
//
// The package exposes the Header constant holding the canonical request-ID
// header name ("X-Request-ID").
//
// # Error Handling
//
// The package does not return errors. Invalid or empty request IDs supplied by
// a client are silently replaced by a freshly generated UUID.
//
// See the package tests for more usage patterns.
//
//go:generate go test -run=Example
package requestid
