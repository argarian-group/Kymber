package services

import (
	"strings"
	"testing"
)

func TestCleanContainerPath(t *testing.T) {
	for raw, ok := range map[string]bool{"/": true, "/etc/app.conf": true, "/a b/c'd": true, "/$(reboot)": true, "": false, "etc": false, "/a/../b": false, "/a/": false, "/a\nb": false, "/a\x00b": false, "/" + strings.Repeat("a", 5000): false} {
		if _, got := cleanContainerPath(raw); got != ok {
			t.Errorf("%q: got %v", raw, got)
		}
	}
}

func TestParseFileEntries(t *testing.T) {
	entries, truncated := parseFileEntries("f 0 12 100 644 /etc/my file.txt\nd 0 4096 200 755 /etc/zeta\nd 1 7 300 777 /etc/Link\nbroken line\no 0 0 400 600 /etc/sock\n", 10)
	if truncated || len(entries) != 4 {
		t.Fatalf("entries = %+v truncated = %v", entries, truncated)
	}
	if entries[0].Name != "Link" || !entries[0].Symlink || entries[0].Type != "directory" || entries[1].Name != "zeta" {
		t.Errorf("directories must come first by name: %+v", entries)
	}
	if entries[2].Name != "my file.txt" || entries[2].Size != 12 || entries[2].Type != "file" || entries[3].Type != "other" {
		t.Errorf("files: %+v", entries)
	}
	if _, truncated := parseFileEntries("f 0 1 1 644 /a\nf 0 1 1 644 /b\n", 1); !truncated {
		t.Error("extra line must report truncation")
	}
}

func TestContainerCommandQuotesArguments(t *testing.T) {
	got := containerCommand("app-service-abc123", true, "echo", "/x'; reboot #")
	want := `docker exec -i 'app-service-abc123' sh -c 'echo' sh '/x'\''; reboot #'`
	if got != want {
		t.Errorf("got %s", got)
	}
	if strings.Contains(containerCommand("n", false, "s"), " -i ") {
		t.Error("stdin must stay closed unless content is sent")
	}
}
