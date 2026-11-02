package services

import (
	"strings"
	"testing"
)

func TestParseTerminalSize(t *testing.T) {
	for raw, want := range map[string]bool{`{"cols":80,"rows":24}`: true, `{"cols":0,"rows":24}`: false, `{"cols":80,"rows":0}`: false, `{"cols":99999,"rows":24}`: false, `{"cols":80,"rows":99999}`: false, `nope`: false} {
		_, _, got := parseTerminalSize([]byte(raw))
		if got != want {
			t.Errorf("%s: got %v", raw, got)
		}
	}
}

func TestContainerShellCommand(t *testing.T) {
	command := containerShellCommand("app-service-abc123", "feed")
	if !strings.HasPrefix(command, "docker exec -it -e TERM=xterm-256color -e KYMBER_TERMINAL=feed 'app-service-abc123' sh -c ") || !strings.Contains(command, "exec bash") {
		t.Errorf("command = %s", command)
	}
	cleanup := containerShellCleanup("app-service-abc123", "feed")
	if !strings.Contains(cleanup, `tr '\''\0'\'' '\''\n'\''`) {
		t.Errorf("cleanup must split environ on NUL: %s", cleanup)
	}
	if strings.Contains(cleanup, " -e ") || !strings.HasSuffix(cleanup, " sh 'feed'") {
		t.Errorf("cleanup = %s", cleanup)
	}
}

func TestTerminalSlots(t *testing.T) {
	slots := terminalSlots{open: map[uint64]int{}}
	for i := 0; i < maxTerminalsPerUser; i++ {
		if !slots.acquire(1) {
			t.Fatalf("slot %d refused", i)
		}
	}
	if slots.acquire(1) {
		t.Error("limit not enforced")
	}
	if !slots.acquire(2) {
		t.Error("limit must be per user")
	}
	slots.release(1)
	if !slots.acquire(1) {
		t.Error("released slot not reusable")
	}
}
