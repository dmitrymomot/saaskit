// Package clientip provides utilities for extracting the originating
// client's IP address from an *http.Request when your application is
// deployed behind one or more reverse proxies.
//
// The implementation is optimised for workloads running on the
// DigitalOcean App Platform behind Cloudflare, but it works in any
// environment that forwards the original client address using standard
// proxy headers.
//
// The resolution algorithm examines several headers in descending
// priority until the first valid IP address is found:
//
//   1. CF-Connecting-IP  – Cloudflare → DigitalOcean Apps
//   2. DO-Connecting-IP  – DigitalOcean App Platform primary header
//   3. X-Forwarded-For   – comma-separated list (the first IP is used)
//   4. X-Real-IP         – set by reverse proxies such as Nginx
//   5. RemoteAddr        – TCP peer address as a fallback
//
// Helper functions are provided for common scenarios:
//
//   • GetIP extracts the client IP from an *http.Request.
//   • SetIPToContext and GetIPFromContext store/retrieve the resolved
//     address inside a context.Context.
//   • Middleware is a net/http compatible middleware that adds the IP to
//     the request's context so downstream handlers can fetch it without
//     duplicating the resolution logic.
//
// # Usage
//
// import "github.com/dmitrymomot/saaskit/pkg/clientip"
//
// // Inside a handler
// func handler(w http.ResponseWriter, r *http.Request) {
//     ip := clientip.GetIP(r)
//     log.Printf("client ip: %s", ip)
// }
//
// // As middleware
// mux := http.NewServeMux()
// mux.HandleFunc("/", handler)
// http.ListenAndServe(":8080", clientip.Middleware(mux))
//
// # Error Handling
//
// GetIP never returns an error. If no valid address is found an empty
// string is returned so callers can decide how to proceed.
//
// # See Also
//
// The standard library packages net/http and net.
package clientip
