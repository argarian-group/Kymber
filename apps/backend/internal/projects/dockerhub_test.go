package projects

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInspectImageReadsPortAndVolumeFromLinuxManifest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("scope"); got != "repository:bitnami/app:pull" {
			t.Errorf("scope = %q", got)
		}
		json.NewEncoder(w).Encode(map[string]string{"token": "abc"})
	})
	requireToken := func(r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer abc" {
			t.Errorf("%s sent without token", r.URL.Path)
		}
	}
	mux.HandleFunc("/v2/bitnami/app/manifests/1.0", func(w http.ResponseWriter, r *http.Request) {
		requireToken(r)
		w.Write([]byte(`{"manifests":[{"digest":"sha256:win","platform":{"os":"windows","architecture":"amd64"}},{"digest":"sha256:arm","platform":{"os":"linux","architecture":"arm64"}},{"digest":"sha256:amd","platform":{"os":"linux","architecture":"amd64"}}]}`))
	})
	mux.HandleFunc("/v2/bitnami/app/manifests/sha256:amd", func(w http.ResponseWriter, r *http.Request) {
		requireToken(r)
		w.Write([]byte(`{"config":{"digest":"sha256:cfg"}}`))
	})
	mux.HandleFunc("/v2/bitnami/app/blobs/sha256:cfg", func(w http.ResponseWriter, r *http.Request) {
		requireToken(r)
		w.Write([]byte(`{"config":{"ExposedPorts":{"9000/udp":{},"8080/tcp":{}},"Volumes":{"/var/data":{},"/data":{}}}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	registryAuthURL, registryBaseURL = server.URL+"/token", server.URL
	port, dataPath, err := inspectImage(context.Background(), "bitnami/app", "1.0")
	if err != nil {
		t.Fatal(err)
	}
	if port != 8080 || dataPath != "/data" {
		t.Errorf("port = %d, dataPath = %q", port, dataPath)
	}
}

func TestDockerHubEndpointsRejectBadInput(t *testing.T) {
	h := &Handler{}
	for name, call := range map[string]func(http.ResponseWriter, *http.Request){
		"/search?q=":                    h.searchDockerHub,
		"/image?repository=ghcr.io/x/y": h.dockerImageDetails,
	} {
		recorder := httptest.NewRecorder()
		call(recorder, httptest.NewRequest(http.MethodGet, name, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s = %d", name, recorder.Code)
		}
	}
}

func TestDockerLogoURL(t *testing.T) {
	cases := map[string]string{
		"https://djeqr6to3dedg.cloudfront.net/repo-logos/library/redis/live/logo.png": "https://djeqr6to3dedg.cloudfront.net/repo-logos/library/redis/live/logo.png",
		"https://www.gravatar.com/avatar/abc?s=80&r=g&d=mm":                           "https://www.gravatar.com/avatar/abc?d=404&r=g&s=80",
		"http://djeqr6to3dedg.cloudfront.net/logo.png":                                "",
		"https://evil.example/logo.png":                                               "",
		"javascript:alert(1)":                                                         "",
		"":                                                                            "",
	}
	for input, want := range cases {
		if got := dockerLogoURL(input); got != want {
			t.Errorf("dockerLogoURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSearchDockerHubKeepsOnlyPullableImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search/v4" || r.URL.Query().Get("query") != "post gres" {
			t.Errorf("request = %s", r.URL)
		}
		w.Write([]byte(`{"results":[
			{"type":"image","slug":"postgres","short_description":"DB","badge":"official","star_count":5,"logo_url":{"small":"https://djeqr6to3dedg.cloudfront.net/l.png"}},
			{"type":"dhi","slug":"dhi/postgres","badge":"hardened"},
			{"type":"image","slug":"acme/postgres","badge":"open_source","logo_url":{"small":"https://evil.example/l.png"}}
		]}`))
	}))
	defer server.Close()
	dockerHubBaseURL = server.URL
	recorder := httptest.NewRecorder()
	(&Handler{}).searchDockerHub(recorder, httptest.NewRequest(http.MethodGet, "/search?q=post+gres", nil))
	var response struct {
		Images []dockerHubImage `json:"images"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	want := []dockerHubImage{
		{Repository: "postgres", Description: "DB", Official: true, Stars: 5, LogoURL: "https://djeqr6to3dedg.cloudfront.net/l.png"},
		{Repository: "acme/postgres"},
	}
	if len(response.Images) != 2 || response.Images[0] != want[0] || response.Images[1] != want[1] {
		t.Errorf("images = %+v", response.Images)
	}
}
