package services

import "testing"

func TestGitBuildSourceValidation(t *testing.T) {
	for _, raw := range []string{"https://github.com/kymber/app", "https://github.com/kymber/app.git"} {
		if !validPublicGitHubURL(raw) {
			t.Errorf("valid GitHub repository rejected: %q", raw)
		}
	}
	for _, raw := range []string{"http://github.com/a/b", "https://github.com.evil/a/b", "https://user@github.com/a/b", "https://github.com/a/b?x=1", "https://127.0.0.1/a/b"} {
		if validPublicGitHubURL(raw) {
			t.Errorf("unsafe GitHub repository accepted: %q", raw)
		}
	}
	for _, ref := range []string{"main", "release/v1.2"} {
		if !validGitRef(ref) {
			t.Errorf("valid Git ref rejected: %q", ref)
		}
	}
	for _, ref := range []string{"../main", "main..x", "a/.hidden", "a/file.lock", "a//b"} {
		if validGitRef(ref) {
			t.Errorf("invalid Git ref accepted: %q", ref)
		}
	}
}
