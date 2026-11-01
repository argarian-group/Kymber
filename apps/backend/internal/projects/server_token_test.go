package projects

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func TestTokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server-token")
	if _, ok := tokenFileDigest(path); ok {
		t.Fatal("missing file reported a digest")
	}
	if err := writeTokenFile(path, "first"); err != nil {
		t.Fatal(err)
	}
	digest, ok := tokenFileDigest(path)
	if !ok || digest != sha256.Sum256([]byte("first")) {
		t.Fatalf("digest = %x, ok = %v", digest, ok)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %v, err = %v", info.Mode(), err)
	}
	if err := writeTokenFile(path, "second"); err != nil {
		t.Fatal(err)
	}
	if digest, _ := tokenFileDigest(path); digest != sha256.Sum256([]byte("second")) {
		t.Fatal("token was not replaced")
	}
	if err := os.WriteFile(path, []byte(" \n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, ok := tokenFileDigest(path); ok {
		t.Fatal("blank file reported a digest")
	}
}
