// Package ratelimiter provides token bucket rate limiting with memory storage and HTTP middleware.
//
// The package implements a token bucket algorithm that allows burst traffic up to a configured
// capacity while maintaining a steady refill rate. It includes an in-memory storage backend
// with automatic cleanup and HTTP middleware for easy integration into web applications.
//
// # Basic Usage
//
// Create a rate limiter with a memory store:
//
//	config := ratelimiter.Config{
//		Capacity:       100,         // Maximum tokens (burst capacity)
//		RefillRate:     10,          // Tokens added per interval
//		RefillInterval: time.Second, // Refill frequency
//	}
//
//	store := ratelimiter.NewMemoryStore()
//	defer store.Close()
//
//	limiter, err := ratelimiter.NewBucket(store, config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Check if a request is allowed
//	result, err := limiter.Allow(ctx, "user:123")
//	if err != nil {
//		// Handle error
//		return
//	}
//
//	if !result.Allowed() {
//		// Rate limit exceeded, retry after result.RetryAfter()
//		return
//	}
//
// # HTTP Middleware
//
// Use the provided middleware for HTTP rate limiting:
//
//	// Simple IP-based rate limiting
//	keyFunc := func(r *http.Request) string {
//		return r.RemoteAddr
//	}
//
//	middleware := ratelimiter.Middleware(limiter, keyFunc)
//
//	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		w.Write([]byte("Hello, World!"))
//	}))
//
//	http.ListenAndServe(":8080", handler)
//
// The middleware automatically sets standard rate limit headers:
//   - X-RateLimit-Limit: Maximum tokens
//   - X-RateLimit-Remaining: Tokens remaining
//   - X-RateLimit-Reset: Unix timestamp of next refill
//
// # Composite Key Functions
//
// Combine multiple key extractors for complex rate limiting scenarios:
//
//	keyFunc := ratelimiter.Composite(
//		func(r *http.Request) string { return r.Header.Get("X-API-Key") },
//		func(r *http.Request) string { return r.RemoteAddr },
//	)
//
// Keys longer than 64 characters are automatically hashed using FNV-1a
// to prevent unbounded storage growth.
//
// # Advanced Operations
//
// Consume multiple tokens at once:
//
//	result, err := limiter.AllowN(ctx, "user:123", 5)
//	if err != nil {
//		return err
//	}
//
// Check bucket status without consuming tokens:
//
//	result, err := limiter.Status(ctx, "user:123")
//	if err != nil {
//		return err
//	}
//
//	fmt.Printf("Remaining: %d, Reset at: %v\n",
//		result.Remaining, result.ResetAt)
//
// Reset a bucket (useful for administrative operations):
//
//	err := limiter.Reset(ctx, "user:123")
//	if err != nil {
//		return err
//	}
//
// # Memory Management
//
// The MemoryStore automatically cleans up stale buckets to prevent memory leaks:
//
//	store := ratelimiter.NewMemoryStore(
//		ratelimiter.WithCleanupInterval(10 * time.Minute),
//	)
//
// Buckets are considered stale if they haven't been accessed for 1 hour.
// Disable cleanup by setting the interval to 0.
//
// # Custom Error Handling
//
// Customize error responses in the HTTP middleware:
//
//	errorResponder := func(w http.ResponseWriter, r *http.Request, result *ratelimiter.Result, err error) {
//		if err != nil {
//			http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
//			return
//		}
//
//		if result != nil && !result.Allowed() {
//			retryAfter := int(result.RetryAfter().Seconds())
//			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
//			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
//		}
//	}
//
//	middleware := ratelimiter.Middleware(limiter, keyFunc,
//		ratelimiter.WithErrorResponder(errorResponder),
//	)
//
// # Error Types
//
// The package defines several error types for different failure scenarios:
//
//	if errors.Is(err, ratelimiter.ErrInvalidConfig) {
//		// Configuration validation failed
//	}
//	if errors.Is(err, ratelimiter.ErrInvalidTokenCount) {
//		// Token count must be positive
//	}
//	if errors.Is(err, ratelimiter.ErrStoreUnavailable) {
//		// Storage backend is unavailable
//	}
//	if errors.Is(err, ratelimiter.ErrContextCancelled) {
//		// Operation cancelled due to context
//	}
//
// # Thread Safety
//
// All operations are thread-safe and can be used concurrently across multiple goroutines.
// The MemoryStore uses read-write mutexes for optimal performance with concurrent access.
//
// # Token Bucket Algorithm
//
// The implementation uses the standard token bucket algorithm:
//  1. Tokens are added to the bucket at the configured RefillRate and RefillInterval
//  2. Each request consumes one or more tokens
//  3. If insufficient tokens are available, the request is denied
//  4. The bucket capacity limits the maximum burst size
//
// This provides smooth rate limiting with burst tolerance, making it suitable for
// web APIs, user rate limiting, and resource protection scenarios.
package ratelimiter
