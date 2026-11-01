package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"kymber/backend/internal/templates"
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

func TestCheckContainer(t *testing.T) {
	cases := []struct {
		name    string
		state   string
		stable  time.Duration
		ready   bool
		wantErr bool
	}{
		{"healthy", "running 0 healthy", 0, true, false},
		{"unhealthy", "running 0 unhealthy", 0, false, true},
		{"still starting", "running 0 starting", time.Minute, false, false},
		{"no healthcheck, just started", "running 0 ", time.Second, false, false},
		{"no healthcheck, stayed up", "running 0 ", containerStableAfter, true, false},
		{"no healthcheck, restarting", "restarting 3 ", time.Minute, false, false},
	}
	for _, c := range cases {
		ready, err := checkContainer(c.state, c.stable)
		if ready != c.ready || (err != nil) != c.wantErr {
			t.Errorf("%s: ready = %v, err = %v", c.name, ready, err)
		}
	}
}

func TestRunArgsOmitOptionalFlagsForCustomImages(t *testing.T) {
	args := strings.Join(runArgs(task{Name: "app", ServiceID: "abc123"}, templates.Template{}, "net", "/env"), " ")
	for _, flag := range []string{"--health-cmd", "--volume"} {
		if strings.Contains(args, flag) {
			t.Errorf("unexpected %s in %q", flag, args)
		}
	}
	args = strings.Join(runArgs(task{Name: "app", ServiceID: "abc123"}, templates.Template{HealthCmd: "true", DataPath: "/data"}, "net", "/env"), " ")
	for _, flag := range []string{"--health-cmd true", "--volume kymber-service-abc123-data:/data", "--env-file /env"} {
		if !strings.Contains(args, flag) {
			t.Errorf("missing %s in %q", flag, args)
		}
	}
}

func TestNetworkNameIsolatesEnvironments(t *testing.T) {
	outside := networkName(task{WorkspaceID: "aaa111"})
	production := networkName(task{WorkspaceID: "aaa111", Environment: "bbb222"})
	development := networkName(task{WorkspaceID: "aaa111", Environment: "ccc333"})
	if outside != "kymber-workspace-aaa111" {
		t.Errorf("apps outside an environment must keep the existing network, got %q", outside)
	}
	if production == outside || production == development {
		t.Errorf("networks not isolated: %q, %q, %q", outside, production, development)
	}
}
