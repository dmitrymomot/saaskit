package ratelimiter

import "time"

// Result contains the result of a rate limit check.
type Result struct {
	Limit     int       // Maximum tokens (bucket capacity)
	Remaining int       // Tokens remaining (negative means denied)
	ResetAt   time.Time // When next token refill occurs
}

func (r *Result) Allowed() bool {
	return r.Remaining >= 0
}

// RetryAfter returns how long to wait before retry is likely to succeed.
// Returns 0 if the request was allowed.
func (r *Result) RetryAfter() time.Duration {
	if r.Allowed() {
		return 0
	}
	return time.Until(r.ResetAt)
}

// Config defines the token bucket rate limiting parameters.
type Config struct {
	Capacity       int           // Maximum tokens (burst capacity)
	RefillRate     int           // Tokens added per interval
	RefillInterval time.Duration // How frequently tokens are added
}
