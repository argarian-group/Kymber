package services

import "testing"

func TestParseLogTail(t *testing.T) {
	for raw, want := range map[string]int{"": 300, "1": 1, "5000": 5000, "50000": 50000} {
		if got, ok := parseLogTail(raw); !ok || got != want {
			t.Errorf("%q: got %d %v", raw, got, ok)
		}
	}
	for _, raw := range []string{"0", "-1", "50001", "abc", "10;reboot", "1.5"} {
		if _, ok := parseLogTail(raw); ok {
			t.Errorf("%q accepted", raw)
		}
	}
}
