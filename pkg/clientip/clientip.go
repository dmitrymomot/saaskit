package clientip

import (
	"net"
	"net/http"
	"strings"
)

// GetIP returns the client's IP address from HTTP request.
// See package documentation for header priority details.
func GetIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		if parsed := parseIP(ip); parsed != "" {
			return parsed
		}
	}

	if ip := r.Header.Get("DO-Connecting-IP"); ip != "" {
		if parsed := parseIP(ip); parsed != "" {
			return parsed
		}
	}

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// X-Forwarded-For contains comma-separated IPs; use the leftmost (client origin)
		for ip := range strings.SplitSeq(forwarded, ",") {
			if parsed := parseIP(strings.TrimSpace(ip)); parsed != "" {
				return parsed
			}
		}
	}

	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		if parsed := parseIP(ip); parsed != "" {
			return parsed
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr might be just IP without port in some environments
		if parsed := parseIP(r.RemoteAddr); parsed != "" {
			return parsed
		}
		return r.RemoteAddr
	}
	if parsed := parseIP(host); parsed != "" {
		return parsed
	}
	return host
}

// parseIP validates and normalizes an IP address string.
func parseIP(ipStr string) string {
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" {
		return ""
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}

	// Reject 0.0.0.0 which indicates no valid client IP was provided
	if ip.Equal(net.IPv4zero) {
		return ""
	}

	return ip.String()
}
