package services

import "testing"

func TestParseDeployProgress(t *testing.T) {
	cases := []struct {
		raw              string
		done, ok, logged bool
		log              string
	}{
		{"__KYMBER_NO_LOG__", false, false, false, ""},
		{"__KYMBER_RUNNING__", false, false, true, ""},
		{"__KYMBER_EXIT__0\nPulled\n", true, true, true, "Pulled"},
		{"__KYMBER_EXIT__1\nboom\n__KYMBER_EXIT__ in text\n", true, false, true, "boom\n__KYMBER_EXIT__ in text"},
		{"__KYMBER_EXIT__garbage\nx", true, false, true, "x"},
	}
	for _, c := range cases {
		p := parseDeployProgress(c.raw)
		if p.done != c.done || p.succeeded != c.ok || p.logExists != c.logged || p.log != c.log {
			t.Errorf("%q: %+v", c.raw, p)
		}
	}
}
