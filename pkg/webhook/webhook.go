package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Sender provides reliable webhook delivery with retries and circuit breaking.
// Zero value is not usable; use NewSender to create instances.
type Sender struct {
	// client is reused across requests for connection pooling and performance
	client *http.Client
}

// NewSender creates a webhook sender with default HTTP client.
// Connection pooling is configured for high-throughput scenarios while preventing
// connection leaks. Timeout values balance responsiveness with allowing slow endpoints.
func NewSender() *Sender {
	return &Sender{
		client: &http.Client{
			Timeout: 30 * time.Second, // Per-request timeout, overridden by WithTimeout
			Transport: &http.Transport{
				MaxIdleConns:        100,              // Total connections across all hosts
				MaxIdleConnsPerHost: 10,               // Connections per webhook endpoint
				IdleConnTimeout:     90 * time.Second, // Close idle connections after 90s
			},
		},
	}
}

// NewSenderWithClient creates a webhook sender with a custom HTTP client.
// This allows for custom transports, proxies, or testing.
func NewSenderWithClient(client *http.Client) *Sender {
	if client == nil {
		return NewSender()
	}
	return &Sender{client: client}
}

// Send delivers a webhook payload to the specified URL with retry logic.
// The payload is automatically marshaled to JSON and sent as a POST request with Content-Type: application/json.
// The data parameter can be any Go value that can be marshaled to JSON (struct, map, slice, etc.).
// Options control timeout, retries, signing, and other behavior.
//
// Example:
//
//	type Event struct {
//		Type string `json:"type"`
//		ID   string `json:"id"`
//		Data map[string]any `json:"data"`
//	}
//
//	event := Event{
//		Type: "user.created",
//		ID:   "evt_123",
//		Data: map[string]any{"user_id": "usr_456"},
//	}
//
//	err := sender.Send(ctx, webhookURL, event, webhook.WithSignature(secret))
func (s *Sender) Send(ctx context.Context, webhookURL string, data any, opts ...SendOption) error {
	// Marshal the data to JSON
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal payload to JSON: %w", err)
	}

	if err := s.validateInputs(webhookURL, payload); err != nil {
		return err
	}

	options := defaultSendOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Allow per-request client override for testing or custom transports
	client := s.client
	if options.httpClient != nil {
		client = options.httpClient
	}

	// Fail fast if circuit breaker is protecting the endpoint
	if options.circuitBreaker != nil && !options.circuitBreaker.Allow() {
		return ErrCircuitOpen
	}

	// Retry loop with exponential backoff
	var lastErr error
	for attempt := 0; attempt <= options.maxRetries; attempt++ {
		// Apply backoff delay, respecting context cancellation
		if attempt > 0 {
			delay := options.backoffStrategy.NextInterval(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		result, err := s.attemptDelivery(ctx, client, webhookURL, payload, options)

		// Notify observers of delivery attempt for metrics/logging
		if options.onDelivery != nil {
			result.Attempt = attempt + 1
			options.onDelivery(result)
		}

		// Update circuit breaker state based on result
		if options.circuitBreaker != nil {
			if err == nil {
				options.circuitBreaker.RecordSuccess()
			} else {
				options.circuitBreaker.RecordFailure()
			}
		}

		if err == nil {
			return nil
		}

		lastErr = err

		// Exit early for permanent failures (4xx codes) to avoid wasting resources
		if isPermanentError(result.StatusCode, err) {
			return fmt.Errorf("%w: %w", ErrPermanentFailure, err)
		}
	}

	return fmt.Errorf("%w after %d attempts: %w", ErrWebhookDeliveryFailed, options.maxRetries+1, lastErr)
}

// validateInputs performs early validation to fail fast on obvious errors
func (s *Sender) validateInputs(webhookURL string, payload []byte) error {
	if webhookURL == "" {
		return fmt.Errorf("%w: URL is required", ErrInvalidURL)
	}

	// Parse and validate URL
	u, err := url.Parse(webhookURL)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}

	// Restrict to HTTP/HTTPS for security and to prevent SSRF attacks
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: only http and https schemes are supported", ErrInvalidURL)
	}

	if u.Host == "" {
		return fmt.Errorf("%w: host is required", ErrInvalidURL)
	}

	// Check payload
	if len(payload) == 0 {
		return fmt.Errorf("%w: payload cannot be empty", ErrInvalidPayload)
	}

	return nil
}

// attemptDelivery makes a single HTTP request attempt with timing and error capture
func (s *Sender) attemptDelivery(ctx context.Context, client *http.Client, webhookURL string, payload []byte, options *sendOptions) (DeliveryResult, error) {
	start := time.Now()
	result := DeliveryResult{}

	// Layer timeout on top of parent context to respect both constraints
	reqCtx, cancel := context.WithTimeout(ctx, options.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		result.Duration = time.Since(start)
		result.Error = err
		return result, fmt.Errorf("failed to create request: %w", err)
	}

	// Set standard headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "saaskit-webhook/1.0")

	// Add custom headers
	for k, v := range options.headers {
		req.Header.Set(k, v)
	}

	// Add signature if secret is provided
	if options.signatureSecret != "" {
		sigHeaders, err := SignPayload(options.signatureSecret, payload)
		if err != nil {
			result.Duration = time.Since(start)
			result.Error = err
			return result, fmt.Errorf("failed to sign payload: %w", err)
		}
		for k, v := range sigHeaders.Headers() {
			req.Header.Set(k, v)
		}
	}

	// Execute request
	resp, err := client.Do(req)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err
		// Check for timeout
		if reqCtx.Err() == context.DeadlineExceeded {
			return result, fmt.Errorf("%w: %w", ErrTimeout, err)
		}
		return result, fmt.Errorf("%w: %w", ErrTemporaryFailure, err)
	}

	defer func() { _ = resp.Body.Close() }()
	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	// Read response body for error context (64KB limit prevents memory exhaustion)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*64))

	// Check status code
	if !result.Success {
		errMsg := fmt.Sprintf("webhook returned status %d", resp.StatusCode)
		if len(body) > 0 {
			// Sanitize response body for safe logging and prevent log injection
			bodyStr := strings.ReplaceAll(string(body), "\n", " ")
			if len(bodyStr) > 200 {
				bodyStr = bodyStr[:200] + "..."
			}
			errMsg += fmt.Sprintf(": %s", bodyStr)
		}
		result.Error = fmt.Errorf("%s", errMsg)
		return result, result.Error
	}

	return result, nil
}

// isPermanentError determines if an error should not be retried based on HTTP semantics.
// Most 4xx errors indicate client-side issues that won't resolve with retries,
// but some 4xx codes represent temporary server-side rate limiting or timing issues.
func isPermanentError(statusCode int, err error) bool {
	if statusCode >= 400 && statusCode < 500 {
		// Exception list: 4xx codes that may resolve with retry
		switch statusCode {
		case 408: // Request Timeout - server couldn't process in time
			return false
		case 425: // Too Early - server not ready yet
			return false
		case 429: // Too Many Requests - rate limiting
			return false
		default:
			// 400, 401, 403, 404, etc. - client errors that won't change
			return true
		}
	}
	// Network errors, 5xx codes, and other issues are considered temporary
	return false
}
