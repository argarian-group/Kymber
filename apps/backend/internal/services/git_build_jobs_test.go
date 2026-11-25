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

func TestGitBuildConfiguration(t *testing.T) {
	source := gitSource{BaseDirectory: "/apps/web/", DockerfilePath: "", BuildArgs: []string{"NODE_ENV=production", "EMPTY="}}
	if err := validateGitBuildConfiguration(&source); err != nil {
		t.Fatal(err)
	}
	if source.BuildPack != "dockerfile" || source.BaseDirectory != "apps/web" || source.DockerfilePath != "Dockerfile" {
		t.Fatalf("defaults not applied: %+v", source)
	}
	for _, invalid := range []gitSource{
		{BaseDirectory: "../outside"},
		{BaseDirectory: "a/../../b"},
		{DockerfilePath: "docker/../../Dockerfile"},
		{DockerfilePath: "a//Dockerfile"},
		{BuildArgs: []string{"NO_VALUE"}},
		{BuildArgs: []string{"1BAD=x"}},
		{BuildArgs: []string{"MULTI=a\nb"}},
		{BuildPack: "nixpacks", StartCommand: "npm start"},
	} {
		if err := validateGitBuildConfiguration(&invalid); err == nil {
			t.Errorf("invalid configuration accepted: %+v", invalid)
		}
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
