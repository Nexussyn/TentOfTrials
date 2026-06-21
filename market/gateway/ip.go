// Package gateway provides HTTP middleware for the market API gateway.
package gateway

import (
	"net"
	"net/http"
	"strings"
)

// ExtractClientIP derives the client IP from an HTTP request.
// It checks proxy headers (X-Forwarded-For, X-Real-IP) first,
// then falls back to RemoteAddr. Returns empty string if unable
// to determine a valid IP.
func ExtractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (first IP in comma-separated list)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" && isValidIP(ip) {
			return ip
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" && isValidIP(xri) {
		return xri
	}

	// Fall back to RemoteAddr
	return extractIPFromAddr(r.RemoteAddr)
}

// extractIPFromAddr parses an IP from host:port or bare IP format.
func extractIPFromAddr(addr string) string {
	if addr == "" {
		return ""
	}

	// Try host:port format
	host, _, err := net.SplitHostPort(addr)
	if err == nil && isValidIP(host) {
		return host
	}

	// Try bare IP
	if isValidIP(addr) {
		return addr
	}

	return ""
}

// isValidIP checks if a string is a valid IPv4 or IPv6 address.
func isValidIP(s string) bool {
	return net.ParseIP(s) != nil
}
