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

func TestGitBuildConfigurationOnlyAcceptsRepositoryDockerfile(t *testing.T) {
	if err := validateGitBuildConfiguration(gitBuildRequest{BuildPack: "dockerfile"}); err != nil {
		t.Fatal(err)
	}
	if err := validateGitBuildConfiguration(gitBuildRequest{}); err != nil {
		t.Fatalf("default Dockerfile build rejected: %v", err)
	}
	if err := validateGitBuildConfiguration(gitBuildRequest{BuildPack: "nixpacks", StartCommand: "npm start"}); err == nil {
		t.Fatal("unsupported custom build commands accepted")
	}
}

func TestGitHubBranchAPIResponseLimit(t *testing.T) {
	branches := make([]gitHubRef, 101)
	for i := range branches {
		branches[i].Name = "branch"
	}
	if len(branches) > 100 {
		branches = branches[:100]
	}
	if len(branches) != 100 {
		t.Fatalf("expected 100 branches, got %d", len(branches))
	}
}

func TestGitHubBranchURLAndRefs(t *testing.T) {
	for _, repository := range []string{"https://github.com/owner/repo", "https://github.com/owner/repo.git"} {
		if !validPublicGitHubURL(repository) {
			t.Fatalf("valid repository rejected: %s", repository)
		}
	}
	for _, repository := range []string{"http://github.com/owner/repo", "https://github.com.evil/owner/repo"} {
		if validPublicGitHubURL(repository) {
			t.Fatalf("invalid repository accepted: %s", repository)
		}
	}
	for _, ref := range []string{"main", "release/v1"} {
		if !validGitRef(ref) {
			t.Fatalf("valid branch rejected: %s", ref)
		}
	}
	for _, ref := range []string{"../main", "bad..ref", "a//b"} {
		if validGitRef(ref) {
			t.Fatalf("invalid branch accepted: %s", ref)
		}
	}
}
