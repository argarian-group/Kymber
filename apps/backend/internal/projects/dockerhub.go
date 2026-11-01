package projects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"kymber/backend/internal/templates"
)

// Variables so tests can point them at a local server.
var (
	dockerHubBaseURL = "https://hub.docker.com"
	registryAuthURL  = "https://auth.docker.io/token"
	registryBaseURL  = "https://registry-1.docker.io"
)

var dockerHubClient = &http.Client{Timeout: 15 * time.Second}

type dockerHubImage struct {
	Repository  string `json:"repository"`
	Description string `json:"description"`
	Official    bool   `json:"official"`
	Stars       int    `json:"stars"`
	LogoURL     string `json:"logo_url,omitempty"`
}

// dockerLogoURL keeps only Docker Hub's logo hosts, because browsers of every workspace member load this URL.
func dockerLogoURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" {
		return ""
	}
	switch parsed.Host {
	case "djeqr6to3dedg.cloudfront.net":
		return raw
	case "www.gravatar.com":
		// Without this, accounts with no avatar return a grey silhouette instead of failing over to our icon.
		query := parsed.Query()
		query.Set("d", "404")
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}
	return ""
}

func (h *Handler) searchDockerHub(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" || len(query) > 100 {
		writeError(w, http.StatusBadRequest, "search text is required")
		return
	}
	var response struct {
		Results []struct {
			Type             string `json:"type"`
			Slug             string `json:"slug"`
			ShortDescription string `json:"short_description"`
			Badge            string `json:"badge"`
			StarCount        int    `json:"star_count"`
			LogoURL          struct {
				Small string `json:"small"`
			} `json:"logo_url"`
		} `json:"results"`
	}
	target := dockerHubBaseURL + "/api/search/v4?size=15&type=image&query=" + url.QueryEscape(query)
	if err := fetchJSON(r.Context(), target, "", &response); err != nil {
		writeError(w, http.StatusBadGateway, "could not search Docker Hub")
		return
	}
	images := make([]dockerHubImage, 0, len(response.Results))
	for _, result := range response.Results {
		// Other types, such as hardened images, ignore the type filter and cannot be pulled from docker.io.
		if result.Type != "image" {
			continue
		}
		images = append(images, dockerHubImage{Repository: result.Slug, Description: result.ShortDescription, Official: result.Badge == "official", Stars: result.StarCount, LogoURL: dockerLogoURL(result.LogoURL.Small)})
	}
	writeJSON(w, http.StatusOK, map[string][]dockerHubImage{"images": images})
}

func (h *Handler) dockerImageDetails(w http.ResponseWriter, r *http.Request) {
	repository, err := templates.ParseRepository(strings.TrimSpace(r.URL.Query().Get("repository")))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var response struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}
	target := fmt.Sprintf("%s/v2/repositories/%s/tags?page_size=25&ordering=last_updated", dockerHubBaseURL, repository)
	if err := fetchJSON(r.Context(), target, "", &response); err != nil {
		writeError(w, http.StatusBadGateway, "could not read this image from Docker Hub")
		return
	}
	if len(response.Results) == 0 {
		writeError(w, http.StatusNotFound, "this image has no tags")
		return
	}
	tags := make([]string, 0, len(response.Results))
	version := response.Results[0].Name
	for _, tag := range response.Results {
		tags = append(tags, tag.Name)
		if tag.Name == "latest" {
			version = "latest"
		}
	}
	// The user can fill port and data path by hand, so a registry failure must not block them.
	port, dataPath, _ := inspectImage(r.Context(), repository, version)
	writeJSON(w, http.StatusOK, map[string]any{"tags": tags, "default_version": version, "port": port, "data_path": dataPath})
}

// inspectImage reads what the image declares about itself.
// shortcut: reads only the lowest TCP port and the first volume, store a list if multi-volume images become common.
func inspectImage(ctx context.Context, repository, tag string) (int, string, error) {
	var auth struct {
		Token string `json:"token"`
	}
	target := registryAuthURL + "?service=registry.docker.io&scope=" + url.QueryEscape("repository:"+repository+":pull")
	if err := fetchJSON(ctx, target, "", &auth); err != nil {
		return 0, "", err
	}
	var manifest struct {
		Config struct {
			Digest string `json:"digest"`
		} `json:"config"`
		Manifests []struct {
			Digest   string `json:"digest"`
			Platform struct {
				OS           string `json:"os"`
				Architecture string `json:"architecture"`
			} `json:"platform"`
		} `json:"manifests"`
	}
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/", registryBaseURL, repository)
	if err := fetchJSON(ctx, manifestURL+url.PathEscape(tag), auth.Token, &manifest); err != nil {
		return 0, "", err
	}
	if len(manifest.Manifests) > 0 {
		digest := ""
		for _, candidate := range manifest.Manifests {
			if candidate.Platform.OS == "linux" && (digest == "" || candidate.Platform.Architecture == "amd64") {
				digest = candidate.Digest
			}
			if candidate.Platform.OS == "linux" && candidate.Platform.Architecture == "amd64" {
				break
			}
		}
		if digest == "" {
			return 0, "", fmt.Errorf("%s has no linux image", repository)
		}
		if err := fetchJSON(ctx, manifestURL+digest, auth.Token, &manifest); err != nil {
			return 0, "", err
		}
	}
	var image struct {
		Config struct {
			ExposedPorts map[string]struct{} `json:"ExposedPorts"`
			Volumes      map[string]struct{} `json:"Volumes"`
		} `json:"config"`
	}
	if err := fetchJSON(ctx, fmt.Sprintf("%s/v2/%s/blobs/%s", registryBaseURL, repository, manifest.Config.Digest), auth.Token, &image); err != nil {
		return 0, "", err
	}
	port := 0
	for exposed := range image.Config.ExposedPorts {
		number, protocol, _ := strings.Cut(exposed, "/")
		if value, err := strconv.Atoi(number); err == nil && protocol != "udp" && (port == 0 || value < port) {
			port = value
		}
	}
	volumes := make([]string, 0, len(image.Config.Volumes))
	for volume := range image.Config.Volumes {
		if templates.ValidDataPath(volume) {
			volumes = append(volumes, volume)
		}
	}
	sort.Strings(volumes)
	if len(volumes) == 0 {
		return port, "", nil
	}
	return port, volumes[0], nil
}

func fetchJSON(ctx context.Context, target, token string, output any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json, application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := dockerHubClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("docker hub returned %s", response.Status)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(output)
}
