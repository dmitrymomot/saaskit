package clientip_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/clientip"
)

func TestGetIP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name: "CF-Connecting-IP priority (Cloudflare → DigitalOcean Apps)",
			headers: map[string]string{
				"CF-Connecting-IP": "203.0.113.195",
				"DO-Connecting-IP": "198.51.100.178",
				"X-Forwarded-For":  "192.168.1.1",
				"X-Real-IP":        "10.0.0.1",
			},
			remoteAddr: "172.16.0.1:54321",
			expected:   "203.0.113.195",
		},
		{
			name: "DO-Connecting-IP priority (DigitalOcean App Platform)",
			headers: map[string]string{
				"DO-Connecting-IP": "198.51.100.178",
				"X-Forwarded-For":  "192.168.1.1, 10.0.0.1",
				"X-Real-IP":        "172.16.0.1",
			},
			remoteAddr: "10.0.0.1:54321",
			expected:   "198.51.100.178",
		},
		{
			name: "X-Forwarded-For when no CF or DO headers",
			headers: map[string]string{
				"X-Forwarded-For": "198.51.100.178, 203.0.113.195",
				"X-Real-IP":       "192.168.1.1",
			},
			remoteAddr: "10.0.0.1:54321",
			expected:   "198.51.100.178",
		},
		{
			name: "X-Real-IP when no forwarded headers",
			headers: map[string]string{
				"X-Real-IP": "192.168.1.1",
			},
			remoteAddr: "10.0.0.1:54321",
			expected:   "192.168.1.1",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "127.0.0.1:8080",
			expected:   "127.0.0.1",
		},
		{
			name: "Invalid CF header falls back to DO header",
			headers: map[string]string{
				"CF-Connecting-IP": "invalid-ip",
				"DO-Connecting-IP": "198.51.100.178",
			},
			remoteAddr: "10.0.0.1:54321",
			expected:   "198.51.100.178",
		},
		{
			name: "Invalid DO header falls back to X-Forwarded-For",
			headers: map[string]string{
				"DO-Connecting-IP": "not-an-ip",
				"X-Forwarded-For":  "198.51.100.178",
			},
			remoteAddr: "10.0.0.1:54321",
			expected:   "198.51.100.178",
		},
		{
			name: "IPv6 address handling",
			headers: map[string]string{
				"DO-Connecting-IP": "2001:db8::1",
			},
			remoteAddr: "[::1]:8080",
			expected:   "2001:db8::1",
		},
		{
			name:       "IPv6 RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "[2001:db8::1]:8080",
			expected:   "2001:db8::1",
		},
		{
			name: "X-Forwarded-For with spaces",
			headers: map[string]string{
				"X-Forwarded-For": " 198.51.100.178 , 203.0.113.195 ",
			},
			remoteAddr: "10.0.0.1:54321",
			expected:   "198.51.100.178",
		},
		{
			name: "X-Forwarded-For with many IPs (tests SplitSeq efficiency)",
			headers: map[string]string{
				"X-Forwarded-For": "invalid1, invalid2, invalid3, invalid4, 198.51.100.178, 203.0.113.195, 192.168.1.1",
			},
			remoteAddr: "10.0.0.1:54321",
			expected:   "198.51.100.178",
		},
		{
			name: "Multiple invalid IPs in X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "invalid, also-invalid, 198.51.100.178",
				"X-Real-IP":       "192.168.1.1",
			},
			remoteAddr: "10.0.0.1:54321",
			expected:   "198.51.100.178",
		},
		{
			name: "Empty headers",
			headers: map[string]string{
				"CF-Connecting-IP": "",
				"DO-Connecting-IP": "",
				"X-Forwarded-For":  "",
				"X-Real-IP":        "",
			},
			remoteAddr: "127.0.0.1:8080",
			expected:   "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := createTestRequest(tt.headers, tt.remoteAddr)
			ip := clientip.GetIP(req)

			if ip != tt.expected {
				t.Errorf("Expected IP %s, got %s", tt.expected, ip)
			}
		})
	}
}

func TestGetIPConsistency(t *testing.T) {
	t.Parallel()
	headers := map[string]string{
		"DO-Connecting-IP": "198.51.100.178",
		"X-Forwarded-For":  "192.168.1.1",
		"User-Agent":       "Test/1.0",
	}

	req1 := createTestRequest(headers, "10.0.0.1:54321")
	req2 := createTestRequest(headers, "10.0.0.1:54321")

	ip1 := clientip.GetIP(req1)
	ip2 := clientip.GetIP(req2)

	if ip1 != ip2 {
		t.Errorf("Expected identical IPs for identical requests, got %s and %s", ip1, ip2)
	}

	if ip1 != "198.51.100.178" {
		t.Errorf("Expected 198.51.100.178, got %s", ip1)
	}
}

func TestGetIPDifferentHeaders(t *testing.T) {
	t.Parallel()
	req1 := createTestRequest(map[string]string{
		"CF-Connecting-IP": "203.0.113.195",
	}, "10.0.0.1:54321")

	req2 := createTestRequest(map[string]string{
		"DO-Connecting-IP": "198.51.100.178",
	}, "10.0.0.1:54321")

	ip1 := clientip.GetIP(req1)
	ip2 := clientip.GetIP(req2)

	if ip1 == ip2 {
		t.Errorf("Expected different IPs for different headers, both got %s", ip1)
	}

	if ip1 != "203.0.113.195" {
		t.Errorf("Expected CF IP 203.0.113.195, got %s", ip1)
	}

	if ip2 != "198.51.100.178" {
		t.Errorf("Expected DO IP 198.51.100.178, got %s", ip2)
	}
}

func TestDigitalOceanAppPlatformScenarios(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		headers  map[string]string
		expected string
		scenario string
	}{
		{
			name: "Direct DigitalOcean Apps deployment",
			headers: map[string]string{
				"DO-Connecting-IP": "203.0.113.195",
				"X-Forwarded-For":  "10.244.0.1", // Internal DO network
			},
			expected: "203.0.113.195",
			scenario: "Primary DO Apps header should be used over internal IPs",
		},
		{
			name: "Cloudflare + DigitalOcean Apps",
			headers: map[string]string{
				"CF-Connecting-IP": "198.51.100.178", // Real client IP from CF
				"DO-Connecting-IP": "203.0.113.195",  // CF edge server IP
				"X-Forwarded-For":  "10.244.0.1",     // Internal DO network
			},
			expected: "198.51.100.178",
			scenario: "CF header should take priority over DO header",
		},
		{
			name: "DigitalOcean Load Balancer",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.195, 10.244.0.1",
				"X-Real-IP":       "203.0.113.195",
			},
			expected: "203.0.113.195",
			scenario: "Standard load balancer headers when DO-Connecting-IP not available",
		},
		{
			name: "Mixed invalid and valid headers",
			headers: map[string]string{
				"CF-Connecting-IP": "",                         // Empty
				"DO-Connecting-IP": "not-an-ip",                // Invalid
				"X-Forwarded-For":  "invalid, , 203.0.113.195", // Mixed
			},
			expected: "203.0.113.195",
			scenario: "Should parse through invalid entries to find valid IP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := createTestRequest(tt.headers, "10.0.0.1:54321")
			ip := clientip.GetIP(req)

			if ip != tt.expected {
				t.Errorf("Scenario: %s\nExpected IP %s, got %s", tt.scenario, tt.expected, ip)
			}
		})
	}
}

func BenchmarkGetIP(b *testing.B) {
	headers := map[string]string{
		"CF-Connecting-IP": "203.0.113.195",
		"DO-Connecting-IP": "198.51.100.178",
		"X-Forwarded-For":  "192.168.1.1, 10.0.0.1",
		"X-Real-IP":        "172.16.0.1",
		"User-Agent":       "Mozilla/5.0 (Test Browser)",
	}
	req := createTestRequest(headers, "10.0.0.1:54321")

	b.ResetTimer()
	for b.Loop() {
		clientip.GetIP(req)
	}
}

func BenchmarkGetIPFallback(b *testing.B) {
	// Test performance when falling back to RemoteAddr
	headers := map[string]string{}
	req := createTestRequest(headers, "192.168.1.100:12345")

	b.ResetTimer()
	for b.Loop() {
		clientip.GetIP(req)
	}
}

func BenchmarkGetIPWithLongForwardedChain(b *testing.B) {
	// Test SplitSeq efficiency with a long chain of IPs
	headers := map[string]string{
		"X-Forwarded-For": "invalid1, invalid2, invalid3, invalid4, invalid5, invalid6, 203.0.113.195, 198.51.100.178, 192.168.1.1",
	}
	req := createTestRequest(headers, "10.0.0.1:54321")

	b.ResetTimer()
	for b.Loop() {
		clientip.GetIP(req)
	}
}

func TestPerformanceRequirement(t *testing.T) {
	t.Parallel()
	headers := map[string]string{
		"CF-Connecting-IP": "203.0.113.195",
		"DO-Connecting-IP": "198.51.100.178",
		"X-Forwarded-For":  "192.168.1.1, 10.0.0.1",
		"X-Real-IP":        "172.16.0.1",
	}
	req := createTestRequest(headers, "10.0.0.1:54321")

	start := time.Now()
	for range 1000 {
		clientip.GetIP(req)
	}
	duration := time.Since(start)

	avgDuration := duration / 1000
	if avgDuration > time.Millisecond {
		t.Errorf("Performance requirement not met: average duration %v > 1ms", avgDuration)
	}
}

func createTestRequest(headers map[string]string, remoteAddr string) *http.Request {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = remoteAddr

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req
}
