package services

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestValidateStorage(t *testing.T) {
	cases := []struct {
		name, kind, source, mount, dataPath string
		root, ok                            bool
	}{
		{"volume", "volume", "", "/cache", "/data", false, true},
		{"file", "file", "", "/etc/app.conf", "", false, true},
		{"bind as root", "bind", "/srv/app", "/app", "", true, true},
		{"bind needs root", "bind", "/srv/app", "/app", "", false, false},
		{"bind needs source", "bind", "", "/app", "", true, false},
		{"bind relative source", "bind", "srv", "/app", "", true, false},
		{"bind colon injection", "bind", "/srv:/etc", "/app", "", true, false},
		{"bind host root", "bind", "/", "/app", "", true, false},
		{"unknown kind", "tmpfs", "", "/x", "", true, false},
		{"empty mount", "volume", "", "", "", false, false},
		{"mount root", "volume", "", "/", "", false, false},
		{"unclean mount", "volume", "", "/a/../b", "", false, false},
		{"mount colon", "volume", "", "/a:ro", "", false, false},
		{"primary volume clash", "volume", "", "/data", "/data", false, false},
	}
	for _, c := range cases {
		if err := validateStorage(c.kind, c.source, c.mount, c.dataPath, c.root); (err == nil) != c.ok {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
}

func TestStorageDeployCommand(t *testing.T) {
	prepare, args := storageDeployCommand("abc123", []serviceStorage{
		{ID: "aaaaaa", Kind: "volume", MountPath: "/cache"},
		{ID: "bbbbbb", Kind: "bind", Source: "/srv/it's", MountPath: "/app"},
		{ID: "cccccc", Kind: "file", MountPath: "/etc/app.conf", content: []byte("a='$(reboot)'\n")},
	})
	for _, want := range []string{" --volume 'kymber-service-abc123-aaaaaa:/cache'", ` --volume '/srv/it'\''s:/app'`, ` --volume "$HOME/.kymber/service-files/abc123/cccccc/content"':/etc/app.conf'`} {
		if !strings.Contains(args, want) {
			t.Errorf("args %q lack %q", args, want)
		}
	}
	// Base64 keeps file content from ever reaching the shell as code.
	if strings.Contains(prepare, "reboot") || !strings.Contains(prepare, base64.StdEncoding.EncodeToString([]byte("a='$(reboot)'\n"))) {
		t.Errorf("prepare = %q", prepare)
	}
	if prepare, args := storageDeployCommand("abc123", nil); prepare != "" || args != "" {
		t.Errorf("no storages: %q %q", prepare, args)
	}
}

func TestStorageDeleteCommand(t *testing.T) {
	command := storageDeleteCommand("abc123")
	if !strings.Contains(command, "name=^kymber-service-abc123-[0-9a-f]{6}$") || !strings.Contains(command, ".kymber/service-files/abc123") {
		t.Errorf("command = %q", command)
	}
}
