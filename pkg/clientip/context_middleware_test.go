package clientip_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/clientip"
)

func TestContextIntegration(t *testing.T) {
	t.Run("set_and_get_ip_from_context", func(t *testing.T) {
		testSetAndGetIPFromContext(t)
	})

	t.Run("context_edge_cases", func(t *testing.T) {
		testContextEdgeCases(t)
	})

	t.Run("context_isolation", func(t *testing.T) {
		testContextIsolation(t)
	})
}

func testSetAndGetIPFromContext(t *testing.T) {
	t.Run("basic_context_operations", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		testIP := "203.0.113.195"

		newCtx := clientip.SetIPToContext(ctx, testIP)
		require.NotNil(t, newCtx, "SetIPToContext should return non-nil context")

		retrievedIP := clientip.GetIPFromContext(newCtx)
		assert.Equal(t, testIP, retrievedIP, "GetIPFromContext should return the stored IP")
	})

	t.Run("multiple_context_operations", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		ctx1 := clientip.SetIPToContext(ctx, "192.168.1.1")
		ctx2 := clientip.SetIPToContext(ctx1, "10.0.0.1")
		ctx3 := clientip.SetIPToContext(ctx2, "203.0.113.1")

		ip := clientip.GetIPFromContext(ctx3)
		assert.Equal(t, "203.0.113.1", ip, "Should get the most recently set IP")

		ip1 := clientip.GetIPFromContext(ctx1)
		assert.Equal(t, "192.168.1.1", ip1, "Original context should retain its IP")

		ip2 := clientip.GetIPFromContext(ctx2)
		assert.Equal(t, "10.0.0.1", ip2, "Intermediate context should retain its IP")
	})

	t.Run("ipv6_in_context", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ipv6Address := "2001:db8:85a3:8d3:1319:8a2e:370:7348"

		newCtx := clientip.SetIPToContext(ctx, ipv6Address)
		retrievedIP := clientip.GetIPFromContext(newCtx)

		assert.Equal(t, ipv6Address, retrievedIP, "Should handle IPv6 addresses correctly")
	})

	t.Run("empty_and_invalid_ips", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		emptyCtx := clientip.SetIPToContext(ctx, "")
		emptyIP := clientip.GetIPFromContext(emptyCtx)
		assert.Equal(t, "", emptyIP, "Should handle empty IP strings")

		invalidCtx := clientip.SetIPToContext(ctx, "not-an-ip")
		invalidIP := clientip.GetIPFromContext(invalidCtx)
		assert.Equal(t, "not-an-ip", invalidIP, "Should store invalid IP strings as-is")
	})
}

func testContextEdgeCases(t *testing.T) {
	t.Run("get_from_empty_context", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		ip := clientip.GetIPFromContext(ctx)
		assert.Equal(t, "", ip, "Should return empty string for context without IP")
	})

	t.Run("get_from_nil_context", func(t *testing.T) {
		t.Parallel()

		// Test with nil context (will panic - this is expected behavior)
		assert.Panics(t, func() {
			clientip.GetIPFromContext(nil)
		}, "Should panic when context is nil")
	})

	t.Run("set_with_nil_context", func(t *testing.T) {
		t.Parallel()

		// Test setting IP with nil context (will panic - this is expected behavior)
		assert.Panics(t, func() {
			clientip.SetIPToContext(nil, "203.0.113.1")
		}, "Should panic when setting IP to nil context")
	})

	t.Run("context_with_other_values", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(context.Background(), "other-key", "other-value")
		ctx = clientip.SetIPToContext(ctx, "203.0.113.1")

		ip := clientip.GetIPFromContext(ctx)
		otherValue := ctx.Value("other-key")

		assert.Equal(t, "203.0.113.1", ip, "IP should be stored correctly")
		assert.Equal(t, "other-value", otherValue, "Other context values should be preserved")
	})
}

func testContextIsolation(t *testing.T) {
	t.Run("context_branching", func(t *testing.T) {
		t.Parallel()

		baseCtx := context.Background()
		baseCtx = clientip.SetIPToContext(baseCtx, "192.168.1.1")

		branch1 := clientip.SetIPToContext(baseCtx, "203.0.113.1")
		branch2 := clientip.SetIPToContext(baseCtx, "198.51.100.1")

		ip1 := clientip.GetIPFromContext(branch1)
		ip2 := clientip.GetIPFromContext(branch2)
		baseIP := clientip.GetIPFromContext(baseCtx)

		assert.Equal(t, "203.0.113.1", ip1, "Branch 1 should have its own IP")
		assert.Equal(t, "198.51.100.1", ip2, "Branch 2 should have its own IP")
		assert.Equal(t, "192.168.1.1", baseIP, "Base context should be unchanged")
	})

	t.Run("context_immutability", func(t *testing.T) {
		t.Parallel()

		originalCtx := context.Background()
		originalCtx = clientip.SetIPToContext(originalCtx, "original-ip")

		newCtx := clientip.SetIPToContext(originalCtx, "new-ip")

		originalIP := clientip.GetIPFromContext(originalCtx)
		newIP := clientip.GetIPFromContext(newCtx)

		assert.Equal(t, "original-ip", originalIP, "Original context should be immutable")
		assert.Equal(t, "new-ip", newIP, "New context should have new IP")
	})
}

func TestMiddlewareIntegration(t *testing.T) {
	t.Run("basic_middleware_functionality", func(t *testing.T) {
		testBasicMiddlewareFunctionality(t)
	})

	t.Run("middleware_error_scenarios", func(t *testing.T) {
		testMiddlewareErrorScenarios(t)
	})

	t.Run("middleware_with_various_headers", func(t *testing.T) {
		testMiddlewareWithVariousHeaders(t)
	})
}

func testBasicMiddlewareFunctionality(t *testing.T) {
	t.Run("middleware_sets_context_correctly", func(t *testing.T) {
		t.Parallel()

		handlerCalled := false
		var contextIP string

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			contextIP = clientip.GetIPFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		middlewareHandler := clientip.Middleware(handler)

		req := createTestRequest(map[string]string{
			"CF-Connecting-IP": "203.0.113.195",
		}, "10.0.0.1:8080")

		w := httptest.NewRecorder()

		middlewareHandler.ServeHTTP(w, req)

		assert.True(t, handlerCalled, "Handler should have been called")
		assert.Equal(t, "203.0.113.195", contextIP, "Context should contain the extracted IP")
		assert.Equal(t, http.StatusOK, w.Code, "Response should be OK")
	})

	t.Run("middleware_preserves_handler_functionality", func(t *testing.T) {
		t.Parallel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientip.GetIPFromContext(r.Context())
			fmt.Fprintf(w, "Your IP is: %s", ip)
		})

		middlewareHandler := clientip.Middleware(handler)

		req := createTestRequest(map[string]string{
			"DO-Connecting-IP": "198.51.100.178",
		}, "127.0.0.1:8080")

		w := httptest.NewRecorder()
		middlewareHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "Your IP is: 198.51.100.178", w.Body.String())
	})

	t.Run("middleware_with_multiple_handlers", func(t *testing.T) {
		t.Parallel()

		handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientip.GetIPFromContext(r.Context())
			w.Header().Set("X-Client-IP", ip)
			w.WriteHeader(http.StatusOK)
		})

		handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientip.GetIPFromContext(r.Context())
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"client_ip": "%s"}`, ip)
		})

		mw1 := clientip.Middleware(handler1)
		mw2 := clientip.Middleware(handler2)

		req1 := createTestRequest(map[string]string{
			"X-Real-IP": "192.168.1.100",
		}, "10.0.0.1:8080")
		w1 := httptest.NewRecorder()
		mw1.ServeHTTP(w1, req1)

		assert.Equal(t, "192.168.1.100", w1.Header().Get("X-Client-IP"))

		req2 := createTestRequest(map[string]string{
			"X-Forwarded-For": "203.0.113.50",
		}, "10.0.0.1:8080")
		w2 := httptest.NewRecorder()
		mw2.ServeHTTP(w2, req2)

		assert.Contains(t, w2.Body.String(), "203.0.113.50")
		assert.Equal(t, "application/json", w2.Header().Get("Content-Type"))
	})
}

func testMiddlewareErrorScenarios(t *testing.T) {

	t.Run("middleware_with_panicking_handler", func(t *testing.T) {
		t.Parallel()

		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientip.GetIPFromContext(r.Context())
			assert.Equal(t, "203.0.113.1", ip, "IP should be set even if handler panics")
			panic("test panic")
		})

		middlewareHandler := clientip.Middleware(panicHandler)

		req := createTestRequest(map[string]string{
			"CF-Connecting-IP": "203.0.113.1",
		}, "10.0.0.1:8080")
		w := httptest.NewRecorder()

		assert.Panics(t, func() {
			middlewareHandler.ServeHTTP(w, req)
		}, "Handler panic should propagate")
	})

	t.Run("middleware_with_malformed_remote_addr", func(t *testing.T) {
		t.Parallel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientip.GetIPFromContext(r.Context())
			assert.NotEmpty(t, ip, "Should have some IP even with malformed RemoteAddr")
			w.WriteHeader(http.StatusOK)
		})

		middlewareHandler := clientip.Middleware(handler)

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "invalid-address" // This should trigger the error path
		req.Header.Set("CF-Connecting-IP", "")

		w := httptest.NewRecorder()
		middlewareHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Should handle malformed RemoteAddr gracefully")
	})
}

func testMiddlewareWithVariousHeaders(t *testing.T) {
	t.Run("middleware_header_priority", func(t *testing.T) {
		t.Parallel()

		headerTests := []struct {
			name     string
			headers  map[string]string
			expected string
		}{
			{
				name: "cf_priority",
				headers: map[string]string{
					"CF-Connecting-IP": "203.0.113.1",
					"DO-Connecting-IP": "198.51.100.1",
					"X-Forwarded-For":  "192.168.1.1",
				},
				expected: "203.0.113.1",
			},
			{
				name: "do_priority",
				headers: map[string]string{
					"DO-Connecting-IP": "198.51.100.1",
					"X-Forwarded-For":  "192.168.1.1",
					"X-Real-IP":        "10.0.0.1",
				},
				expected: "198.51.100.1",
			},
			{
				name: "forwarded_for_fallback",
				headers: map[string]string{
					"X-Forwarded-For": "192.168.1.1, 203.0.113.1",
					"X-Real-IP":       "10.0.0.1",
				},
				expected: "192.168.1.1",
			},
		}

		for _, tt := range headerTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var contextIP string
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					contextIP = clientip.GetIPFromContext(r.Context())
					w.WriteHeader(http.StatusOK)
				})

				middlewareHandler := clientip.Middleware(handler)
				req := createTestRequest(tt.headers, "127.0.0.1:8080")
				w := httptest.NewRecorder()

				middlewareHandler.ServeHTTP(w, req)

				assert.Equal(t, tt.expected, contextIP,
					"Middleware should respect header priority for %s", tt.name)
			})
		}
	})

	t.Run("middleware_with_ipv6", func(t *testing.T) {
		t.Parallel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientip.GetIPFromContext(r.Context())
			assert.Equal(t, "2001:db8::1", ip, "Should handle IPv6 addresses")
			w.WriteHeader(http.StatusOK)
		})

		middlewareHandler := clientip.Middleware(handler)

		req := createTestRequest(map[string]string{
			"DO-Connecting-IP": "2001:db8::1",
		}, "[::1]:8080")

		w := httptest.NewRecorder()
		middlewareHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("middleware_stress_test", func(t *testing.T) {
		t.Parallel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientip.GetIPFromContext(r.Context())
			assert.NotEmpty(t, ip, "Should always have an IP")
			w.WriteHeader(http.StatusOK)
		})

		middlewareHandler := clientip.Middleware(handler)

		for i := 0; i < 1000; i++ {
			headers := map[string]string{
				"CF-Connecting-IP": fmt.Sprintf("203.0.113.%d", i%255+1),
			}
			req := createTestRequest(headers, "10.0.0.1:8080")
			w := httptest.NewRecorder()

			middlewareHandler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i)
		}
	})
}

func TestIntegratedIPExtractionAndContext(t *testing.T) {
	t.Run("direct_vs_middleware_consistency", func(t *testing.T) {
		t.Parallel()

		headers := map[string]string{
			"CF-Connecting-IP": "203.0.113.195",
			"X-Forwarded-For":  "192.168.1.1",
		}

		req1 := createTestRequest(headers, "10.0.0.1:8080")
		directIP := clientip.GetIP(req1)

		var middlewareIP string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareIP = clientip.GetIPFromContext(r.Context())
		})

		middlewareHandler := clientip.Middleware(handler)
		req2 := createTestRequest(headers, "10.0.0.1:8080")
		w := httptest.NewRecorder()
		middlewareHandler.ServeHTTP(w, req2)

		assert.Equal(t, directIP, middlewareIP,
			"Direct extraction and middleware should produce same result")
		assert.Equal(t, "203.0.113.195", directIP, "Both should extract correct IP")
	})

	t.Run("context_persistence_through_chain", func(t *testing.T) {
		t.Parallel()

		var capturedIPs []string

		middleware1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ip := clientip.GetIPFromContext(r.Context())
				capturedIPs = append(capturedIPs, ip)
				next.ServeHTTP(w, r)
			})
		}

		middleware2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ip := clientip.GetIPFromContext(r.Context())
				capturedIPs = append(capturedIPs, ip)
				next.ServeHTTP(w, r)
			})
		}

		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientip.GetIPFromContext(r.Context())
			capturedIPs = append(capturedIPs, ip)
			w.WriteHeader(http.StatusOK)
		})

		chain := clientip.Middleware(middleware1(middleware2(finalHandler)))

		req := createTestRequest(map[string]string{
			"DO-Connecting-IP": "198.51.100.178",
		}, "127.0.0.1:8080")

		w := httptest.NewRecorder()
		chain.ServeHTTP(w, req)

		require.Len(t, capturedIPs, 3, "Should have captured IP at each middleware level")
		for i, ip := range capturedIPs {
			assert.Equal(t, "198.51.100.178", ip,
				"IP should be consistent at middleware level %d", i)
		}
	})
}
