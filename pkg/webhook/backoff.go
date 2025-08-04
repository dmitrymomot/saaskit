package webhook

import (
	"math"
	"math/rand"
	"time"
)

// BackoffStrategy defines the interface for calculating retry delays.
// Implementations should be safe for concurrent use.
type BackoffStrategy interface {
	// NextInterval returns the next backoff duration based on the attempt number.
	// Attempt starts at 1 for the first retry.
	NextInterval(attempt int) time.Duration
}

// ExponentialBackoff implements exponential backoff with jitter.
// Jitter prevents thundering herd when multiple clients retry simultaneously.
type ExponentialBackoff struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	JitterFactor    float64
}

// NextInterval calculates exponential backoff with jitter to prevent coordinated retry storms.
// Formula: min(InitialInterval * (Multiplier ^ (attempt-1)) * (1 ± JitterFactor), MaxInterval)
func (e ExponentialBackoff) NextInterval(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Apply sensible defaults for webhook retry scenarios
	initial := e.InitialInterval
	if initial == 0 {
		initial = time.Second
	}

	max := e.MaxInterval
	if max == 0 {
		max = 30 * time.Second
	}

	multiplier := e.Multiplier
	if multiplier == 0 {
		multiplier = 2
	}

	// Zero jitter is intentionally allowed for deterministic behavior
	jitter := e.JitterFactor

	// Calculate exponential growth: initial * (multiplier ^ (attempt-1))
	interval := float64(initial) * math.Pow(multiplier, float64(attempt-1))

	// Apply jitter to spread retry times and prevent thundering herd
	if jitter > 0 {
		// Generate random factor between (1-jitter) and (1+jitter)
		randomJitter := (rand.Float64()*2 - 1) * jitter
		interval = interval * (1 + randomJitter)
	}

	// Respect maximum interval to prevent excessively long delays
	if interval > float64(max) {
		interval = float64(max)
	}

	return time.Duration(interval)
}

// LinearBackoff implements simple linear backoff without jitter.
// Provides predictable retry intervals that increase linearly.
type LinearBackoff struct {
	Interval    time.Duration
	MaxInterval time.Duration
}

// NextInterval returns linearly increasing delays suitable for predictable retry patterns.
// Formula: min(Interval * attempt, MaxInterval)
func (l LinearBackoff) NextInterval(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	interval := l.Interval
	if interval == 0 {
		interval = time.Second
	}

	max := l.MaxInterval
	if max == 0 {
		max = 30 * time.Second
	}

	delay := interval * time.Duration(attempt)
	if delay > max {
		delay = max
	}

	return delay
}

// FixedBackoff implements a constant delay between retries.
type FixedBackoff struct {
	// Interval is the fixed delay between retries
	Interval time.Duration
}

// NextInterval always returns the same interval regardless of attempt number.
func (f FixedBackoff) NextInterval(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	return f.Interval
}

// DefaultBackoffStrategy returns production-ready exponential backoff.
// Balances quick recovery with protection against overloading failing services.
func DefaultBackoffStrategy() BackoffStrategy {
	return ExponentialBackoff{
		InitialInterval: time.Second,      // Start with short delay for transient issues
		MaxInterval:     30 * time.Second, // Cap at 30s to avoid excessive delays
		Multiplier:      2,                // Double delay each attempt
		JitterFactor:    0.1,              // Add 10% jitter to prevent thundering herd
	}
}
