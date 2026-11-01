package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestValidateAPIURL(t *testing.T) {
	cases := map[string]bool{
		"http://127.0.0.1:8080":        true,
		"http://localhost:8080":        true,
		"http://[::1]:8080":            true,
		"http://backend:8080":          true,
		"http://backend.example.com":   false,
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

func TestContainerIDIgnoresPullProgress(t *testing.T) {
	output := "Unable to find image 'redis:latest' locally\nlatest: Pulling from library/redis\nStatus: Downloaded newer image\nabc123"
	if got := containerIDFrom(output); got != "abc123" {
		t.Errorf("containerIDFrom = %q", got)
	}
}

func TestResultFitsAPILimits(t *testing.T) {
	long := strings.Repeat("é", 2000) + "the real error"
	got := fitResult(result{Message: long, ContainerID: strings.Repeat("a", 200)})
	if len(got.Message) > 2048 || !utf8.ValidString(got.Message) || !strings.HasSuffix(got.Message, "the real error") {
		t.Errorf("message not fitted: %d bytes", len(got.Message))
	}
	if len(got.ContainerID) > 128 {
		t.Errorf("container ID not fitted: %d", len(got.ContainerID))
	}
	short := result{Message: "ok", ContainerID: "abc"}
	if fitResult(short) != short {
		t.Errorf("short result changed: %+v", fitResult(short))
	}
}

func TestAgentToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server-token")
	if got := agentToken("fixed", path); got != "fixed" {
		t.Errorf("explicit token = %q", got)
	}
	if got := agentToken("", path); got != "" {
		t.Errorf("missing file = %q", got)
	}
	if err := os.WriteFile(path, []byte("from-file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := agentToken("", path); got != "from-file" {
		t.Errorf("file token = %q", got)
	}
}
