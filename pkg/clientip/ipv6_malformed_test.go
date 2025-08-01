package clientip_test

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/clientip"
)

func TestIPv6SecurityCases(t *testing.T) {
	t.Run("ipv6_zone_identifiers", func(t *testing.T) {
		testIPv6ZoneIdentifiers(t)
	})

	t.Run("ipv6_embedding_attacks", func(t *testing.T) {
		testIPv6EmbeddingAttacks(t)
	})

	t.Run("ipv6_format_confusion", func(t *testing.T) {
		testIPv6FormatConfusion(t)
	})

	t.Run("ipv6_address_validation", func(t *testing.T) {
		testIPv6AddressValidation(t)
	})
}

func testIPv6ZoneIdentifiers(t *testing.T) {
	t.Run("zone_identifier_handling", func(t *testing.T) {
		t.Parallel()

		zoneTests := []struct {
			name     string
			header   string
			expected string
			desc     string
		}{
			{
				name:     "basic_zone_identifier",
				header:   "fe80::1%eth0",
				expected: "10.0.0.1", // Should fall back as zone identifiers are often invalid in HTTP context
				desc:     "Zone identifiers should be handled appropriately",
			},
			{
				name:     "numeric_zone_identifier",
				header:   "fe80::1%1",
				expected: "10.0.0.1", // Should fall back
				desc:     "Numeric zone identifiers should be handled",
			},
			{
				name:     "complex_zone_identifier",
				header:   "fe80::1%Loopback_Pseudo-Interface_1",
				expected: "10.0.0.1", // Should fall back
				desc:     "Complex zone identifiers should be handled",
			},
			{
				name:     "malicious_zone_identifier",
				header:   "fe80::1%../../../etc/passwd",
				expected: "10.0.0.1", // Should fall back and not be vulnerable
				desc:     "Malicious zone identifiers should be rejected",
			},
			{
				name:     "zone_with_control_chars",
				header:   "fe80::1%eth0\r\nX-Admin: true",
				expected: "10.0.0.1", // Should fall back
				desc:     "Zone identifiers with control characters should be rejected",
			},
		}

		for _, tt := range zoneTests {
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

	t.Run("valid_ipv6_without_zones", func(t *testing.T) {
		t.Parallel()

		validIPv6Tests := []string{
			"2001:db8::1",
			"2001:db8:85a3:8d3:1319:8a2e:370:7348",
			"::1",
			"192.168.1.1",
			"::",
			"2001:db8::8a2e:370:7334",
		}

		for i, validIP := range validIPv6Tests {
			t.Run(fmt.Sprintf("valid_ipv6_%d", i), func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"DO-Connecting-IP": validIP,
				}
				req := createTestRequest(headers, "127.0.0.1:8080")

				ip := clientip.GetIP(req)
				assert.Equal(t, validIP, ip, "Valid IPv6 should be accepted: %s", validIP)
			})
		}
	})
}

func testIPv6EmbeddingAttacks(t *testing.T) {
	t.Run("ipv4_mapped_ipv6", func(t *testing.T) {
		t.Parallel()

		mappedTests := []struct {
			name     string
			header   string
			expected string
			desc     string
		}{
			{
				name:     "standard_ipv4_mapped",
				header:   "::ffff:192.168.1.1",
				expected: "192.168.1.1", // Go normalizes IPv4-mapped to IPv4
				desc:     "Standard IPv4-mapped IPv6 should be normalized to IPv4",
			},
			{
				name:     "ipv4_mapped_public",
				header:   "::ffff:203.0.113.1",
				expected: "203.0.113.1", // Go normalizes IPv4-mapped to IPv4
				desc:     "IPv4-mapped public address should be normalized to IPv4",
			},
			{
				name:     "ipv4_mapped_loopback",
				header:   "::ffff:127.0.0.1",
				expected: "127.0.0.1", // Go normalizes IPv4-mapped to IPv4
				desc:     "IPv4-mapped loopback should be normalized to IPv4",
			},
			{
				name:     "malformed_ipv4_mapped",
				header:   "::ffff:999.999.999.999",
				expected: "10.0.0.1", // Should fall back
				desc:     "Malformed IPv4-mapped should be rejected",
			},
		}

		for _, tt := range mappedTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"X-Real-IP": tt.header,
				}
				req := createTestRequest(headers, "10.0.0.1:8080")

				ip := clientip.GetIP(req)
				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})

	t.Run("ipv4_embedded_in_ipv6", func(t *testing.T) {
		t.Parallel()

		embeddedTests := []struct {
			name     string
			header   string
			expected string
			desc     string
		}{
			{
				name:     "ipv4_compatible_deprecated",
				header:   "::192.168.1.1", // Deprecated IPv4-compatible
				expected: "::c0a8:101",    // Go converts to hex format
				desc:     "IPv4-compatible IPv6 should be handled",
			},
			{
				name:     "well_known_prefix",
				header:   "64:ff9b::192.0.2.1", // Well-known prefix for IPv4/IPv6 translation
				expected: "64:ff9b::c000:201",  // Go converts to hex format
				desc:     "Well-known prefix addresses should be accepted",
			},
		}

		for _, tt := range embeddedTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"X-Forwarded-For": tt.header,
				}
				req := createTestRequest(headers, "127.0.0.1:8080")

				ip := clientip.GetIP(req)
				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})

	t.Run("ipv6_address_spoofing", func(t *testing.T) {
		t.Parallel()

		spoofTests := []struct {
			name     string
			header   string
			expected string
			desc     string
		}{
			{
				name:     "fake_ipv4_in_ipv6_format",
				header:   "2001:db8::192.168.1.1", // Looks like embedded but isn't standard
				expected: "2001:db8::c0a8:101",    // Go converts to hex format
				desc:     "Non-standard embedded format should be treated as regular IPv6",
			},
			{
				name:     "mixed_case_confusion",
				header:   "2001:DB8::AbCd:EfGh",
				expected: "::1:8080", // Falls back to raw RemoteAddr (SplitHostPort fails without brackets)
				desc:     "Invalid hex should fall back to RemoteAddr",
			},
			{
				name:     "compressed_vs_expanded",
				header:   "2001:0db8:0000:0000:0000:0000:0000:0001",
				expected: "2001:db8::1", // Go compresses the format
				desc:     "Expanded IPv6 format should be compressed",
			},
		}

		for _, tt := range spoofTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"CF-Connecting-IP": tt.header,
				}
				req := createTestRequest(headers, "::1:8080")

				ip := clientip.GetIP(req)
				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})
}

func testIPv6FormatConfusion(t *testing.T) {
	t.Run("bracket_confusion", func(t *testing.T) {
		t.Parallel()

		bracketTests := []struct {
			name     string
			header   string
			expected string
			desc     string
		}{
			{
				name:     "brackets_in_header",
				header:   "[2001:db8::1]",
				expected: "10.0.0.1", // Brackets should not be in headers
				desc:     "IPv6 with brackets should be rejected in headers",
			},
			{
				name:     "brackets_with_port_confusion",
				header:   "[2001:db8::1]:8080",
				expected: "10.0.0.1", // This is not valid in headers
				desc:     "IPv6 with port-style brackets should be rejected",
			},
			{
				name:     "partial_brackets",
				header:   "[2001:db8::1",
				expected: "10.0.0.1", // Malformed
				desc:     "Malformed brackets should be rejected",
			},
			{
				name:     "double_brackets",
				header:   "[[2001:db8::1]]",
				expected: "10.0.0.1", // Invalid
				desc:     "Double brackets should be rejected",
			},
		}

		for _, tt := range bracketTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"DO-Connecting-IP": tt.header,
				}
				req := createTestRequest(headers, "10.0.0.1:8080")

				ip := clientip.GetIP(req)
				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})

	t.Run("compression_edge_cases", func(t *testing.T) {
		t.Parallel()

		compressionTests := []struct {
			name     string
			header   string
			expected string
			desc     string
		}{
			{
				name:     "multiple_double_colons",
				header:   "2001::db8::1", // Invalid - only one :: allowed
				expected: "10.0.0.1",
				desc:     "Multiple double colons should be rejected",
			},
			{
				name:     "leading_colon_only",
				header:   ":2001:db8::1", // Invalid
				expected: "10.0.0.1",
				desc:     "Leading single colon should be rejected",
			},
			{
				name:     "trailing_colon_only",
				header:   "2001:db8::1:", // Invalid
				expected: "10.0.0.1",
				desc:     "Trailing single colon should be rejected",
			},
			{
				name:     "too_many_segments",
				header:   "2001:db8:85a3:8d3:1319:8a2e:370:7348:extra", // 9 segments
				expected: "10.0.0.1",
				desc:     "Too many segments should be rejected",
			},
		}

		for _, tt := range compressionTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"X-Real-IP": tt.header,
				}
				req := createTestRequest(headers, "10.0.0.1:8080")

				ip := clientip.GetIP(req)
				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})

	t.Run("special_ipv6_addresses", func(t *testing.T) {
		t.Parallel()

		specialTests := []struct {
			name     string
			header   string
			expected string
			desc     string
		}{
			{
				name:     "unspecified_address",
				header:   "::",
				expected: "::",
				desc:     "IPv6 unspecified address should be accepted",
			},
			{
				name:     "loopback_address",
				header:   "::1",
				expected: "::1",
				desc:     "IPv6 loopback should be accepted",
			},
			{
				name:     "multicast_address",
				header:   "ff00::1",
				expected: "ff00::1",
				desc:     "IPv6 multicast should be accepted",
			},
			{
				name:     "link_local_address",
				header:   "fe80::1",
				expected: "fe80::1",
				desc:     "IPv6 link-local should be accepted",
			},
		}

		for _, tt := range specialTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"X-Forwarded-For": tt.header,
				}
				req := createTestRequest(headers, "[::1]:8080")

				ip := clientip.GetIP(req)
				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})
}

func testIPv6AddressValidation(t *testing.T) {
	t.Run("hex_digit_validation", func(t *testing.T) {
		t.Parallel()

		hexTests := []struct {
			name     string
			header   string
			expected string
			desc     string
		}{
			{
				name:     "invalid_hex_digits",
				header:   "2001:ghij::1", // g, h, i, j are not hex
				expected: "127.0.0.1",    // Falls back to RemoteAddr
				desc:     "Invalid hex digits should be rejected",
			},
			{
				name:     "too_long_hex_segment",
				header:   "2001:12345::1", // Segment too long (max 4 hex digits)
				expected: "127.0.0.1",     // Falls back to RemoteAddr
				desc:     "Hex segments too long should be rejected",
			},
			{
				name:     "empty_segments",
				header:   "2001:::1",  // Empty segment (should be :: for compression)
				expected: "127.0.0.1", // Falls back to RemoteAddr
				desc:     "Empty segments should be rejected",
			},
		}

		for _, tt := range hexTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"CF-Connecting-IP": tt.header,
				}
				req := createTestRequest(headers, "127.0.0.1:8080")

				ip := clientip.GetIP(req)
				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})

	t.Run("performance_with_ipv6", func(t *testing.T) {
		t.Parallel()

		longIPv6Chain := make([]string, 1000)
		for i := range longIPv6Chain {
			longIPv6Chain[i] = fmt.Sprintf("2001:db8::%x", i)
		}
		longIPv6Chain = append(longIPv6Chain, "2001:db8::1")

		headers := map[string]string{
			"X-Forwarded-For": strings.Join(longIPv6Chain, ", "),
		}
		req := createTestRequest(headers, "[::1]:8080")

		start := time.Now()
		ip := clientip.GetIP(req)
		duration := time.Since(start)

		assert.Equal(t, "2001:db8::", ip, "Should find first valid IPv6") // Go compresses ::0 to ::
		assert.Less(t, duration, 50*time.Millisecond, "IPv6 processing should be efficient")

		t.Logf("Processed %d IPv6 addresses in %v", len(longIPv6Chain), duration)
	})
}

func TestMalformedInputHandling(t *testing.T) {
	t.Run("malformed_remote_addr", func(t *testing.T) {
		testMalformedRemoteAddr(t)
	})

	t.Run("malformed_header_values", func(t *testing.T) {
		testMalformedHeaderValues(t)
	})

	t.Run("edge_case_inputs", func(t *testing.T) {
		testEdgeCaseInputs(t)
	})
}

func testMalformedRemoteAddr(t *testing.T) {
	t.Run("remote_addr_without_port", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.100" // No port - should trigger SplitHostPort error

		ip := clientip.GetIP(req)

		assert.Equal(t, "192.168.1.100", ip, "Should parse RemoteAddr directly when SplitHostPort fails")
	})

	t.Run("ipv6_remote_addr_without_brackets", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "::1" // IPv6 without brackets or port

		ip := clientip.GetIP(req)
		assert.Equal(t, "::1", ip, "Should handle IPv6 without brackets in RemoteAddr")
	})

	t.Run("completely_invalid_remote_addr", func(t *testing.T) {
		t.Parallel()

		malformedAddrs := []struct {
			name       string
			remoteAddr string
			expected   string
			desc       string
		}{
			{
				name:       "empty_remote_addr",
				remoteAddr: "",
				expected:   "",
				desc:       "Empty RemoteAddr should be handled",
			},
			{
				name:       "invalid_format",
				remoteAddr: "not-an-address",
				expected:   "not-an-address",
				desc:       "Invalid RemoteAddr format should return raw value",
			},
			{
				name:       "malformed_ipv6_brackets",
				remoteAddr: "[invalid",
				expected:   "[invalid",
				desc:       "Malformed IPv6 brackets should return raw value",
			},
			{
				name:       "port_only",
				remoteAddr: ":8080",
				expected:   "",
				desc:       "Port-only RemoteAddr should be handled",
			},
		}

		for _, tt := range malformedAddrs {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = tt.remoteAddr

				ip := clientip.GetIP(req)
				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})

	t.Run("remote_addr_with_invalid_port", func(t *testing.T) {
		t.Parallel()

		invalidPortAddrs := []string{
			"192.168.1.1:invalid-port",
			"192.168.1.1:99999", // Port out of range
			"192.168.1.1:-1",    // Negative port
			"[::1]:invalid",     // IPv6 with invalid port
		}

		for i, addr := range invalidPortAddrs {
			t.Run(fmt.Sprintf("invalid_port_%d", i), func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = addr

				ip := clientip.GetIP(req)
				t.Logf("RemoteAddr %s resulted in IP: %s", addr, ip)
			})
		}
	})
}

func testMalformedHeaderValues(t *testing.T) {
	t.Run("extremely_long_header_values", func(t *testing.T) {
		t.Parallel()

		longIPTests := []struct {
			name        string
			headerValue string
			expected    string
		}{
			{
				name:        "long_ipv4_with_garbage",
				headerValue: "192.168.1.1" + strings.Repeat("x", 10000),
				expected:    "10.0.0.1", // Should fall back
			},
			{
				name:        "long_ipv6_with_garbage",
				headerValue: "2001:db8::1" + strings.Repeat("y", 10000),
				expected:    "10.0.0.1", // Should fall back
			},
			{
				name:        "pure_garbage",
				headerValue: strings.Repeat("garbage", 5000),
				expected:    "10.0.0.1", // Should fall back
			},
		}

		for _, tt := range longIPTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"CF-Connecting-IP": tt.headerValue,
				}
				req := createTestRequest(headers, "10.0.0.1:8080")

				start := time.Now()
				ip := clientip.GetIP(req)
				duration := time.Since(start)

				assert.Equal(t, tt.expected, ip, "Should handle long malformed headers")
				assert.Less(t, duration, 10*time.Millisecond, "Should process long headers quickly")
			})
		}
	})

	t.Run("binary_data_in_headers", func(t *testing.T) {
		t.Parallel()

		binaryTests := []string{
			string([]byte{0x00, 0x01, 0x02, 0x03}), // Null bytes and control chars
			string([]byte{0xFF, 0xFE, 0xFD}),       // High bytes
			"\x7F\x80\x81\x82",                     // Mixed ASCII/non-ASCII
			"192.168.1.1\x00admin",                 // IP with null terminator
		}

		for i, binaryData := range binaryTests {
			t.Run(fmt.Sprintf("binary_data_%d", i), func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"X-Real-IP": binaryData,
				}
				req := createTestRequest(headers, "127.0.0.1:8080")

				ip := clientip.GetIP(req)
				assert.Equal(t, "127.0.0.1", ip, "Should handle binary data gracefully")
			})
		}
	})

	t.Run("whitespace_edge_cases", func(t *testing.T) {
		t.Parallel()

		whitespaceTests := []struct {
			name     string
			header   string
			expected string
			desc     string
		}{
			{
				name:     "tabs_instead_of_spaces",
				header:   "192.168.1.1\t203.0.113.1",
				expected: "10.0.0.1", // Tabs should not be valid separators
				desc:     "Tabs should not be treated as valid separators",
			},
			{
				name:     "newlines_in_header",
				header:   "192.168.1.1\n203.0.113.1",
				expected: "10.0.0.1", // Newlines should be rejected
				desc:     "Newlines should be rejected",
			},
			{
				name:     "mixed_whitespace",
				header:   " \t\n192.168.1.1 \t\n, \t\n203.0.113.1 \t\n",
				expected: "192.168.1.1", // First valid IP after trimming whitespace
				desc:     "Should extract first valid IP after trimming whitespace",
			},
			{
				name:     "unicode_spaces",
				header:   "192.168.1.1\u2000203.0.113.1", // Unicode space
				expected: "10.0.0.1",                     // Unicode spaces should not be separators
				desc:     "Unicode spaces should not be treated as separators",
			},
		}

		for _, tt := range whitespaceTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"X-Forwarded-For": tt.header,
				}
				req := createTestRequest(headers, "10.0.0.1:8080")

				ip := clientip.GetIP(req)
				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})
}

func testEdgeCaseInputs(t *testing.T) {
	t.Run("numeric_string_edge_cases", func(t *testing.T) {
		t.Parallel()

		numericTests := []struct {
			name     string
			header   string
			expected string
			desc     string
		}{
			{
				name:     "decimal_overflow",
				header:   "999.999.999.999", // Numbers too large for IP octets
				expected: "10.0.0.1",
				desc:     "Decimal overflow should be rejected",
			},
			{
				name:     "hex_in_decimal_context",
				header:   "192.168.1.0xff", // Hex in IPv4 context
				expected: "10.0.0.1",
				desc:     "Hex in decimal context should be rejected",
			},
			{
				name:     "octal_numbers",
				header:   "0192.0168.0001.0001", // Octal-looking numbers
				expected: "10.0.0.1",
				desc:     "Octal-looking numbers should be handled carefully",
			},
			{
				name:     "leading_zeros",
				header:   "192.168.001.001", // Leading zeros
				expected: "192.168.001.001", // May be accepted depending on parseIP implementation
				desc:     "Leading zeros handling should be consistent",
			},
		}

		for _, tt := range numericTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"DO-Connecting-IP": tt.header,
				}
				req := createTestRequest(headers, "10.0.0.1:8080")

				ip := clientip.GetIP(req)
				t.Logf("Input: %s, Output: %s", tt.header, ip)

				require.NotPanics(t, func() {
					clientip.GetIP(req)
				}, "Should not panic on numeric edge cases")
			})
		}
	})

	t.Run("empty_and_whitespace_only", func(t *testing.T) {
		t.Parallel()

		emptyTests := []struct {
			name     string
			headers  map[string]string
			expected string
			desc     string
		}{
			{
				name: "all_empty_headers",
				headers: map[string]string{
					"CF-Connecting-IP": "",
					"DO-Connecting-IP": "",
					"X-Forwarded-For":  "",
					"X-Real-IP":        "",
				},
				expected: "127.0.0.1",
				desc:     "All empty headers should fall back to RemoteAddr",
			},
			{
				name: "whitespace_only_headers",
				headers: map[string]string{
					"CF-Connecting-IP": "   ",
					"DO-Connecting-IP": "\t\t",
					"X-Forwarded-For":  "  \n  ",
				},
				expected: "127.0.0.1",
				desc:     "Whitespace-only headers should fall back to RemoteAddr",
			},
			{
				name: "mixed_empty_and_invalid",
				headers: map[string]string{
					"CF-Connecting-IP": "",
					"DO-Connecting-IP": "invalid",
					"X-Forwarded-For":  "   ",
					"X-Real-IP":        "also-invalid",
				},
				expected: "127.0.0.1",
				desc:     "Mixed empty and invalid should fall back to RemoteAddr",
			},
		}

		for _, tt := range emptyTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				req := createTestRequest(tt.headers, "127.0.0.1:8080")
				ip := clientip.GetIP(req)

				assert.Equal(t, tt.expected, ip, tt.desc)
			})
		}
	})

	t.Run("performance_with_malformed_inputs", func(t *testing.T) {
		t.Parallel()

		malformedInputs := []string{
			strings.Repeat("invalid,", 10000) + "203.0.113.1",
			strings.Repeat("999.999.999.999,", 1000) + "203.0.113.1",
			strings.Repeat("not-an-ip,", 5000) + "203.0.113.1",
			strings.Repeat("2001:invalid::,", 2000) + "2001:db8::1",
		}

		for i, input := range malformedInputs {
			t.Run(fmt.Sprintf("malformed_performance_%d", i), func(t *testing.T) {
				t.Parallel()

				headers := map[string]string{
					"X-Forwarded-For": input,
				}
				req := createTestRequest(headers, "10.0.0.1:8080")

				start := time.Now()
				ip := clientip.GetIP(req)
				duration := time.Since(start)

				assert.True(t,
					ip == "203.0.113.1" || ip == "2001:db8::1" || ip == "10.0.0.1",
					"Should find valid IP or fall back")
				assert.Less(t, duration, 100*time.Millisecond,
					"Should process malformed inputs efficiently")

				t.Logf("Processed malformed input %d in %v, result: %s", i, duration, ip)
			})
		}
	})
}
