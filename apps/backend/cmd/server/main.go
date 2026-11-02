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
)

type task struct {
	ID           string           `json:"id"`
	ServiceID    string           `json:"service_id"`
	WorkspaceID  string           `json:"workspace_id"`
	Action       string           `json:"action"`
	Engine       string           `json:"engine"`
	Name         string           `json:"name"`
	ImageVersion string           `json:"image_version"`
	Image        string           `json:"image"`
	Port         int              `json:"port"`
	DataPath     string           `json:"data_path"`
	Environment  string           `json:"environment_id"`
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
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	fixedToken := os.Getenv("KYMBER_SERVER_TOKEN")
	tokenFile := os.Getenv("KYMBER_SERVER_TOKEN_FILE")
	if fixedToken == "" && tokenFile == "" {
		return errors.New("KYMBER_SERVER_TOKEN or KYMBER_SERVER_TOKEN_FILE is required")
	}
	if err := validateAPIURL(baseURL); err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for {
		ctx := context.Background()
		token := agentToken(fixedToken, tokenFile)
		if token == "" {
			time.Sleep(2 * time.Second)
			continue
		}
		if err := request(ctx, client, baseURL, token, "/api/agent/heartbeat", nil, nil); err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		time.Sleep(15 * time.Second)
	}
}

func agentToken(fixed, path string) string {
	if fixed != "" {
		return fixed
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func containerIDFrom(output string) string {
	return output[strings.LastIndex(output, "\n")+1:]
}

func fitResult(r result) result {
	if len(r.Message) > 2048 {
		r.Message = strings.ToValidUTF8(r.Message[len(r.Message)-2048:], "")
	}
	if len(r.ContainerID) > 128 {
		r.ContainerID = r.ContainerID[:128]
	}
	return r
}

func validateAPIURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("API URL is invalid")
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	local := host == "localhost" || host == "backend" || ip != nil && ip.IsLoopback()
	if parsed.Scheme == "https" || parsed.Scheme == "http" && local {
		return nil
	}
	return errors.New("API URL must use HTTPS unless it points to this machine or the local backend")
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
			imageTemplate, ok := serviceTemplate(t)
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
			if template, ok := serviceTemplate(t); !ok || template.DataPath != "" {
				if _, err := docker("volume", "rm", volumeName(t)); err != nil {
					return result{Message: err.Error()}
				}
			}
		}
		if t.Action == "delete" && t.Environment != "" {
			_, _ = docker("network", "rm", networkName(t))
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
	template, ok := serviceTemplate(t)
	if !ok {
		return result{Message: "unsupported engine"}
	}
	if !templates.ValidDataPath(template.DataPath) {
		return result{Message: "invalid data path"}
	}
	network := networkName(t)
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
	args := runArgs(t, template, network, envFile)
	engineArgs, ok := templates.DockerArgs(t.Engine, t.Config)
	if !ok {
		return result{Message: "unsupported engine"}
	}
	image, err := template.ImageRef(t.ImageVersion)
	if err != nil {
		return result{Message: err.Error()}
	}
	args = append(args, image)
	args = append(args, engineArgs...)
	output, err := docker(args...)
	if err != nil {
		return result{Message: err.Error()}
	}
	id := containerIDFrom(output)
	if err := waitHealthy(containerName(t), 2*time.Minute); err != nil {
		return result{Message: err.Error(), ContainerID: id}
	}
	return result{Success: true, ContainerID: id}
}

func networkName(t task) string {
	if t.Environment != "" {
		return fmt.Sprintf("kymber-workspace-%s-env-%s", t.WorkspaceID, t.Environment)
	}
	return fmt.Sprintf("kymber-workspace-%s", t.WorkspaceID)
}

func serviceTemplate(t task) (templates.Template, bool) {
	template, ok := templates.Get(t.Engine)
	return template.ForService(t.Image, t.Port, t.DataPath), ok
}

func runArgs(t task, template templates.Template, network, envFile string) []string {
	args := []string{"run", "--pull=missing", "-d", "--name", containerName(t), "--restart", "unless-stopped", "--network", network, "--network-alias", template.InternalHost(t.ServiceID), "--label", "kymber.managed=true", "--label", "kymber.service=" + t.ServiceID}
	if template.DataPath != "" {
		args = append(args, "--volume", template.VolumeName(t.ServiceID)+":"+template.DataPath)
	}
	if template.HealthCmd != "" {
		args = append(args, "--health-cmd", template.HealthCmd, "--health-interval", "10s", "--health-timeout", "5s", "--health-retries", "10", "--health-start-period", "30s")
	}
	return append(args, "--env-file", envFile)
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
	last, since := "", time.Now()
	for time.Now().Before(deadline) {
		state, err := docker("inspect", "-f", "{{.State.Status}} {{.RestartCount}} {{if .State.Health}}{{.State.Health.Status}}{{end}}", name)
		if err == nil {
			if state != last {
				last, since = state, time.Now()
			}
			ready, err := checkContainer(state, time.Since(since))
			if err != nil {
				return err
			}
			if ready {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return errors.New("container did not become ready in time")
}

const containerStableAfter = 10 * time.Second

func checkContainer(state string, stableFor time.Duration) (bool, error) {
	fields := strings.Fields(state)
	if len(fields) >= 3 {
		if fields[2] == "unhealthy" {
			return false, errors.New("container failed its health check")
		}
		return fields[2] == "healthy", nil
	}
	return len(fields) > 0 && fields[0] == "running" && stableFor >= containerStableAfter, nil
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
