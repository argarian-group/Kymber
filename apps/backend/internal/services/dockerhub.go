package services

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
	"sync"
	"time"

	"kymber/backend/internal/templates"
)

var (
	dockerHubBaseURL = "https://hub.docker.com"
	registryAuthURL  = "https://auth.docker.io/token"
	registryBaseURL  = "https://registry-1.docker.io"
)

var dockerHubClient = &http.Client{Timeout: 15 * time.Second}
var dockerHubRequests = make(chan struct{}, 2)
var dockerHubSearchCache = struct {
	sync.Mutex
	entries map[string]dockerHubSearchResult
}{entries: make(map[string]dockerHubSearchResult)}

type dockerHubSearchResult struct {
	images  []dockerHubImage
	expires time.Time
}

type dockerHubImage struct {
	Repository  string `json:"repository"`
	Description string `json:"description"`
	Official    bool   `json:"official"`
	Stars       int    `json:"stars"`
	LogoURL     string `json:"logo_url,omitempty"`
	Category    string `json:"category"`
}

const (
	categoryApplication = "application"
	categoryDatabase    = "database"
	categoryService     = "service"
)

var dockerServiceCategories = map[string]bool{
	"api-management":               true,
	"integration-and-delivery":     true,
	"message-queues":               true,
	"monitoring-and-observability": true,
	"networking":                   true,
	"security":                     true,
	"web-analytics":                true,
	"web-servers":                  true,
}

func dockerImageCategory(slugs []string) string {
	category := categoryApplication
	for _, slug := range slugs {
		if slug == "databases-and-storage" {
			return categoryDatabase
		}
		if dockerServiceCategories[slug] {
			category = categoryService
		}
	}
	return category
}

func serviceCategory(stored string) string {
	if stored == categoryDatabase || stored == categoryService {
		return stored
	}
	return categoryApplication
}

func dockerLogoURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" {
		return ""
	}
	switch parsed.Host {
	case "djeqr6to3dedg.cloudfront.net":
		return raw
	case "www.gravatar.com":
		return parsed.String()
	case "avatars.githubusercontent.com":
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
	dockerHubSearchCache.Lock()
	if cached, ok := dockerHubSearchCache.entries[query]; ok && time.Now().Before(cached.expires) {
		dockerHubSearchCache.Unlock()
		writeJSON(w, http.StatusOK, map[string][]dockerHubImage{"images": cached.images})
		return
	}
	dockerHubSearchCache.Unlock()
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
			Categories []struct {
				Slug string `json:"slug"`
			} `json:"categories"`
		} `json:"results"`
	}
	target := dockerHubBaseURL + "/api/search/v4?size=25&type=image&query=" + url.QueryEscape(query)
	if err := fetchJSON(r.Context(), target, "", &response); err != nil {
		writeError(w, http.StatusBadGateway, "could not search Docker Hub")
		return
	}
	images := make([]dockerHubImage, 0, len(response.Results))
	for _, result := range response.Results {
		if result.Type != "image" {
			continue
		}
		logoURL := dockerLogoURL(result.LogoURL.Small)
		if logoURL == "" {
			if namespace, _, ok := strings.Cut(result.Slug, "/"); ok {
				for _, candidate := range response.Results {
					if candidate.Slug == namespace {
						logoURL = dockerLogoURL(candidate.LogoURL.Small)
						break
					}
				}
			}
		}
		slugs := make([]string, 0, len(result.Categories))
		for _, category := range result.Categories {
			slugs = append(slugs, category.Slug)
		}
		images = append(images, dockerHubImage{Repository: result.Slug, Description: result.ShortDescription, Official: result.Badge == "official", Stars: result.StarCount, LogoURL: logoURL, Category: dockerImageCategory(slugs)})
	}
	dockerHubSearchCache.Lock()
	if len(dockerHubSearchCache.entries) >= 256 {
		oldest, expiry := "", time.Time{}
		for key, entry := range dockerHubSearchCache.entries {
			if oldest == "" || entry.expires.Before(expiry) {
				oldest, expiry = key, entry.expires
			}
		}
		delete(dockerHubSearchCache.entries, oldest)
	}
	dockerHubSearchCache.entries[query] = dockerHubSearchResult{images: images, expires: time.Now().Add(24 * time.Hour)}
	dockerHubSearchCache.Unlock()
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
	if version != "latest" {
		var latest struct{}
		if err := fetchJSON(r.Context(), fmt.Sprintf("%s/v2/repositories/%s/tags/latest", dockerHubBaseURL, repository), "", &latest); err == nil {
			version = "latest"
			tags = append([]string{"latest"}, tags...)
		}
	}
	port, dataPath, _ := inspectImage(r.Context(), repository, version)
	writeJSON(w, http.StatusOK, map[string]any{"tags": tags, "default_version": version, "port": port, "data_path": dataPath})
}

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
	for attempt := 0; attempt < 4; attempt++ {
		dockerHubRequests <- struct{}{}
		response, err := dockerHubClient.Do(request)
		<-dockerHubRequests
		if err != nil {
			return err
		}
		if response.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Second
			if seconds, err := strconv.Atoi(response.Header.Get("Retry-After")); err == nil && seconds > 0 {
				delay = time.Duration(seconds) * time.Second
			}
			response.Body.Close()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("docker hub returned %s", response.Status)
		}
		return json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(output)
	}
	return fmt.Errorf("docker hub rate limit exceeded")
}
