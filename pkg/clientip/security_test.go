package clientip_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/clientip"
)

func TestSecurityAttackVectors(t *testing.T) {
	t.Run("header_injection_attacks", func(t *testing.T) {
		testHeaderInjectionAttacks(t)
	})

	t.Run("ip_spoofing_scenarios", func(t *testing.T) {
		testIPSpoofingScenarios(t)
	})

	t.Run("dos_attack_vectors", func(t *testing.T) {
		testDoSAttackVectors(t)
	})

	t.Run("trust_boundary_validation", func(t *testing.T) {
		testTrustBoundaryValidation(t)
	})
}

func testHeaderInjectionAttacks(t *testing.T) {
	t.Run("crlf_injection_in_headers", func(t *testing.T) {
		t.Parallel()

		injectionTests := []struct {
			name        string
			headerName  string
			headerValue string
			expected    string
			desc        string
		}{
			{
				name:        "crlf_in_cf_connecting_ip",
				headerName:  "CF-Connecting-IP",
				headerValue: "192.168.1.1\r\nX-Admin: true",
				expected:    "10.0.0.1", // Should fall back to RemoteAddr
				desc:        "CRLF injection in CF-Connecting-IP should be rejected",
			},
			{
				name:        "crlf_in_do_connecting_ip",
				headerName:  "DO-Connecting-IP",
				headerValue: "203.0.113.1\r\nHost: evil.com",
				expected:    "10.0.0.1", // Should fall back to RemoteAddr
				desc:        "CRLF injection in DO-Connecting-IP should be rejected",
			},
			{
				name:        "crlf_in_x_forwarded_for",
				headerName:  "X-Forwarded-For",
				headerValue: "198.51.100.1\r\nX-Injected: malicious",
				expected:    "10.0.0.1", // Should fall back to RemoteAddr
				desc:        "CRLF injection in X-Forwarded-For should be rejected",
			},
			{
				name:        "crlf_in_x_real_ip",
				headerName:  "X-Real-IP",
				headerValue: "127.0.0.1\nSet-Cookie: malicious=true",
				expected:    "10.0.0.1", // Should fall back to RemoteAddr
				desc:        "LF injection in X-Real-IP should be rejected",
			},
			{
				name:        "null_byte_injection_cf",
				headerName:  "CF-Connecting-IP",
				headerValue: "192.168.1.1\x00admin",
				expected:    "10.0.0.1", // Should fall back to RemoteAddr
				desc:        "Null byte injection should be rejected",
			},
			{
				name:        "tab_injection_attack",
				headerName:  "X-Forwarded-For",
				headerValue: "203.0.113.1\tX-Admin: true",
				expected:    "10.0.0.1", // Should fall back to RemoteAddr
				desc:        "Tab character injection should be rejected",
			},
		}

		for _, tt := range injectionTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					tt.headerName: tt.headerValue,
				}
				req := createTestRequest(headers, "10.0.0.1:8080")

				ip := clientip.GetIP(req)

				assert.Equal(t, tt.expected, ip, tt.desc)

				assert.NotContains(t, ip, "\r", "Result should not contain CR character")
				assert.NotContains(t, ip, "\n", "Result should not contain LF character")
				assert.NotContains(t, ip, "\x00", "Result should not contain null bytes")
				assert.NotContains(t, ip, "\t", "Result should not contain tab characters")
			})
		}
	})

	t.Run("header_value_overflow", func(t *testing.T) {
		t.Parallel()

		longIP := strings.Repeat("255.", 1000) + "1"

		headers := map[string]string{
			"CF-Connecting-IP": longIP,
		}
		req := createTestRequest(headers, "10.0.0.1:8080")

		start := time.Now()
		ip := clientip.GetIP(req)
		duration := time.Since(start)

		assert.Equal(t, "10.0.0.1", ip, "Should fall back to RemoteAddr for extremely long headers")
		assert.Less(t, duration, 10*time.Millisecond, "Should process long headers quickly")
	})

	t.Run("unicode_normalization_attacks", func(t *testing.T) {
		t.Parallel()

		unicodeTests := []struct {
			name     string
			header   string
			expected string
			desc     string
		}{
			{
				name:     "unicode_ip_confusion",
				header:   "１９２．１６８．１．１", // Full-width Unicode digits
				expected: "10.0.0.1",    // Should not parse Unicode digits as IP
				desc:     "Unicode digits should not be parsed as valid IP",
			},
			{
				name:     "unicode_dot_confusion",
				header:   "192․168․1․1", // Unicode bullet point instead of dot
				expected: "10.0.0.1",    // Should not parse Unicode bullets as dots
				desc:     "Unicode bullet points should not be treated as dots",
			},
			{
				name:     "mixed_encoding_attack",
				header:   "192.168.1.1\u2028admin", // Unicode line separator
				expected: "10.0.0.1",               // Should reject due to Unicode control char
				desc:     "Unicode line separators should be rejected",
			},
		}

		for _, tt := range unicodeTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"CF-Connecting-IP": tt.header,
				}
				req := createTestRequest(headers, "10.0.0.1:8080")

				ip := clientip.GetIP(req)
				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})
}

func testIPSpoofingScenarios(t *testing.T) {
	t.Run("proxy_chain_spoofing", func(t *testing.T) {
		t.Parallel()

		spoofingTests := []struct {
			name     string
			headers  map[string]string
			expected string
			desc     string
		}{
			{
				name: "public_to_private_spoofing",
				headers: map[string]string{
					"X-Forwarded-For": "8.8.8.8, 192.168.1.1, 10.0.0.1", // Public → Private spoofing
				},
				expected: "8.8.8.8", // Should take first valid public IP
				desc:     "Should not be fooled by private IPs in forwarded chain",
			},
			{
				name: "domain_injection_attempt",
				headers: map[string]string{
					"X-Forwarded-For": "203.0.113.1, evil.com, 127.0.0.1",
				},
				expected: "203.0.113.1", // Should ignore invalid domain
				desc:     "Should ignore domain names in forwarded chain",
			},
			{
				name: "localhost_spoofing_attempt",
				headers: map[string]string{
					"CF-Connecting-IP": "127.0.0.1", // Attacker claims to be localhost
					"X-Forwarded-For":  "8.8.8.8",   // Real IP in secondary header
				},
				expected: "127.0.0.1", // CF header has priority, even if suspicious
				desc:     "Should respect header priority even for suspicious IPs",
			},
			{
				name: "ipv4_in_ipv6_spoofing",
				headers: map[string]string{
					"DO-Connecting-IP": "::ffff:192.168.1.1", // IPv4-mapped IPv6 private
				},
				expected: "192.168.1.1", // Go normalizes IPv4-mapped to IPv4
				desc:     "Should handle IPv4-mapped IPv6 addresses correctly",
			},
			{
				name: "multiple_header_confusion",
				headers: map[string]string{
					"CF-Connecting-IP": "",             // Empty highest priority
					"DO-Connecting-IP": "attacker.com", // Invalid domain
					"X-Forwarded-For":  "192.168.1.1",  // Private IP
					"X-Real-IP":        "203.0.113.1",  // Valid public IP
				},
				expected: "192.168.1.1", // Should follow priority order
				desc:     "Should follow header priority even with mixed invalid headers",
			},
		}

		for _, tt := range spoofingTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				req := createTestRequest(tt.headers, "10.0.0.1:8080")
				ip := clientip.GetIP(req)

				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})

	t.Run("cdn_bypass_attempts", func(t *testing.T) {
		t.Parallel()

		bypassTests := []struct {
			name     string
			headers  map[string]string
			expected string
			desc     string
		}{
			{
				name: "fake_cloudflare_header",
				headers: map[string]string{
					"CF-Connecting-IP": "8.8.8.8",     // Attacker sets fake CF header
					"X-Forwarded-For":  "203.0.113.1", // Real forwarded IP
				},
				expected: "8.8.8.8", // CF header takes priority
				desc:     "Should respect header priority regardless of authenticity concerns",
			},
			{
				name: "mixed_cdn_headers",
				headers: map[string]string{
					"CF-Connecting-IP": "203.0.113.1",  // Cloudflare
					"DO-Connecting-IP": "198.51.100.1", // DigitalOcean
					"X-Forwarded-For":  "8.8.8.8",      // Generic proxy
					"X-Real-IP":        "1.1.1.1",      // Another proxy
					"Fly-Client-IP":    "9.9.9.9",      // Fly.io (not in priority list)
				},
				expected: "203.0.113.1", // CF should win
				desc:     "Should follow priority order with multiple CDN headers",
			},
		}

		for _, tt := range bypassTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				req := createTestRequest(tt.headers, "127.0.0.1:8080")
				ip := clientip.GetIP(req)

				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})
}

func testDoSAttackVectors(t *testing.T) {
	t.Run("extremely_long_forwarded_chains", func(t *testing.T) {
		t.Parallel()

		const chainLength = 50000
		longChain := strings.Repeat("invalid,", chainLength-1) + "203.0.113.1"

		headers := map[string]string{
			"X-Forwarded-For": longChain,
		}
		req := createTestRequest(headers, "10.0.0.1:8080")

		start := time.Now()
		ip := clientip.GetIP(req)
		duration := time.Since(start)

		assert.Equal(t, "203.0.113.1", ip, "Should find valid IP even in very long chain")
		assert.Less(t, duration, 100*time.Millisecond, "Should process long chains within reasonable time")

		t.Logf("Processed chain of %d IPs in %v", chainLength, duration)
	})

	t.Run("memory_exhaustion_attack", func(t *testing.T) {
		t.Parallel()

		headers := map[string]string{}

		for i := 0; i < 1000; i++ {
			headerName := fmt.Sprintf("X-Custom-Header-%d", i)
			headerValue := strings.Repeat("x", 8192) // 8KB per header
			headers[headerName] = headerValue
		}

		headers["CF-Connecting-IP"] = "203.0.113.1"

		req := createTestRequest(headers, "10.0.0.1:8080")

		start := time.Now()
		ip := clientip.GetIP(req)
		duration := time.Since(start)

		assert.Equal(t, "203.0.113.1", ip, "Should extract IP despite many large headers")
		assert.Less(t, duration, 10*time.Millisecond, "Should not be slowed down by irrelevant large headers")
	})

	t.Run("recursive_parsing_attack", func(t *testing.T) {
		t.Parallel()

		recursivePatterns := []string{
			strings.Repeat("((", 1000) + "203.0.113.1" + strings.Repeat("))", 1000),
			strings.Repeat("[", 500) + "203.0.113.1" + strings.Repeat("]", 500),
			strings.Repeat("{{", 250) + "203.0.113.1" + strings.Repeat("}}", 250),
		}

		for i, pattern := range recursivePatterns {
			t.Run(fmt.Sprintf("recursive_pattern_%d", i), func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"X-Forwarded-For": pattern,
				}
				req := createTestRequest(headers, "10.0.0.1:8080")

				start := time.Now()
				ip := clientip.GetIP(req)
				duration := time.Since(start)

				assert.NotEmpty(t, ip, "Should return some IP")
				assert.Less(t, duration, 50*time.Millisecond, "Should not spend excessive time on recursive patterns")
			})
		}
	})

	t.Run("repeated_header_attack", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:8080"

		for i := 0; i < 10000; i++ {
			req.Header.Add("X-Forwarded-For", fmt.Sprintf("192.168.1.%d", i%255+1))
		}
		req.Header.Add("X-Forwarded-For", "203.0.113.1")

		start := time.Now()
		ip := clientip.GetIP(req)
		duration := time.Since(start)

		// Note: Go's header handling may optimize this, but we should still be reasonably fast
		assert.NotEmpty(t, ip, "Should return some IP despite header bombing")
		assert.Less(t, duration, 100*time.Millisecond, "Should handle repeated headers efficiently")

		t.Logf("Processed 10000+ repeated headers in %v, got IP: %s", duration, ip)
	})
}

func testTrustBoundaryValidation(t *testing.T) {
	t.Run("private_ip_handling", func(t *testing.T) {
		t.Parallel()

		privateIPTests := []struct {
			name     string
			headers  map[string]string
			expected string
			desc     string
		}{
			{
				name: "private_ip_in_cf_header",
				headers: map[string]string{
					"CF-Connecting-IP": "192.168.1.100", // Private IP from "trusted" source
				},
				expected: "192.168.1.100",
				desc:     "Should accept private IPs from trusted headers",
			},
			{
				name: "private_vs_public_priority",
				headers: map[string]string{
					"CF-Connecting-IP": "10.0.0.1",    // Private IP in high priority header
					"X-Forwarded-For":  "203.0.113.1", // Public IP in lower priority header
				},
				expected: "10.0.0.1", // Should respect priority over public/private distinction
				desc:     "Header priority should override public/private preference",
			},
			{
				name: "mixed_private_public_chain",
				headers: map[string]string{
					"X-Forwarded-For": "203.0.113.1, 192.168.1.1, 10.0.0.1, 172.16.0.1",
				},
				expected: "203.0.113.1", // Should take first valid IP (public)
				desc:     "Should take first valid IP in chain",
			},
			{
				name: "all_private_chain",
				headers: map[string]string{
					"X-Forwarded-For": "192.168.1.1, 10.0.0.1, 172.16.0.1",
				},
				expected: "192.168.1.1", // Should take first valid private IP
				desc:     "Should handle all-private chains correctly",
			},
		}

		for _, tt := range privateIPTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				req := createTestRequest(tt.headers, "127.0.0.1:8080")
				ip := clientip.GetIP(req)

				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})

	t.Run("localhost_and_loopback_handling", func(t *testing.T) {
		t.Parallel()

		loopbackTests := []struct {
			name     string
			headers  map[string]string
			expected string
			desc     string
		}{
			{
				name: "ipv4_loopback",
				headers: map[string]string{
					"CF-Connecting-IP": "127.0.0.1",
				},
				expected: "127.0.0.1",
				desc:     "Should handle IPv4 loopback correctly",
			},
			{
				name: "ipv6_loopback",
				headers: map[string]string{
					"DO-Connecting-IP": "::1",
				},
				expected: "::1",
				desc:     "Should handle IPv6 loopback correctly",
			},
			{
				name: "mixed_loopback_chain",
				headers: map[string]string{
					"X-Forwarded-For": "127.0.0.1, ::1, 203.0.113.1",
				},
				expected: "127.0.0.1", // Should take first valid IP
				desc:     "Should handle mixed loopback addresses",
			},
		}

		for _, tt := range loopbackTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				req := createTestRequest(tt.headers, "10.0.0.1:8080")
				ip := clientip.GetIP(req)

				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})

	t.Run("zero_and_broadcast_addresses", func(t *testing.T) {
		t.Parallel()

		specialIPTests := []struct {
			name     string
			header   string
			expected string
			desc     string
		}{
			{
				name:     "zero_address",
				header:   "0.0.0.0",
				expected: "10.0.0.1", // Should fall back to RemoteAddr
				desc:     "Should reject 0.0.0.0 as invalid",
			},
			{
				name:     "broadcast_address",
				header:   "255.255.255.255",
				expected: "255.255.255.255", // Actually valid IP, should accept
				desc:     "Should handle broadcast address",
			},
			{
				name:     "ipv6_zero",
				header:   "::",
				expected: "::", // Valid IPv6 zero address
				desc:     "Should handle IPv6 zero address",
			},
		}

		for _, tt := range specialIPTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"CF-Connecting-IP": tt.header,
				}
				req := createTestRequest(headers, "10.0.0.1:8080")

				ip := clientip.GetIP(req)
				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})
}

func TestConcurrentSafety(t *testing.T) {
	t.Run("concurrent_ip_extraction", func(t *testing.T) {
		t.Parallel()

		const numGoroutines = 100
		const numIterations = 10

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*numIterations)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < numIterations; j++ {
					expectedIP := fmt.Sprintf("203.0.113.%d", (goroutineID*numIterations+j)%255+1)

					headers := map[string]string{
						"CF-Connecting-IP": expectedIP,
						"X-Forwarded-For":  fmt.Sprintf("192.168.1.%d", goroutineID%255+1),
					}

					req := createTestRequest(headers, "10.0.0.1:8080")
					ip := clientip.GetIP(req)

					if ip != expectedIP {
						errors <- fmt.Errorf("goroutine %d iteration %d: expected IP %s, got %s",
							goroutineID, j, expectedIP, ip)
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		var errorCount int
		for err := range errors {
			t.Errorf("Concurrent access error: %v", err)
			errorCount++
		}

		assert.Equal(t, 0, errorCount, "Expected no errors in concurrent IP extraction")
	})

	t.Run("race_condition_detection", func(t *testing.T) {
		t.Parallel()

		const duration = 1 * time.Second
		const numGoroutines = 20

		var wg sync.WaitGroup
		stop := make(chan struct{})

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				req := createTestRequest(map[string]string{
					"CF-Connecting-IP": fmt.Sprintf("203.0.113.%d", goroutineID%255+1),
				}, "10.0.0.1:8080")

				for {
					select {
					case <-stop:
						return
					default:
						_ = clientip.GetIP(req)
					}
				}
			}(i)
		}

		time.Sleep(duration)
		close(stop)
		wg.Wait()

	})
}

func createTestRequestWithHelper(t *testing.T, headers map[string]string, remoteAddr string) *http.Request {
	t.Helper()

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = remoteAddr

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req
}
