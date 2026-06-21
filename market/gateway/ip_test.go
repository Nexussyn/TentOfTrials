package gateway

import (
	"net/http"
	"testing"
)

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{"host_port_ipv4", "192.168.1.100:45678", nil, "192.168.1.100"},
		{"host_port_ipv6", "[2001:db8::1]:8080", nil, "2001:db8::1"},
		{"bare_ipv4", "10.0.0.1", nil, "10.0.0.1"},
		{"bare_ipv6", "::1", nil, "::1"},
		{"empty_remote_addr", "", nil, ""},
		{"malformed_garbage", "not-an-ip", nil, ""},
		{"malformed_partial", "192.168.1", nil, ""},
		{"malformed_port_only", ":8080", nil, ""},
		{"xff_single", "10.0.0.1:8080", map[string]string{"X-Forwarded-For": "203.0.113.50"}, "203.0.113.50"},
		{"xff_multiple", "10.0.0.1:8080", map[string]string{"X-Forwarded-For": "203.0.113.50, 10.1.1.1"}, "203.0.113.50"},
		{"xff_invalid", "10.0.0.1:8080", map[string]string{"X-Forwarded-For": "invalid"}, "10.0.0.1"},
		{"xri_valid", "10.0.0.1:8080", map[string]string{"X-Real-IP": "203.0.113.100"}, "203.0.113.100"},
		{"xri_invalid", "10.0.0.1:8080", map[string]string{"X-Real-IP": "bad"}, "10.0.0.1"},
		{"xff_precedence", "10.0.0.1:8080", map[string]string{"X-Forwarded-For": "203.0.113.50", "X-Real-IP": "203.0.113.100"}, "203.0.113.50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				RemoteAddr: tt.remoteAddr,
				Header:     make(http.Header),
			}
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			if got := ExtractClientIP(req); got != tt.want {
				t.Errorf("ExtractClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
