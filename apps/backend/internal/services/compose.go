package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
)

func (h *Handler) readComposeContainers(r *http.Request, service databaseService) ([]composeContainer, error) {
	if service.ServerID == "" {
		return nil, nil
	}
	command := "docker compose -p " + shellQuote(service.Name) + " -f " + shellQuote("/tmp/kymber-compose-"+service.ID+".yaml") + " ps --all --format json"
	output, err := h.executeSSH(r.Context(), service.ServerID, command)
	if err != nil {
		return nil, err
	}
	type publisher struct {
		URL           string `json:"URL"`
		TargetPort    int    `json:"TargetPort"`
		PublishedPort int    `json:"PublishedPort"`
		Protocol      string `json:"Protocol"`
	}
	type composePSItem struct {
		Name       string      `json:"Name"`
		Service    string      `json:"Service"`
		Image      string      `json:"Image"`
		State      string      `json:"State"`
		Status     string      `json:"Status"`
		Publishers []publisher `json:"Publishers"`
	}
	items := []composePSItem{}
	if err := json.Unmarshal(output, &items); err != nil {
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if line == "" {
				continue
			}
			var item composePSItem
			if err := json.Unmarshal([]byte(line), &item); err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}
	containers := make([]composeContainer, 0, len(items))
	for _, item := range items {
		container := composeContainer{Name: item.Service, ContainerName: item.Name, Image: item.Image, State: item.State, Status: item.Status, Ports: []string{}}
		for _, port := range item.Publishers {
			if port.PublishedPort == 0 {
				container.Ports = append(container.Ports, fmt.Sprintf("%d/%s", port.TargetPort, port.Protocol))
			} else {
				container.Ports = append(container.Ports, fmt.Sprintf("%s:%d→%d/%s", port.URL, port.PublishedPort, port.TargetPort, port.Protocol))
			}
		}
		containers = append(containers, container)
	}
	return containers, nil
}

var composeUnsafePattern = regexp.MustCompile(`(?im)^\s*(privileged|network_mode|devices|cap_add|cap_drop|pid|ipc|uts|userns_mode|cgroup|cgroup_parent|runtime|security_opt|volumes_from|configs|secrets|include|extends|external|driver_opts)\s*:`)
var composeServiceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}$`)
var composeBindMountPattern = regexp.MustCompile(`(?m)^\s+-\s+["']?(?:/|\./|\.\./|~/|\$\{[^}]+\}/)`)
var composeBindTypePattern = regexp.MustCompile(`(?im)^\s*type\s*:\s*bind\s*$`)

func composeContainerExists(file, name string) bool {
	if !composeServiceNamePattern.MatchString(name) {
		return false
	}
	inside := false
	for _, line := range strings.Split(file, "\n") {
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 0 && trimmed == "services:" {
			inside = true
			continue
		}
		if inside && indent == 0 {
			break
		}
		if inside && indent == 2 && strings.TrimSuffix(trimmed, ":") == name && strings.HasSuffix(trimmed, ":") {
			return true
		}
	}
	return false
}

func (h *Handler) composeContainerAction(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return
	}
	serviceID, ok := pathID(w, r, "serviceID")
	if !ok {
		return
	}
	container := r.PathValue("container")
	action := r.PathValue("action")
	if (action != "start" && action != "stop") || !composeServiceNamePattern.MatchString(container) {
		writeError(w, http.StatusBadRequest, "unsupported Compose container action")
		return
	}
	service, err := h.readService(r, workspaceID, projectID, serviceID, false)
	if err != nil || !service.Compose {
		writeError(w, http.StatusNotFound, "Compose stack not found")
		return
	}
	if service.Status == "deploying" || service.Status == "queued" {
		writeError(w, http.StatusConflict, "wait for the stack deployment to finish")
		return
	}
	if !composeContainerExists(service.ComposeFile, container) {
		writeError(w, http.StatusBadRequest, "unknown Compose service")
		return
	}
	if service.ServerID == "" {
		writeError(w, http.StatusConflict, "stack has not been deployed")
		return
	}
	command := "docker compose -p " + shellQuote(service.Name) + " -f " + shellQuote("/tmp/kymber-compose-"+service.ID+".yaml") + " " + action + " " + shellQuote(container)
	if _, err := h.executeSSH(r.Context(), service.ServerID, command); err != nil {
		writeError(w, http.StatusBadGateway, "could not "+action+" Compose service: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateComposeFile(file string) error {
	if len(file) == 0 || len(file) > 65536 || strings.ContainsRune(file, '\x00') || !strings.Contains(file, "services:") {
		return fmt.Errorf("compose file must be valid text under 64 KB and define services")
	}
	if composeUnsafePattern.MatchString(file) || composeBindMountPattern.MatchString(file) || composeBindTypePattern.MatchString(file) || regexp.MustCompile(`(?im)^\s*volumes\s*:\s*\[`).MatchString(file) {
		return fmt.Errorf("compose file uses unsupported or unsafe options")
	}
	services := false
	serviceCount := 0
	serviceImages := map[string]bool{}
	currentService := ""
	for _, line := range strings.Split(file, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "services:" {
			services = true
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if !services || trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "-") && indent == 2 {
			name := strings.TrimSuffix(trimmed, ":")
			if !composeServiceNamePattern.MatchString(name) {
				return fmt.Errorf("invalid compose service name")
			}
			serviceCount++
			currentService = name
		}
		if indent >= 4 && strings.HasPrefix(trimmed, "build:") {
			return fmt.Errorf("build directives are not supported")
		}
		if indent >= 4 && strings.HasPrefix(trimmed, "image:") {
			image := strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "image:")), "\"'")
			if image == "" || strings.ContainsAny(image, " \t") || strings.Contains(image, "${") {
				return fmt.Errorf("every service must use a fixed published image")
			}
			serviceImages[currentService] = true
		}
	}
	if !services || serviceCount == 0 || serviceCount > 20 {
		return fmt.Errorf("compose file must define between 1 and 20 services")
	}
	if len(serviceImages) != serviceCount {
		return fmt.Errorf("every service must use a fixed published image")
	}
	return nil
}

func composePrimaryService(file string) string {
	inServices := false
	for _, line := range strings.Split(file, "\n") {
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent == 0 {
			inServices = trimmed == "services:"
			continue
		}
		if inServices && indent == 2 && strings.HasSuffix(trimmed, ":") {
			return strings.TrimSuffix(trimmed, ":")
		}
	}
	return ""
}

func (h *Handler) createComposeService(w http.ResponseWriter, r *http.Request, user auth.User, workspaceID, projectID, name, environmentID, file string) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = "stack-" + strings.ToLower(strings.TrimSpace(projectID))
	}
	if !serviceNamePattern.MatchString(name) {
		writeError(w, http.StatusBadRequest, "a valid stack name is required")
		return
	}
	if strings.HasPrefix(name, "kymber") {
		writeError(w, http.StatusBadRequest, "stack name is reserved")
		return
	}
	if err := validateComposeFile(file); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if environmentID != "" {
		var exists bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM project_environments WHERE id = ? AND project_id = ?)", environmentID, projectID).Scan(&exists); err != nil || !exists {
			writeError(w, http.StatusBadRequest, "environment not found")
			return
		}
	}
	var serverID string
	if err := h.db.QueryRowContext(r.Context(), "SELECT s.id FROM servers s WHERE s.connection_type = 'local' AND "+serverStatusSQL+" = 'online' ORDER BY s.created_at LIMIT 1").Scan(&serverID); err != nil {
		writeError(w, http.StatusConflict, "Compose stacks require an online local server")
		return
	}
	service := databaseService{ID: "check-" + name, Name: name, ComposeFile: file, Compose: true}
	if err := h.validateComposeOnServer(r, serverID, service, file); err != nil {
		writeError(w, http.StatusBadRequest, "invalid Compose file: "+err.Error())
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate stack id")
		return
	}
	_, err = h.db.ExecContext(r.Context(), `INSERT INTO database_services (id, workspace_id, project_id, engine, name, image_version, image, category, environment_id, encrypted_config, compose_file, compose_project, created_by) VALUES (?, ?, ?, 'compose', ?, 'latest', 'docker.io/library/compose', 'application', ?, '{}', ?, ?, ?)`, id, workspaceID, projectID, name, environmentID, file, name, user.ID)
	if err != nil {
		writeError(w, http.StatusConflict, "could not create stack; its name may already be used")
		return
	}
	writeJSON(w, http.StatusCreated, databaseService{ID: id, WorkspaceID: workspaceID, ProjectID: projectID, Engine: "compose", Name: name, Image: "docker.io/library/compose", ImageVersion: "latest", Status: "draft", Category: "application", EnvironmentID: environmentID, Compose: true})
}

func (h *Handler) composeSSHCommand(ctx context.Context, service databaseService, action string) (string, error) {
	file := service.ComposeFile
	if err := validateComposeFile(file); err != nil {
		return "", err
	}
	path := "/tmp/kymber-compose-" + service.ID + ".yaml"
	project := service.Name
	compose := "docker compose -p " + shellQuote(project) + " -f " + shellQuote(path)
	baseCompose := compose
	ports, err := h.readPorts(ctx, service.ID)
	if err != nil {
		return "", err
	}
	portsPath := "/tmp/kymber-compose-" + service.ID + "-ports.yaml"
	write := "mkdir -p " + shellQuote(filepath.Dir(path)) + " && printf %s " + shellQuote(base64.StdEncoding.EncodeToString([]byte(file))) + " | base64 -d > " + shellQuote(path)
	if len(ports) > 0 || service.Domain != "" {
		primaryService := composePrimaryService(file)
		if primaryService == "" {
			return "", fmt.Errorf("could not find a primary Compose service")
		}
		var override strings.Builder
		fmt.Fprintf(&override, "services:\n  %s:\n", strconv.Quote(primaryService))
		if len(ports) > 0 {
			override.WriteString("    ports:\n")
			for _, port := range ports {
				fmt.Fprintf(&override, "      - %s\n", strconv.Quote(fmt.Sprintf("%s:%d:%d/%s", port.Bind, port.HostPort, port.ContainerPort, port.Protocol)))
			}
		}
		if service.Domain != "" {
			name := "kymber-" + service.ID
			rule := "Host(`" + service.Domain + "`)"
			labels := []string{
				"traefik.enable=true",
				"traefik.docker.network=kymber",
				"traefik.http.routers." + name + "-http.rule=" + rule,
				"traefik.http.routers." + name + "-http.entrypoints=web",
				"traefik.http.routers." + name + "-http.middlewares=" + name + "-https",
				"traefik.http.routers." + name + "-http.service=" + name,
				"traefik.http.middlewares." + name + "-https.redirectscheme.scheme=https",
				"traefik.http.routers." + name + ".rule=" + rule,
				"traefik.http.routers." + name + ".entrypoints=websecure",
				"traefik.http.routers." + name + ".tls.certresolver=letsencrypt",
				"traefik.http.routers." + name + ".service=" + name,
			}
			override.WriteString("    labels:\n")
			for _, label := range labels {
				fmt.Fprintf(&override, "      - %s\n", strconv.Quote(label))
			}
		}
		write += " && printf %s " + shellQuote(base64.StdEncoding.EncodeToString([]byte(override.String()))) + " | base64 -d > " + shellQuote(portsPath)
		compose += " -f " + shellQuote(portsPath)
	}
	switch action {
	case "deploy", "start", "restart":
		command := write + " && " + compose + " config --quiet && " + compose + " up -d --remove-orphans"
		if service.Domain != "" {
			primaryService := composePrimaryService(file)
			command = "docker network inspect kymber >/dev/null 2>&1 || { echo 'Traefik network kymber is missing; start the Kymber proxy first' >&2; exit 1; }; " + command
			command += " && for c in $(" + compose + " ps -q " + shellQuote(primaryService) + "); do docker network connect kymber \"$c\" 2>/dev/null || docker inspect \"$c\" --format '{{json .NetworkSettings.Networks}}' | grep -q '\"kymber\"' || exit 1; done"
		}
		return command, nil
	case "stop":
		return "docker ps -q --filter label=com.docker.compose.project=" + shellQuote(project) + " | xargs -r docker stop", nil
	case "delete":
		return baseCompose + " down --remove-orphans && rm -f " + shellQuote(path) + " " + shellQuote(portsPath), nil
	case "validate":
		return "set -eu; path=$(mktemp /tmp/kymber-compose-check-XXXXXX.yaml); trap 'rm -f \"$path\"' EXIT; printf %s " + shellQuote(base64.StdEncoding.EncodeToString([]byte(file))) + " | base64 -d > \"$path\"; docker compose -f \"$path\" config --quiet", nil
	default:
		return "", fmt.Errorf("unsupported compose action")
	}
}

func (h *Handler) validateComposeOnServer(r *http.Request, serverID string, service databaseService, file string) error {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	command := "set -eu; path=$(mktemp /tmp/kymber-compose-check-XXXXXX.yaml); trap 'rm -f \"$path\"' EXIT; printf %s " + shellQuote(base64.StdEncoding.EncodeToString([]byte(file))) + " | base64 -d > \"$path\"; docker compose -p " + shellQuote(service.Name) + " -f \"$path\" config --quiet"
	_, err := h.executeSSH(ctx, serverID, command)
	return err
}

func (h *Handler) startComposeDeploySSH(r *http.Request, serverID, serviceID, action string) error {
	service, err := h.readServiceByID(r, serviceID)
	if err != nil {
		return err
	}
	command, err := h.composeSSHCommand(r.Context(), service, action)
	if err != nil {
		return err
	}
	logFile, statusFile := deploymentLogFiles(serviceID)
	pidFile := deploymentProcessFile(serviceID)
	script := "echo $$ > " + shellQuote(pidFile) + "; (" + command + ") > " + shellQuote(logFile) + " 2>&1; code=$?; rm -f " + shellQuote(pidFile) + "; printf '%s' \"$code\" > " + shellQuote(statusFile+".tmp") + "; mv " + shellQuote(statusFile+".tmp") + " " + shellQuote(statusFile)
	launch := ": > " + shellQuote(logFile) + "; rm -f " + shellQuote(statusFile) + " " + shellQuote(pidFile) + "; nohup setsid sh -c " + shellQuote(script) + " </dev/null >/dev/null 2>&1 &"
	_, err = h.executeSSH(r.Context(), serverID, launch)
	return err
}

func (h *Handler) executeComposeSSH(r *http.Request, serverID string, service databaseService, action string) error {
	command, err := h.composeSSHCommand(r.Context(), service, action)
	if err != nil {
		return err
	}
	_, err = h.executeSSH(r.Context(), serverID, command)
	return err
}
