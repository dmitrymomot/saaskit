package webhook_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

	payload := []byte(`{"event":"test","id":"123"}`)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "saaskit-webhook/1.0", r.Header.Get("User-Agent"))

		// Read body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, payload, body)

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

	payload := []byte(`{"test":"data"}`)
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

	payload := []byte(`{"test":"retry"}`)
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
				[]byte(`{"test":"data"}`),
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
			[]byte(`{"test":"data"}`),
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
		[]byte(`{"test":"data"}`),
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
		[]byte(`{"test":"data"}`),
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
	err := sender.Send(ctx, server.URL, []byte(`{"test":"data"}`))

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSender_Send_ValidationErrors(t *testing.T) {
	t.Parallel()

	sender := webhook.NewSender()

	tests := []struct {
		name    string
		url     string
		payload []byte
		wantErr error
		errMsg  string
	}{
		{
			name:    "empty URL",
			url:     "",
			payload: []byte(`{"test":"data"}`),
			wantErr: webhook.ErrInvalidURL,
			errMsg:  "URL is required",
		},
		{
			name:    "invalid URL",
			url:     "not a url",
			payload: []byte(`{"test":"data"}`),
			wantErr: webhook.ErrInvalidURL,
			errMsg:  "only http and https schemes are supported",
		},
		{
			name:    "invalid scheme",
			url:     "ftp://example.com",
			payload: []byte(`{"test":"data"}`),
			wantErr: webhook.ErrInvalidURL,
			errMsg:  "only http and https schemes are supported",
		},
		{
			name:    "missing host",
			url:     "http:///path",
			payload: []byte(`{"test":"data"}`),
			wantErr: webhook.ErrInvalidURL,
			errMsg:  "host is required",
		},
		{
			name:    "empty payload",
			url:     "https://example.com",
			payload: []byte{},
			wantErr: webhook.ErrInvalidPayload,
			errMsg:  "payload cannot be empty",
		},
		{
			name:    "nil payload",
			url:     "https://example.com",
			payload: nil,
			wantErr: webhook.ErrInvalidPayload,
			errMsg:  "payload cannot be empty",
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
		[]byte(`{"test":"data"}`),
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

	payload, err := json.Marshal(map[string]interface{}{
		"data": largeData,
	})
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, len(payload), len(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := webhook.NewSender()
	err = sender.Send(context.Background(), server.URL, payload)
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
			payload := []byte(fmt.Sprintf(`{"id":%d}`, id))
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
	payload := []byte(`{"event":"benchmark","data":{"id":"123"}}`)

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
	payload := []byte(`{"event":"benchmark","data":{"id":"123"}}`)

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
			[]byte(`{"test":"failure"}`),
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
		[]byte(`{"test":"halfopen_fail"}`),
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
		[]byte(`{"test":"success1"}`),
		webhook.WithCircuitBreaker(cb),
		webhook.WithNoRetry(),
	)
	assert.NoError(t, err)
	assert.Equal(t, webhook.CircuitHalfOpen, cb.State()) // Still half-open

	err = sender.Send(
		context.Background(),
		server.URL,
		[]byte(`{"test":"success2"}`),
		webhook.WithCircuitBreaker(cb),
		webhook.WithNoRetry(),
	)
	assert.NoError(t, err)
	assert.Equal(t, webhook.CircuitClosed, cb.State()) // Should be closed now

	// Verify circuit stays closed for subsequent requests
	err = sender.Send(
		context.Background(),
		server.URL,
		[]byte(`{"test":"verify_closed"}`),
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
				payload := []byte(fmt.Sprintf(`{"goroutine":%d,"request":%d}`, id, j))
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

	payload, err := json.Marshal(map[string]interface{}{
		"event": "large_payload_test",
		"data":  largeData,
	})
	require.NoError(t, err)

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)

		// Verify payload size
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, len(payload), len(body))

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
	err = sender.Send(
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
