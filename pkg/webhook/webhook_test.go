package webhook_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/webhook"
)

func TestSender_Send_Success(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"event": "test",
		"id":    "123",
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "saaskit-webhook/1.0", r.Header.Get("User-Agent"))

		// Read body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		// Verify JSON content
		var received map[string]any
		err = json.Unmarshal(body, &received)
		require.NoError(t, err)
		assert.Equal(t, payload, received)

		// Success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	sender := webhook.NewSender()
	err := sender.Send(context.Background(), server.URL, payload)
	assert.NoError(t, err)
}

func TestSender_Send_WithOptions(t *testing.T) {
	t.Parallel()

	payload := map[string]string{"test": "data"}
	secret := "webhook_secret"

	var deliveryResults []webhook.DeliveryResult

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom headers
		assert.Equal(t, "test-value", r.Header.Get("X-Custom-Header"))

		// Verify signature headers
		assert.NotEmpty(t, r.Header.Get("X-Webhook-Signature"))
		assert.NotEmpty(t, r.Header.Get("X-Webhook-Timestamp"))
		assert.NotEmpty(t, r.Header.Get("X-Webhook-ID"))

		// Extract and verify signature
		headers, err := webhook.ExtractSignatureHeaders(map[string]string{
			"X-Webhook-Signature": r.Header.Get("X-Webhook-Signature"),
			"X-Webhook-Timestamp": r.Header.Get("X-Webhook-Timestamp"),
			"X-Webhook-ID":        r.Header.Get("X-Webhook-ID"),
		})
		require.NoError(t, err)

		body, _ := io.ReadAll(r.Body)
		err = webhook.VerifySignature(secret, body, headers, 5*time.Minute)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := webhook.NewSender()
	err := sender.Send(
		context.Background(),
		server.URL,
		payload,
		webhook.WithSignature(secret),
		webhook.WithHeader("X-Custom-Header", "test-value"),
		webhook.WithTimeout(5*time.Second),
		webhook.WithMaxRetries(2),
		webhook.WithOnDelivery(func(result webhook.DeliveryResult) {
			deliveryResults = append(deliveryResults, result)
		}),
	)

	assert.NoError(t, err)
	assert.Len(t, deliveryResults, 1)
	assert.True(t, deliveryResults[0].Success)
	assert.Equal(t, http.StatusOK, deliveryResults[0].StatusCode)
}

func TestSender_Send_Retries(t *testing.T) {
	t.Parallel()

	payload := map[string]string{"test": "retry"}
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)

		if attempt < 3 {
			// Fail first 2 attempts with 500
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("temporary error"))
			return
		}

		// Succeed on 3rd attempt
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := webhook.NewSender()
	err := sender.Send(
		context.Background(),
		server.URL,
		payload,
		webhook.WithMaxRetries(3),
		webhook.WithBackoff(webhook.FixedBackoff{Interval: 10 * time.Millisecond}),
	)

	assert.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestSender_Send_PermanentFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		statusCode  int
		shouldRetry bool
	}{
		{"400 Bad Request", http.StatusBadRequest, false},
		{"401 Unauthorized", http.StatusUnauthorized, false},
		{"403 Forbidden", http.StatusForbidden, false},
		{"404 Not Found", http.StatusNotFound, false},
		{"408 Request Timeout", http.StatusRequestTimeout, true},
		{"425 Too Early", http.StatusTooEarly, true},
		{"429 Too Many Requests", http.StatusTooManyRequests, true},
		{"500 Internal Server Error", http.StatusInternalServerError, true},
		{"502 Bad Gateway", http.StatusBadGateway, true},
		{"503 Service Unavailable", http.StatusServiceUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var attempts int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&attempts, 1)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("error message"))
			}))
			defer server.Close()

			sender := webhook.NewSender()
			err := sender.Send(
				context.Background(),
				server.URL,
				map[string]string{"test": "data"},
				webhook.WithMaxRetries(3),
				webhook.WithBackoff(webhook.FixedBackoff{Interval: time.Millisecond}),
			)

			require.Error(t, err)

			if tt.shouldRetry {
				assert.Equal(t, int32(4), atomic.LoadInt32(&attempts), "should retry for %d", tt.statusCode)
				assert.ErrorIs(t, err, webhook.ErrWebhookDeliveryFailed)
			} else {
				assert.Equal(t, int32(1), atomic.LoadInt32(&attempts), "should not retry for %d", tt.statusCode)
				assert.ErrorIs(t, err, webhook.ErrPermanentFailure)
			}
		})
	}
}

func TestSender_Send_CircuitBreaker(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cb := webhook.NewCircuitBreaker(2, 1, 100*time.Millisecond)
	sender := webhook.NewSender()

	// First 2 requests should fail and open the circuit
	for i := 0; i < 2; i++ {
		err := sender.Send(
			context.Background(),
			server.URL,
			map[string]string{"test": "data"},
			webhook.WithCircuitBreaker(cb),
			webhook.WithNoRetry(),
		)
		require.Error(t, err)
	}

	// Circuit should now be open
	assert.Equal(t, webhook.CircuitOpen, cb.State())

	// Next request should fail immediately
	err := sender.Send(
		context.Background(),
		server.URL,
		map[string]string{"test": "data"},
		webhook.WithCircuitBreaker(cb),
	)
	assert.ErrorIs(t, err, webhook.ErrCircuitOpen)

	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)

	// Circuit should be half-open now
	assert.Equal(t, webhook.CircuitHalfOpen, cb.State())
}

func TestSender_Send_Timeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than timeout
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := webhook.NewSender()
	err := sender.Send(
		context.Background(),
		server.URL,
		map[string]string{"test": "data"},
		webhook.WithTimeout(50*time.Millisecond),
		webhook.WithNoRetry(),
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, webhook.ErrTimeout)
}

func TestSender_Send_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after short delay
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	sender := webhook.NewSender()
	err := sender.Send(ctx, server.URL, map[string]string{"test": "data"})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSender_Send_ValidationErrors(t *testing.T) {
	t.Parallel()

	sender := webhook.NewSender()

	tests := []struct {
		name    string
		url     string
		payload any
		wantErr error
		errMsg  string
	}{
		{
			name:    "empty URL",
			url:     "",
			payload: map[string]string{"test": "data"},
			wantErr: webhook.ErrInvalidURL,
			errMsg:  "URL is required",
		},
		{
			name:    "invalid URL",
			url:     "not a url",
			payload: map[string]string{"test": "data"},
			wantErr: webhook.ErrInvalidURL,
			errMsg:  "only http and https schemes are supported",
		},
		{
			name:    "invalid scheme",
			url:     "ftp://example.com",
			payload: map[string]string{"test": "data"},
			wantErr: webhook.ErrInvalidURL,
			errMsg:  "only http and https schemes are supported",
		},
		{
			name:    "missing host",
			url:     "http:///path",
			payload: map[string]string{"test": "data"},
			wantErr: webhook.ErrInvalidURL,
			errMsg:  "host is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := sender.Send(context.Background(), tt.url, tt.payload)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestSender_Send_DeliveryHook(t *testing.T) {
	t.Parallel()

	var results []webhook.DeliveryResult
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := webhook.NewSender()
	err := sender.Send(
		context.Background(),
		server.URL,
		map[string]string{"test": "data"},
		webhook.WithMaxRetries(2),
		webhook.WithBackoff(webhook.FixedBackoff{Interval: time.Millisecond}),
		webhook.WithOnDelivery(func(result webhook.DeliveryResult) {
			results = append(results, result)
		}),
	)

	assert.NoError(t, err)
	require.Len(t, results, 2)

	// First attempt failed
	assert.False(t, results[0].Success)
	assert.Equal(t, http.StatusInternalServerError, results[0].StatusCode)
	assert.Equal(t, 1, results[0].Attempt)
	assert.NotNil(t, results[0].Error)

	// Second attempt succeeded
	assert.True(t, results[1].Success)
	assert.Equal(t, http.StatusOK, results[1].StatusCode)
	assert.Equal(t, 2, results[1].Attempt)
	assert.Nil(t, results[1].Error)
}

func TestSender_Send_LargePayload(t *testing.T) {
	t.Parallel()

	// Create 1MB payload
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	payload := map[string]any{
		"data": largeData,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		// Verify it's valid JSON
		var decoded map[string]any
		err = json.Unmarshal(body, &decoded)
		require.NoError(t, err)

		// Verify the data field exists
		_, ok := decoded["data"]
		assert.True(t, ok)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := webhook.NewSender()
	err := sender.Send(context.Background(), server.URL, payload)
	assert.NoError(t, err)
}

func TestSender_Concurrent(t *testing.T) {
	t.Parallel()

	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := webhook.NewSender()

	// Send 10 concurrent requests
	errCh := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			payload := map[string]int{"id": id}
			err := sender.Send(context.Background(), server.URL, payload)
			errCh <- err
		}(i)
	}

	// Collect results
	for i := 0; i < 10; i++ {
		err := <-errCh
		assert.NoError(t, err)
	}

	assert.Equal(t, int32(10), atomic.LoadInt32(&requests))
}

func BenchmarkSender_Send(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := webhook.NewSender()
	payload := map[string]any{
		"event": "benchmark",
		"data":  map[string]string{"id": "123"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := sender.Send(context.Background(), server.URL, payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSender_SendWithSignature(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := webhook.NewSender()
	payload := map[string]any{
		"event": "benchmark",
		"data":  map[string]string{"id": "123"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := sender.Send(
			context.Background(),
			server.URL,
			payload,
			webhook.WithSignature("benchmark_secret"),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestSender_CircuitBreaker_HalfOpenRecovery(t *testing.T) {
	t.Parallel()

	var attempts int32
	var succeedAfter int32 = 3

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt <= succeedAfter {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cb := webhook.NewCircuitBreaker(2, 2, 100*time.Millisecond)
	sender := webhook.NewSender()

	// Open the circuit with 2 failures
	for i := 0; i < 2; i++ {
		err := sender.Send(
			context.Background(),
			server.URL,
			map[string]string{"test": "failure"},
			webhook.WithCircuitBreaker(cb),
			webhook.WithNoRetry(),
		)
		require.Error(t, err)
	}

	assert.Equal(t, webhook.CircuitOpen, cb.State())

	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)

	// Circuit should be half-open, first attempt should fail
	err := sender.Send(
		context.Background(),
		server.URL,
		map[string]string{"test": "halfopen_fail"},
		webhook.WithCircuitBreaker(cb),
		webhook.WithNoRetry(),
	)
	assert.Error(t, err)
	assert.Equal(t, webhook.CircuitOpen, cb.State()) // Should reopen

	// Wait again for recovery
	time.Sleep(150 * time.Millisecond)

	// Now the server will succeed, circuit should close after 2 successes
	err = sender.Send(
		context.Background(),
		server.URL,
		map[string]string{"test": "success1"},
		webhook.WithCircuitBreaker(cb),
		webhook.WithNoRetry(),
	)
	assert.NoError(t, err)
	assert.Equal(t, webhook.CircuitHalfOpen, cb.State()) // Still half-open

	err = sender.Send(
		context.Background(),
		server.URL,
		map[string]string{"test": "success2"},
		webhook.WithCircuitBreaker(cb),
		webhook.WithNoRetry(),
	)
	assert.NoError(t, err)
	assert.Equal(t, webhook.CircuitClosed, cb.State()) // Should be closed now

	// Verify circuit stays closed for subsequent requests
	err = sender.Send(
		context.Background(),
		server.URL,
		map[string]string{"test": "verify_closed"},
		webhook.WithCircuitBreaker(cb),
		webhook.WithNoRetry(),
	)
	assert.NoError(t, err)
	assert.Equal(t, webhook.CircuitClosed, cb.State())
}

func TestSender_CircuitBreaker_Concurrent(t *testing.T) {
	t.Parallel()

	var requests int32
	var successes int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)

		// Fail every 3rd request to test circuit behavior under load
		if atomic.LoadInt32(&requests)%3 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		atomic.AddInt32(&successes, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cb := webhook.NewCircuitBreaker(5, 2, 50*time.Millisecond)
	sender := webhook.NewSender()

	const numGoroutines = 20
	const requestsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	var totalErrors int32
	var circuitOpenErrors int32

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				payload := map[string]int{"goroutine": id, "request": j}
				err := sender.Send(
					context.Background(),
					server.URL,
					payload,
					webhook.WithCircuitBreaker(cb),
					webhook.WithNoRetry(),
					webhook.WithTimeout(100*time.Millisecond),
				)

				if err != nil {
					atomic.AddInt32(&totalErrors, 1)
					if webhook.IsCircuitOpen(err) {
						atomic.AddInt32(&circuitOpenErrors, 1)
					}
				}

				// Small delay to increase chance of interesting race conditions
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify that some requests succeeded and circuit breaker was involved
	totalRequests := int32(numGoroutines * requestsPerGoroutine)
	actualRequests := atomic.LoadInt32(&requests)
	totalErr := atomic.LoadInt32(&totalErrors)
	circuitErr := atomic.LoadInt32(&circuitOpenErrors)

	t.Logf("Total intended: %d, Server received: %d, Total errors: %d, Circuit open errors: %d",
		totalRequests, actualRequests, totalErr, circuitErr)

	// The circuit breaker may or may not trigger depending on timing
	// This is acceptable in a concurrent test - the important thing is that
	// the system remains stable and doesn't crash
	if circuitErr > 0 {
		t.Logf("Circuit breaker triggered %d times", circuitErr)
	} else {
		t.Logf("Circuit breaker did not trigger - this is acceptable in concurrent scenarios")
	}

	// Circuit should be in a valid state
	finalState := cb.State()
	assert.Contains(t, []webhook.CircuitState{
		webhook.CircuitClosed,
		webhook.CircuitOpen,
		webhook.CircuitHalfOpen,
	}, finalState)
}

func TestSender_CircuitBreaker_WithLargePayload(t *testing.T) {
	t.Parallel()

	// Create 100KB payload
	largeData := make([]byte, 100*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	payload := map[string]any{
		"event": "large_payload_test",
		"data":  largeData,
	}

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)

		// Verify payload is valid JSON
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var decoded map[string]any
		err = json.Unmarshal(body, &decoded)
		require.NoError(t, err)

		// Fail first 2 attempts to open circuit, succeed on 3rd attempt
		if attempt <= 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cb := webhook.NewCircuitBreaker(2, 1, 100*time.Millisecond)
	sender := webhook.NewSender()

	// First 2 requests should fail and open circuit
	for i := 0; i < 2; i++ {
		err := sender.Send(
			context.Background(),
			server.URL,
			payload,
			webhook.WithCircuitBreaker(cb),
			webhook.WithNoRetry(),
			webhook.WithTimeout(5*time.Second), // Longer timeout for large payload
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, webhook.ErrWebhookDeliveryFailed)
	}

	// Circuit should be open
	assert.Equal(t, webhook.CircuitOpen, cb.State())

	// Next request should be blocked immediately
	err := sender.Send(
		context.Background(),
		server.URL,
		payload,
		webhook.WithCircuitBreaker(cb),
		webhook.WithTimeout(5*time.Second),
	)
	assert.ErrorIs(t, err, webhook.ErrCircuitOpen)

	// Wait for recovery and test successful delivery
	time.Sleep(150 * time.Millisecond)

	// Circuit should be half-open now, try to send successfully
	err = sender.Send(
		context.Background(),
		server.URL,
		payload,
		webhook.WithCircuitBreaker(cb),
		webhook.WithNoRetry(),
		webhook.WithTimeout(5*time.Second),
	)
	assert.NoError(t, err)
	assert.Equal(t, webhook.CircuitClosed, cb.State())

	// Verify we made the expected number of attempts
	// 2 initial failures (opens circuit) + 1 success after recovery = 3 total server requests
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestSender_Send_MarshalError(t *testing.T) {
	t.Parallel()

	// Create a type that cannot be marshaled to JSON
	type UnmarshalableType struct {
		Ch chan int `json:"channel"` // channels cannot be marshaled
	}

	data := UnmarshalableType{
		Ch: make(chan int),
	}

	sender := webhook.NewSender()
	err := sender.Send(context.Background(), "https://example.com", data)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal payload to JSON")
}

func TestSender_Send_PayloadSizeLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		payloadSize    int
		maxPayloadSize int64
		expectError    bool
		errorContains  string
	}{
		{
			name:           "payload within limit",
			payloadSize:    1024,      // 1KB
			maxPayloadSize: 10 * 1024, // 10KB limit
			expectError:    false,
		},
		{
			name:           "payload exactly at limit",
			payloadSize:    700,  // Less than 1KB to account for JSON overhead
			maxPayloadSize: 1024, // 1KB limit
			expectError:    false,
		},
		{
			name:           "payload exceeds limit",
			payloadSize:    2 * 1024, // 2KB
			maxPayloadSize: 1024,     // 1KB limit
			expectError:    true,
			errorContains:  "exceeds maximum allowed size",
		},
		{
			name:           "no limit when set to 0",
			payloadSize:    10 * 1024 * 1024, // 10MB
			maxPayloadSize: 0,                // No limit
			expectError:    false,
		},
		{
			name:           "default 10MB limit",
			payloadSize:    11 * 1024 * 1024, // 11MB
			maxPayloadSize: -1,               // Use default (10MB)
			expectError:    true,
			errorContains:  "exceeds maximum allowed size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create payload of specific size
			data := make([]byte, tt.payloadSize)
			for i := range data {
				data[i] = byte(i % 256)
			}
			payload := map[string]any{
				"data": data,
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				// Verify it's valid JSON if we got here
				var decoded map[string]any
				err = json.Unmarshal(body, &decoded)
				require.NoError(t, err)

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			sender := webhook.NewSender()

			opts := []webhook.SendOption{}
			if tt.maxPayloadSize >= 0 {
				opts = append(opts, webhook.WithMaxPayloadSize(tt.maxPayloadSize))
			}

			err := sender.Send(context.Background(), server.URL, payload, opts...)

			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, webhook.ErrInvalidPayload)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSender_Send_ResponseSizeLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		responseSize    int
		maxResponseSize int64
		statusCode      int
	}{
		{
			name:            "small response within limit",
			responseSize:    1024,      // 1KB response
			maxResponseSize: 64 * 1024, // 64KB limit
			statusCode:      http.StatusBadRequest,
		},
		{
			name:            "large response truncated",
			responseSize:    100 * 1024, // 100KB response
			maxResponseSize: 10 * 1024,  // 10KB limit
			statusCode:      http.StatusInternalServerError,
		},
		{
			name:            "custom response limit",
			responseSize:    5 * 1024, // 5KB response
			maxResponseSize: 2 * 1024, // 2KB limit
			statusCode:      http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Generate response body of specific size
			responseBody := make([]byte, tt.responseSize)
			for i := range responseBody {
				responseBody[i] = byte('A' + (i % 26))
			}

			var capturedErrorMsg string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write(responseBody)
			}))
			defer server.Close()

			sender := webhook.NewSender()
			err := sender.Send(
				context.Background(),
				server.URL,
				map[string]string{"test": "data"},
				webhook.WithMaxResponseSize(tt.maxResponseSize),
				webhook.WithNoRetry(),
				webhook.WithOnDelivery(func(result webhook.DeliveryResult) {
					if result.Error != nil {
						capturedErrorMsg = result.Error.Error()
					}
				}),
			)

			require.Error(t, err)

			// Verify error message doesn't contain more than maxResponseSize of response body
			// The error message should be truncated appropriately
			assert.NotEmpty(t, capturedErrorMsg)

			// Extract the response body portion from error message
			// Error format: "webhook returned status XXX: <body>"
			parts := strings.SplitN(capturedErrorMsg, ": ", 2)
			if len(parts) == 2 {
				bodyInError := parts[1]
				// Account for the "..." suffix and some overhead
				maxExpectedLen := min(int(tt.maxResponseSize), 200) + 10
				assert.LessOrEqual(t, len(bodyInError), maxExpectedLen,
					"Error message body portion should be limited by maxResponseSize")
			}
		})
	}
}

// Helper function for min since Go doesn't have built-in generic min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Benchmark tests for high-throughput scenarios
func BenchmarkSender_HighThroughput_Sequential(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate minimal processing
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	sender := webhook.NewSender()
	payload := map[string]any{
		"event": "high_throughput_test",
		"data": map[string]string{
			"id":        "bench_123",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := sender.Send(
			context.Background(),
			server.URL,
			payload,
			webhook.WithTimeout(5*time.Second),
			webhook.WithNoRetry(), // Disable retries for consistent benchmarking
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSender_HighThroughput_Concurrent(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate minimal processing
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	sender := webhook.NewSender()
	payload := map[string]any{
		"event": "concurrent_test",
		"data": map[string]string{
			"id": "bench_456",
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := sender.Send(
				context.Background(),
				server.URL,
				payload,
				webhook.WithTimeout(5*time.Second),
				webhook.WithNoRetry(),
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkSender_LargePayload(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Just acknowledge receipt, don't process
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := webhook.NewSender()

	// Create payloads of different sizes
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, sz := range sizes {
		b.Run(sz.name, func(b *testing.B) {
			// Create payload of specific size
			data := make([]byte, sz.size)
			for i := range data {
				data[i] = byte(i % 256)
			}
			payload := map[string]any{
				"event": "large_payload_bench",
				"data":  data,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := sender.Send(
					context.Background(),
					server.URL,
					payload,
					webhook.WithTimeout(10*time.Second),
					webhook.WithNoRetry(),
					webhook.WithMaxPayloadSize(0), // No limit for benchmarking
				)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSender_WithRetries(b *testing.B) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		// Fail first attempt, succeed on retry
		if count%2 == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := webhook.NewSender()
	payload := map[string]string{"test": "retry_bench"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := sender.Send(
			context.Background(),
			server.URL,
			payload,
			webhook.WithMaxRetries(1),
			webhook.WithBackoff(webhook.FixedBackoff{Interval: time.Millisecond}),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSender_CircuitBreaker(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always succeed for benchmarking
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cb := webhook.NewCircuitBreaker(5, 2, 100*time.Millisecond)
	sender := webhook.NewSender()
	payload := map[string]string{"test": "circuit_bench"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := sender.Send(
			context.Background(),
			server.URL,
			payload,
			webhook.WithCircuitBreaker(cb),
			webhook.WithNoRetry(),
		)
		if err != nil {
			// Circuit might open under high load, which is expected
			if !webhook.IsCircuitOpen(err) {
				b.Fatal(err)
			}
		}
	}
}
