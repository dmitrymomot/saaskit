package clientip

import (
	"net"
	"net/http"
	"strings"
)

// GetIP returns the client's IP address from HTTP request.
// Priority order optimized for DigitalOcean App Platform:
// 1. CF-Connecting-IP (Cloudflare → DigitalOcean Apps)
// 2. DO-Connecting-IP (DigitalOcean App Platform primary header)
// 3. X-Forwarded-For (Standard proxy header, parse first IP)
// 4. X-Real-IP (Nginx reverse proxy)
// 5. RemoteAddr (Direct connection fallback)
func GetIP(r *http.Request) string {
	// Check Cloudflare header first (highest priority for CF+DO deployments)
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		if parsed := parseIP(ip); parsed != "" {
			return parsed
		}
	}

	// Check DigitalOcean App Platform header (DO-specific primary header)
	if ip := r.Header.Get("DO-Connecting-IP"); ip != "" {
		if parsed := parseIP(ip); parsed != "" {
			return parsed
		}
	}

	// Check standard forwarded header
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, find the first valid one
		// Use SplitSeq for better efficiency (Go 1.24+)
		for ip := range strings.SplitSeq(forwarded, ",") {
			if parsed := parseIP(strings.TrimSpace(ip)); parsed != "" {
				return parsed
			}
		}
	}

	// Check Nginx real IP header
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		if parsed := parseIP(ip); parsed != "" {
			return parsed
		}
	}

	// Fallback to remote address
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, assume it's already just an IP
		return parseIP(r.RemoteAddr)
	}
	return parseIP(host)
}

// parseIP validates and normalizes an IP address string.
// Returns empty string if the IP is invalid.
func parseIP(ipStr string) string {
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" {
		return ""
	}

	// Parse and validate the IP
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}

	// Return the normalized string representation
	return ip.String()
}
