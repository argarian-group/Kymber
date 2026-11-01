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
