# Client IP Detection Package

A simple, reliable Go package for determining client IP addresses from HTTP requests with optimized support for DigitalOcean App Platform deployments.

## Features

- **DigitalOcean App Platform Optimized**: First-class support for `DO-Connecting-IP` header
- **Cloudflare Integration**: Handles `CF-Connecting-IP` for Cloudflare → DigitalOcean Apps traffic
- **Standard Proxy Support**: Works with `X-Forwarded-For`, `X-Real-IP` headers
- **IPv4/IPv6 Compatible**: Full support for both address types
- **No External Dependencies**: Uses only Go standard library
- **High Performance**: < 1ms average execution time (146-650 ns/op depending on scenario)
- **Memory Efficient**: Uses Go 1.24+ `strings.SplitSeq` for zero-allocation string iteration
- **Modern Go**: Leverages latest Go language features for optimal performance
- **Robust Error Handling**: Graceful fallback through header priority chain

## Usage

```go
import "your-app/internal/pkg/clientip"

func handleRequest(w http.ResponseWriter, r *http.Request) {
    ip := clientip.GetIP(r)
    fmt.Printf("Client IP: %s\n", ip)
}
```

## Header Priority Chain

The package checks headers in the following priority order, optimized for DigitalOcean App Platform:

1. **`CF-Connecting-IP`** - Cloudflare edge → DigitalOcean Apps
2. **`DO-Connecting-IP`** - DigitalOcean App Platform (primary for DO deployments)
3. **`X-Forwarded-For`** - Standard load balancer/proxy header
4. **`X-Real-IP`** - Nginx reverse proxy header
5. **`RemoteAddr`** - Direct connection fallback

## DigitalOcean App Platform Integration

### Direct DigitalOcean Apps Deployment

```go
// Request headers from DO Apps:
// DO-Connecting-IP: 203.0.113.195 (real client IP)
// X-Forwarded-For: 10.244.0.1 (internal DO network)

ip := clientip.GetIP(request) // Returns: "203.0.113.195"
```

### Cloudflare + DigitalOcean Apps

```go
// Request headers with Cloudflare:
// CF-Connecting-IP: 198.51.100.178 (real client IP from CF)
// DO-Connecting-IP: 203.0.113.195 (CF edge server IP)
// X-Forwarded-For: 10.244.0.1 (internal DO network)

ip := clientip.GetIP(request) // Returns: "198.51.100.178" (CF takes priority)
```

### DigitalOcean Load Balancer

```go
// Standard load balancer deployment:
// X-Forwarded-For: 203.0.113.195, 10.244.0.1
// X-Real-IP: 203.0.113.195

ip := clientip.GetIP(request) // Returns: "203.0.113.195"
```

## Function Reference

### `GetIP(r *http.Request) string`

Returns the client's IP address from the HTTP request.

**Parameters:**

- `r *http.Request` - The HTTP request object

**Returns:**

- `string` - The client's IP address (IPv4 or IPv6 format)

**Behavior:**

- Always returns a valid IP string (never empty)
- Validates IP format using Go's `net.ParseIP`
- Strips port numbers from `RemoteAddr`
- Handles comma-separated IP lists in `X-Forwarded-For`
- Normalizes IPv6 addresses to standard format
- Falls back to `RemoteAddr` if no proxy headers are valid

## Examples

### Basic Usage

```go
package main

import (
    "fmt"
    "net/http"
    "your-app/internal/pkg/clientip"
)

func handler(w http.ResponseWriter, r *http.Request) {
    ip := clientip.GetIP(r)

    // Log the client IP
    fmt.Printf("Request from IP: %s\n", ip)

    // Use in security context
    if isBlockedIP(ip) {
        http.Error(w, "Access denied", http.StatusForbidden)
        return
    }

    // Use in analytics
    trackUserRequest(ip, r.URL.Path)

    w.WriteHeader(http.StatusOK)
}

func main() {
    http.HandleFunc("/", handler)
    http.ListenAndServe(":8080", nil)
}
```

### Rate Limiting by IP

```go
import (
    "sync"
    "time"
    "your-app/internal/pkg/clientip"
)

type RateLimiter struct {
    requests map[string][]time.Time
    mutex    sync.RWMutex
}

func (rl *RateLimiter) IsAllowed(r *http.Request, maxRequests int, window time.Duration) bool {
    ip := clientip.GetIP(r)

    rl.mutex.Lock()
    defer rl.mutex.Unlock()

    now := time.Now()
    cutoff := now.Add(-window)

    // Clean old requests
    var validRequests []time.Time
    for _, reqTime := range rl.requests[ip] {
        if reqTime.After(cutoff) {
            validRequests = append(validRequests, reqTime)
        }
    }

    // Check limit
    if len(validRequests) >= maxRequests {
        return false
    }

    // Add current request
    validRequests = append(validRequests, now)
    rl.requests[ip] = validRequests

    return true
}
```

### Middleware Integration

The package provides built-in middleware for extracting and storing client IP in context:

```go
import "github.com/dmitrymomot/saaskit/pkg/clientip"

// Use the built-in middleware
router.Use(clientip.Middleware)

// In your handlers, retrieve IP from context
func MyHandler(w http.ResponseWriter, r *http.Request) {
    ip := clientip.GetIPFromContext(r.Context())
    fmt.Printf("Client IP: %s\n", ip)
}
```

### Context Helpers

```go
// Store IP in context (done automatically by middleware)
ctx := clientip.SetIPToContext(ctx, "192.168.1.1")

// Retrieve IP from context
ip := clientip.GetIPFromContext(ctx) // Returns: "192.168.1.1"
```

## Error Handling

The package is designed to never panic and always return a usable IP address:

- **Invalid IP formats**: Skipped, continues to next header
- **Empty headers**: Ignored, continues to next priority
- **Malformed X-Forwarded-For**: Parses valid IPs, skips invalid ones
- **No proxy headers**: Falls back to `RemoteAddr`
- **Invalid RemoteAddr**: Returns the raw `RemoteAddr` string

## Performance

The package is optimized for high-performance scenarios:

- **Benchmark**: 146-650 ns/op depending on scenario (well under 1ms requirement)
- **Memory**: Zero-allocation string iteration using Go 1.24+ `strings.SplitSeq`
- **CPU**: Early exit on first valid IP found in forwarded chains
- **Modern Go**: Uses latest language features for optimal performance
- **Scalability**: Suitable for high-traffic applications

## Deployment Scenarios

### DigitalOcean App Platform

```yaml
# app.yaml
name: my-app
services:
    - name: web
      source_dir: .
      github:
          repo: your-org/your-repo
          branch: main
      run_command: ./your-app
      environment_slug: go
```

The `DO-Connecting-IP` header will automatically be set by the platform.

### Docker + DigitalOcean Load Balancer

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o app ./cmd/app

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/app .
CMD ["./app"]
```

Configure your load balancer to preserve client IPs via `X-Forwarded-For`.

### Local Development

```go
// For local testing, the package will fall back to RemoteAddr
// which will typically be 127.0.0.1 or ::1

ip := clientip.GetIP(request) // Returns: "127.0.0.1" for local requests
```

## Testing

The package includes comprehensive test coverage:

```bash
# Run tests
go test -race ./internal/pkg/clientip

# Run benchmarks
go test -bench=. ./internal/pkg/clientip

# Check coverage
go test -cover -race ./internal/pkg/clientip
```

## Security Considerations

- **Header Spoofing**: Only trust proxy headers from your own infrastructure
- **IPv6 Support**: The package correctly handles both IPv4 and IPv6 addresses
- **Logging**: Consider logging IP addresses for security auditing
- **Rate Limiting**: Use the detected IP for rate limiting and abuse prevention

## Migration from Other Solutions

### From `r.RemoteAddr`

```go
// Before
ip, _, _ := net.SplitHostPort(r.RemoteAddr)

// After
ip := clientip.GetIP(r)
```

### From Manual Header Checking

```go
// Before
ip := r.Header.Get("X-Forwarded-For")
if ip == "" {
    ip = r.Header.Get("X-Real-IP")
}
if ip == "" {
    ip, _, _ = net.SplitHostPort(r.RemoteAddr)
}

// After
ip := clientip.GetIP(r)
```

## Troubleshooting

### No Client IP Detected

1. Check if your load balancer/proxy is setting the expected headers
2. Verify the header names match your infrastructure
3. Test with `curl -H "DO-Connecting-IP: 1.2.3.4" your-app.com`

### Wrong IP Address

1. Ensure you're using the correct deployment configuration
2. Check if multiple proxies are adding headers
3. Verify header priority matches your infrastructure setup

### Performance Issues

1. Profile your application to identify bottlenecks
2. Check if IP parsing is being called too frequently
3. Consider caching results if the same request is processed multiple times
