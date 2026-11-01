package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"kymber/backend/internal/templates"
	"kymber/backend/internal/templates/catalog"
)

type task struct {
	ID           string           `json:"id"`
	ServiceID    string           `json:"service_id"`
	WorkspaceID  string           `json:"workspace_id"`
	Action       string           `json:"action"`
	Engine       string           `json:"engine"`
	Name         string           `json:"name"`
	ImageVersion string           `json:"image_version"`
	Database     string           `json:"database_name"`
	Username     string           `json:"database_user"`
	Password     string           `json:"password"`
	Config       templates.Config `json:"config"`
}

type result struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ContainerID string `json:"container_id"`
}

func main() {
	_ = godotenv.Load("../../.env")
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	baseURL := strings.TrimRight(os.Getenv("KYMBER_API_URL"), "/")
	token := os.Getenv("KYMBER_SERVER_TOKEN")
	if baseURL == "" || token == "" {
		return errors.New("KYMBER_API_URL and KYMBER_SERVER_TOKEN are required")
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" || parsedURL.Hostname() != "localhost" && (net.ParseIP(parsedURL.Hostname()) == nil || !net.ParseIP(parsedURL.Hostname()).IsLoopback()) || parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("API URL must use loopback HTTP or HTTPS")
	}
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for {
		ctx := context.Background()
		if err := request(ctx, client, baseURL, token, "/api/agent/heartbeat", nil, nil); err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		var current task
		err := request(ctx, client, baseURL, token, "/api/agent/tasks/claim", nil, &current)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if current.ID == "" {
			time.Sleep(2 * time.Second)
			continue
		}
		outcome := execute(current)
		body, _ := json.Marshal(outcome)
		for {
			err := request(ctx, client, baseURL, token, fmt.Sprintf("/api/agent/tasks/%s/result", current.ID), body, nil)
			if err == nil {
				break
			}
			fmt.Fprintf(os.Stderr, "could not report task %s result: %v\n", current.ID, err)
			time.Sleep(5 * time.Second)
		}
	}
}

func request(ctx context.Context, client *http.Client, baseURL, token, path string, body []byte, output any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("API request failed: %s", response.Status)
	}
	if output != nil {
		return json.NewDecoder(response.Body).Decode(output)
	}
	return nil
}

func execute(t task) result {
	if t.Action == "stop" || t.Action == "delete" || t.Action == "restart" || t.Action == "remove_container" || t.Action == "remove_container_image" {
		verb := "stop"
		if t.Action == "delete" || t.Action == "remove_container" || t.Action == "remove_container_image" {
			verb = "rm"
		}
		if t.Action == "restart" {
			verb = "restart"
		}
		args := []string{verb}
		if t.Action == "delete" {
			args = append(args, "-f")
		}
		args = append(args, containerName(t))
		if _, err := docker(args...); err != nil && t.Action != "delete" {
			return result{Message: err.Error()}
		}
		if t.Action == "remove_container_image" {
			imageTemplate, ok := catalog.Get(t.Engine)
			if !ok {
				return result{Message: "unsupported engine"}
			}
			image, err := imageTemplate.ImageRef(t.ImageVersion)
			if err != nil {
				return result{Message: err.Error()}
			}
			if _, err := docker("image", "rm", image); err != nil {
				return result{Message: err.Error()}
			}
		}
		if t.Action == "delete" {
			if _, err := docker("volume", "rm", volumeName(t)); err != nil {
				return result{Message: err.Error()}
			}
		}
		if t.Action == "stop" || t.Action == "delete" || t.Action == "remove_container" || t.Action == "remove_container_image" {
			return result{Success: true, Message: t.Action}
		}
	}
	if t.Action == "start" {
		if _, err := docker("start", containerName(t)); err != nil {
			return result{Message: err.Error()}
		}
		id, err := docker("inspect", "-f", "{{.Id}}", containerName(t))
		if err != nil {
			return result{Message: err.Error()}
		}
		return result{Success: true, ContainerID: id}
	}
	if t.Action == "restart" {
		id, err := docker("inspect", "-f", "{{.Id}}", containerName(t))
		if err != nil {
			return result{Message: err.Error()}
		}
		return result{Success: true, ContainerID: id}
	}
	if t.Action != "deploy" {
		return result{Message: "unsupported action"}
	}
	template, ok := catalog.Get(t.Engine)
	if !ok {
		return result{Message: "unsupported engine"}
	}
	network := fmt.Sprintf("kymber-workspace-%s", t.WorkspaceID)
	if _, err := docker("network", "inspect", network); err != nil {
		if _, err = docker("network", "create", "--label", "kymber.managed=true", network); err != nil {
			return result{Message: err.Error()}
		}
	}
	_, _ = docker("rm", "-f", containerName(t))
	envFile, err := writeEnvironmentFile(t.Config)
	if err != nil {
		return result{Message: "could not prepare container environment"}
	}
	defer os.Remove(envFile)
	args := []string{"run", "--pull=missing", "-d", "--name", containerName(t), "--restart", "unless-stopped", "--network", network, "--network-alias", template.InternalHost(t.ServiceID), "--label", "kymber.managed=true", "--label", "kymber.service=" + t.ServiceID, "--volume", template.VolumeName(t.ServiceID) + ":" + template.DataPath, "--health-cmd", template.HealthCmd, "--health-interval", "10s", "--health-timeout", "5s", "--health-retries", "10", "--health-start-period", "30s"}
	args = append(args, "--env-file", envFile)
	engineArgs, ok := catalog.DockerArgs(t.Engine, t.Config)
	if !ok {
		return result{Message: "unsupported engine"}
	}
	image, err := template.ImageRef(t.ImageVersion)
	if err != nil {
		return result{Message: err.Error()}
	}
	args = append(args, image)
	args = append(args, engineArgs...)
	id, err := docker(args...)
	if err != nil {
		return result{Message: err.Error()}
	}
	if err := waitHealthy(containerName(t), 2*time.Minute); err != nil {
		return result{Message: err.Error(), ContainerID: id}
	}
	return result{Success: true, ContainerID: id}
}

func docker(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "docker", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker operation failed: %s", strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func waitHealthy(name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := docker("inspect", "-f", "{{.State.Health.Status}}", name)
		if err == nil && status == "healthy" {
			return nil
		}
		if err == nil && status == "unhealthy" {
			return errors.New("database container failed its health check")
		}
		time.Sleep(2 * time.Second)
	}
	return errors.New("database container health check timed out")
}

func containerName(t task) string { return fmt.Sprintf("%s-service-%s", t.Name, t.ServiceID) }
func volumeName(t task) string    { return fmt.Sprintf("kymber-service-%s-data", t.ServiceID) }

func writeEnvironmentFile(config templates.Config) (string, error) {
	file, err := os.CreateTemp("", "kymber-env-")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if err := file.Chmod(0600); err != nil {
		file.Close()
		os.Remove(path)
		return "", err
	}
	keys := make([]string, 0, len(config))
	for key := range config {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := config[key]
		if !environmentNamePattern.MatchString(key) || strings.ContainsAny(value, "\r\n\x00") {
			file.Close()
			os.Remove(path)
			return "", errors.New("invalid container environment")
		}
		if _, err := fmt.Fprintf(file, "%s=%s\n", key, value); err != nil {
			file.Close()
			os.Remove(path)
			return "", err
		}
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

var environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
