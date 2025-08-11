package ratelimiter

import "time"

// Result contains the result of a rate limit check.
type Result struct {
	Limit     int       // Maximum tokens (bucket capacity)
	Remaining int       // Tokens remaining
	ResetAt   time.Time // Time when tokens will be refilled
}

// Allowed returns whether the request is allowed based on remaining tokens.
func (r *Result) Allowed() bool {
	return r.Remaining >= 0
}

// RetryAfter returns how long to wait before the next request.
// Returns 0 if the request was allowed.
func (r *Result) RetryAfter() time.Duration {
	if r.Allowed() {
		return 0
	}
	return time.Until(r.ResetAt)
}

// Config defines the token bucket configuration.
type Config struct {
	Capacity       int           // Maximum tokens the bucket can hold (burst limit)
	RefillRate     int           // Number of tokens added per refill interval
	RefillInterval time.Duration // How often tokens are added
}
