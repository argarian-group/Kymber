package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name, forwarded string
		trust           bool
		want            string
	}{
		{"direct", "", false, "10.0.0.5"},
		{"ignores header when untrusted", "203.0.113.9", false, "10.0.0.5"},
		{"proxy", "203.0.113.9", true, "203.0.113.9"},
		{"spoofed left entries ignored", "1.1.1.1, 203.0.113.9", true, "203.0.113.9"},
		{"ipv6", "2001:db8::1", true, "2001:db8::1"},
		{"garbage falls back", "not-an-ip", true, "10.0.0.5"},
		{"trusted without header", "", true, "10.0.0.5"},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "10.0.0.5:1234"
		if c.forwarded != "" {
			r.Header.Set("X-Forwarded-For", c.forwarded)
		}
		if got := clientIP(r, c.trust); got != c.want {
			t.Errorf("%s: got %s", c.name, got)
		}
	}
}
