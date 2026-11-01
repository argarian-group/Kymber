package main

import "testing"

func TestValidateAPIURL(t *testing.T) {
	cases := map[string]bool{
		"http://127.0.0.1:8080":        true,
		"http://localhost:8080":        true,
		"http://[::1]:8080":            true,
		"https://kymber.example.com":   true,
		"https://10.0.0.5:8443":        true,
		"http://kymber.example.com":    false,
		"http://10.0.0.5:8080":         false,
		"http://localhost.example.com": false,
		"ftp://localhost":              false,
		"https://user:pw@example.com":  false,
		"https://example.com?a=1":      false,
		"https://":                     false,
		"not a url":                    false,
	}
	for raw, valid := range cases {
		if err := validateAPIURL(raw); (err == nil) != valid {
			t.Errorf("validateAPIURL(%q) = %v, want valid=%v", raw, err, valid)
		}
	}
}
