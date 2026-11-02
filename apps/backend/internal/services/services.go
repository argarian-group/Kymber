package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/secrets"
	"kymber/backend/internal/templates"
	"kymber/backend/internal/templates/catalog"
)

type databaseService struct {
	ID              string           `json:"id"`
	WorkspaceID     string           `json:"workspace_id"`
	ProjectID       string           `json:"project_id"`
	Engine          string           `json:"engine"`
	Image           string           `json:"image"`
	ImageVersion    string           `json:"image_version"`
	ServerID        string           `json:"server_id,omitempty"`
	ServerName      string           `json:"server_name,omitempty"`
	Name            string           `json:"name"`
	DatabaseName    string           `json:"database_name"`
	DatabaseUser    string           `json:"database_user"`
	Status          string           `json:"status"`
	LastError       string           `json:"last_error,omitempty"`
	InternalHost    string           `json:"internal_host,omitempty"`
	Port            int              `json:"port,omitempty"`
	DataPath        string           `json:"data_path,omitempty"`
	LogoURL         string           `json:"logo_url,omitempty"`
	Category        string           `json:"category"`
	EnvironmentID   string           `json:"environment_id,omitempty"`
	EnvironmentName string           `json:"environment_name,omitempty"`
	ConnectionURL   string           `json:"connection_url,omitempty"`
	Environment     templates.Config `json:"environment,omitempty"`
}

var serviceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)

func (h *Handler) listServices(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, false) {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `SELECT d.id, d.workspace_id, d.project_id, d.engine, d.name, d.image_version, d.image, d.port, d.data_path, d.logo_url, d.category, d.environment_id, COALESCE(e.name, ''), COALESCE(d.server_id, ''), COALESCE(v.name, ''), d.database_name, d.database_user, d.status, COALESCE(d.last_error, '') FROM database_services d LEFT JOIN servers v ON v.id = d.server_id LEFT JOIN project_environments e ON e.id = d.environment_id WHERE d.workspace_id = ? AND d.project_id = ? ORDER BY d.created_at`, workspaceID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list services")
		return
	}
	defer rows.Close()
	services := make([]databaseService, 0)
	for rows.Next() {
		var service databaseService
		if err := rows.Scan(&service.ID, &service.WorkspaceID, &service.ProjectID, &service.Engine, &service.Name, &service.ImageVersion, &service.Image, &service.Port, &service.DataPath, &service.LogoURL, &service.Category, &service.EnvironmentID, &service.EnvironmentName, &service.ServerID, &service.ServerName, &service.DatabaseName, &service.DatabaseUser, &service.Status, &service.LastError); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read services")
			return
		}
		if template, ok := catalog.Get(service.Engine); ok {
			template = template.ForService(service.Image, service.Port, service.DataPath)
			service.Image, service.Port, service.DataPath = template.Image, 0, template.DataPath
		}
		service.Category = serviceCategory(service.Category)
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read services")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]databaseService{"services": services})
}

func (h *Handler) listDatabaseTemplates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string][]templates.Template{"templates": catalog.All()})
}

func (h *Handler) createService(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, true) {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return
	}
	var input struct {
		Engine        string           `json:"engine"`
		Name          string           `json:"name"`
		ImageVersion  string           `json:"image_version"`
		Environment   templates.Config `json:"environment"`
		Repository    string           `json:"repository"`
		Port          int              `json:"port"`
		DataPath      string           `json:"data_path"`
		LogoURL       string           `json:"logo_url"`
		Category      string           `json:"category"`
		EnvironmentID string           `json:"environment_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Engine = strings.ToLower(strings.TrimSpace(input.Engine))
	input.Name = strings.ToLower(strings.TrimSpace(input.Name))
	template, supported := catalog.Get(input.Engine)
	var err error
	if input.Name == "" && supported {
		suffix, err := randomNameSuffix()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not generate service name")
			return
		}
		input.Name = input.Engine + "-" + suffix[:6]
	}
	if !supported || !serviceNamePattern.MatchString(input.Name) {
		writeError(w, http.StatusBadRequest, "supported engine and valid name are required")
		return
	}
	environmentName := ""
	if input.EnvironmentID != "" {
		if err := h.db.QueryRowContext(r.Context(), "SELECT name FROM project_environments WHERE id = ? AND project_id = ?", input.EnvironmentID, projectID).Scan(&environmentName); err != nil {
			writeError(w, http.StatusBadRequest, "environment not found")
			return
		}
	}
	customImage, logoURL, category := "", dockerLogoURL(input.LogoURL), template.Category
	if template.Custom {
		repository, err := templates.ParseRepository(strings.TrimSpace(input.Repository))
		if err != nil || input.Port < 0 || input.Port > 65535 || !templates.ValidDataPath(input.DataPath) {
			writeError(w, http.StatusBadRequest, "valid repository, port and data path are required")
			return
		}
		customImage, category = "docker.io/"+repository, serviceCategory(input.Category)
		template = template.ForService(customImage, input.Port, input.DataPath)
	}
	if input.ImageVersion == "" {
		input.ImageVersion = template.DefaultVersion
	}
	if _, err := template.ImageRef(input.ImageVersion); err != nil {
		writeError(w, http.StatusBadRequest, "invalid image version")
		return
	}
	input.Environment, err = templates.Normalize(template, input.Environment)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	databaseName := input.Environment[template.Connection.DatabaseField]
	databaseUser := input.Environment[template.Connection.UserField]
	config, err := secrets.Encrypt(mustJSON(input.Environment))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "service encryption is not configured")
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate application id")
		return
	}
	_, err = h.db.ExecContext(r.Context(), `INSERT INTO database_services (id, workspace_id, project_id, engine, name, image_version, image, port, data_path, logo_url, category, environment_id, database_name, database_user, encrypted_config, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, workspaceID, projectID, input.Engine, input.Name, input.ImageVersion, customImage, input.Port, input.DataPath, logoURL, category, input.EnvironmentID, databaseName, databaseUser, config, user.ID)
	if err != nil {
		writeError(w, http.StatusConflict, "could not create service; its name may already be used")
		return
	}
	service := databaseService{ID: id, WorkspaceID: workspaceID, ProjectID: projectID, Engine: input.Engine, Image: template.Image, ImageVersion: input.ImageVersion, Name: input.Name, DatabaseName: databaseName, DatabaseUser: databaseUser, Status: "draft", InternalHost: template.InternalHost(id), Port: template.Port, DataPath: template.DataPath, LogoURL: logoURL, Category: category, EnvironmentID: input.EnvironmentID, EnvironmentName: environmentName, ConnectionURL: template.ConnectionURL(id, input.Environment), Environment: input.Environment}
	writeJSON(w, http.StatusCreated, service)
}

func (h *Handler) serviceAction(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, true) {
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
	action := r.PathValue("action")
	if action != "deploy" && action != "start" && action != "stop" && action != "restart" && action != "remove_container" && action != "remove_container_image" {
		writeError(w, http.StatusBadRequest, "unsupported service action")
		return
	}
	imageVersion, requestedServer := "", ""
	if action == "deploy" && r.ContentLength != 0 {
		var input struct {
			ImageVersion string `json:"image_version"`
			ServerID     string `json:"server_id"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		imageVersion, requestedServer = input.ImageVersion, input.ServerID
	}
	if action == "deploy" {
		service, err := h.readService(r, workspaceID, projectID, serviceID, false)
		if err != nil {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		template := serviceTemplate(service)
		if imageVersion == "" {
			imageVersion = service.ImageVersion
		}
		if _, err := template.ImageRef(imageVersion); err != nil {
			writeError(w, http.StatusBadRequest, "invalid image version")
			return
		}
	}
	var nullableServerID sql.NullString
	if err := h.db.QueryRowContext(r.Context(), "SELECT server_id FROM database_services WHERE id = ? AND workspace_id = ? AND project_id = ?", serviceID, workspaceID, projectID).Scan(&nullableServerID); err != nil {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if requestedServer != "" && nullableServerID.Valid && requestedServer != nullableServerID.String {
		writeError(w, http.StatusConflict, "app already runs on another server")
		return
	}
	if action == "deploy" && !nullableServerID.Valid {
		if requestedServer != "" {
			if !h.requireServerOnline(w, r, requestedServer) {
				return
			}
			nullableServerID = sql.NullString{String: requestedServer, Valid: true}
		} else if err := h.db.QueryRowContext(r.Context(), "SELECT s.id FROM servers s WHERE s.connection_type = 'local' AND "+serverStatusSQL+" = 'online' ORDER BY s.created_at LIMIT 1").Scan(&nullableServerID); errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusConflict, "no online server is available")
			return
		} else if err != nil {
			writeError(w, http.StatusInternalServerError, "could not find an online server")
			return
		}
		if _, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET server_id = ? WHERE id = ?", nullableServerID.String, serviceID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not assign server")
			return
		}
	}
	if !nullableServerID.Valid {
		writeError(w, http.StatusConflict, "service has not been deployed")
		return
	}
	var connectionType string
	if err := h.db.QueryRowContext(r.Context(), "SELECT connection_type FROM servers WHERE id = ?", nullableServerID.String).Scan(&connectionType); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read server connection")
		return
	}
	if connectionType == "local" && !h.requireServerOnline(w, r, nullableServerID.String) {
		return
	}
	if action == "remove_container" || action == "remove_container_image" {
		var status string
		if err := h.db.QueryRowContext(r.Context(), "SELECT status FROM database_services WHERE id = ?", serviceID).Scan(&status); err != nil || status != "stopped" {
			writeError(w, http.StatusConflict, "stop the application before removing its container")
			return
		}
	}
	serverID := nullableServerID.String
	var sshConfigured bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT encrypted_ssh_key IS NOT NULL FROM servers WHERE id = ?", serverID).Scan(&sshConfigured); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read server credentials")
		return
	}
	if action == "deploy" {
		ports, err := h.readPorts(r.Context(), serviceID)
		if err == nil {
			err = h.checkPortConflicts(r.Context(), serverID, serviceID, ports)
		}
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	}
	if connectionType == "ssh" || sshConfigured {
		if action != "deploy" {
			logFile, statusFile := deploymentLogFiles(serviceID)
			if _, err := h.executeSSH(r.Context(), serverID, "rm -f "+shellQuote(logFile)+" "+shellQuote(statusFile)+" "+shellQuote(statusFile+".tmp")); err != nil {
				writeError(w, http.StatusBadGateway, "could not clear previous deployment logs")
				return
			}
		}
		status := "deploying"
		if action == "stop" {
			status = "stopped"
		}
		if _, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET status = ?, last_error = NULL WHERE id = ?", status, serviceID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not update service status")
			return
		}
		if action == "deploy" {
			if err := h.startServiceDeploySSH(r, serverID, serviceID, imageVersion); err != nil {
				_, _ = h.db.ExecContext(r.Context(), "UPDATE database_services SET status = 'failed', last_error = ? WHERE id = ?", err.Error(), serviceID)
				writeError(w, http.StatusBadGateway, "SSH service deployment failed: "+err.Error())
				return
			}
			go h.watchDeploy(serverID, serviceID, false)
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if err := h.executeServiceSSH(r, serverID, serviceID, action, imageVersion); err != nil {
			_, _ = h.db.ExecContext(r.Context(), "UPDATE database_services SET status = 'failed', last_error = ? WHERE id = ?", err.Error(), serviceID)
			writeError(w, http.StatusBadGateway, "SSH service action failed: "+err.Error())
			return
		}
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if err := h.queueServiceAction(r, serverID, serviceID, action, imageVersion); err != nil {
		writeError(w, http.StatusInternalServerError, "could not queue service action")
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) serviceLogs(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, false) {
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
	service, err := h.readService(r, workspaceID, projectID, serviceID, false)
	if err != nil {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if service.ServerID == "" {
		writeError(w, http.StatusConflict, "service has not been deployed")
		return
	}
	var configured bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT encrypted_ssh_key IS NOT NULL FROM servers WHERE id = ?", service.ServerID).Scan(&configured); err != nil || !configured {
		writeError(w, http.StatusConflict, "server does not support SSH logs")
		return
	}
	name := service.Name + "-service-" + service.ID
	if r.URL.Query().Get("source") == "runtime" {
		tail, ok := parseLogTail(r.URL.Query().Get("tail"))
		if !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("tail must be between 1 and %d", maxLogTail))
			return
		}
		logs, err := h.executeSSH(r.Context(), service.ServerID, runtimeLogsCommand(name, tail))
		if err != nil {
			writeError(w, http.StatusBadGateway, "could not read container logs: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": service.Status, "logs": string(logs)})
		return
	}
	logFile, statusFile := deploymentLogFiles(service.ID)
	command := "if test -f " + shellQuote(logFile) + "; then printf '__KYMBER_LOG__'; tail -c 65536 " + shellQuote(logFile) + "; else printf '__KYMBER_NO_LOG__'; fi; if test -f " + shellQuote(statusFile) + "; then code=$(cat " + shellQuote(statusFile) + "); printf '\\n__KYMBER_EXIT__%s' \"$code\"; if test \"$code\" = 0; then printf '\\n__KYMBER_CONTAINER__'; if docker inspect " + shellQuote(name) + " >/dev/null 2>&1; then docker logs --tail 300 " + shellQuote(name) + " 2>&1; else printf 'Container output is not available yet.'; fi; fi; fi"
	logs, err := h.executeSSH(r.Context(), service.ServerID, command)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not read deployment logs: "+err.Error())
		return
	}
	status := service.Status
	if (service.Status == "deploying" || service.Status == "running") && strings.Contains(string(logs), "__KYMBER_LOG__") {
		_, output, _ := strings.Cut(string(logs), "__KYMBER_LOG__")
		if before, after, found := strings.Cut(output, "__KYMBER_EXIT__"); found {
			output = before
			code, after, _ := strings.Cut(after, "\n")
			code = strings.TrimSpace(code)
			if code == "0" {
				status = "running"
				h.finishDeploy(r.Context(), serviceID, true, "")
				if _, container, found := strings.Cut(after, "__KYMBER_CONTAINER__"); found {
					output = strings.TrimSpace(output) + "\n\n--- Container output ---\n" + strings.TrimSpace(container)
				}
			} else {
				status = "failed"
				h.finishDeploy(r.Context(), serviceID, false, strings.TrimSpace(output))
			}
		}
		logs = []byte(strings.TrimSpace(output))
	} else if service.Status == "deploying" || service.Status == "draft" || service.Status == "queued" {
		logs = []byte("Preparing Docker deployment…")
	} else {
		logs, err = h.executeSSH(r.Context(), service.ServerID, "if docker inspect "+shellQuote(name)+" >/dev/null 2>&1; then docker logs --tail 300 "+shellQuote(name)+" 2>&1; else printf 'Container is not running.'; fi")
		if err != nil {
			writeError(w, http.StatusBadGateway, "could not read container logs: "+err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status, "logs": string(logs)})
}

const (
	maxLogTail  = 50000
	maxLogBytes = 10 << 20
)

func parseLogTail(raw string) (int, bool) {
	if raw == "" {
		return 300, true
	}
	tail, err := strconv.Atoi(raw)
	return tail, err == nil && tail >= 1 && tail <= maxLogTail
}

// The byte cap keeps a few enormous lines from loading the whole log into the API's memory.
func runtimeLogsCommand(container string, tail int) string {
	return fmt.Sprintf("if docker inspect %[1]s >/dev/null 2>&1; then docker logs --tail %[2]d %[1]s 2>&1 | tail -c %[3]d; else printf 'Container is not running.\\n'; fi", shellQuote(container), tail, maxLogBytes)
}

func (h *Handler) executeServiceSSH(r *http.Request, serverID, serviceID, action, imageVersion string) error {
	service, err := h.readServiceByID(r, serviceID)
	if err != nil {
		return err
	}
	if action != "deploy" {
		imageVersion = service.ImageVersion
	}
	template := serviceTemplate(service)
	imageRef, err := template.ImageRef(imageVersion)
	if err != nil {
		return err
	}
	status := "deploying"
	if action == "stop" {
		status = "stopped"
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET status = ?, last_error = NULL WHERE id = ?", status, serviceID); err != nil {
		return err
	}
	storages, err := h.readStorages(r.Context(), serviceID, false)
	if err != nil {
		return err
	}
	ports, err := h.readPorts(r.Context(), serviceID)
	if err != nil {
		return err
	}
	command, err := h.sshDockerCommand(service, template, imageRef, action, storages, ports)
	if err == nil {
		_, err = h.executeSSH(r.Context(), serverID, command)
	}
	if err != nil {
		_, _ = h.db.ExecContext(r.Context(), "UPDATE database_services SET status = 'failed', last_error = ? WHERE id = ?", err.Error(), serviceID)
		return err
	}
	status = "running"
	switch action {
	case "stop":
		status = "stopped"
	case "delete":
		status = "deleted"
	case "remove_container", "remove_container_image":
		status = "stopped"
	}
	_, err = h.db.ExecContext(r.Context(), "UPDATE database_services SET status = ?, last_error = NULL WHERE id = ?", status, serviceID)
	return err
}

func (h *Handler) startServiceDeploySSH(r *http.Request, serverID, serviceID, imageVersion string) error {
	service, err := h.readServiceByID(r, serviceID)
	if err != nil {
		return err
	}
	template := serviceTemplate(service)
	image, err := template.ImageRef(imageVersion)
	if err != nil {
		return err
	}
	storages, err := h.readStorages(r.Context(), serviceID, true)
	if err != nil {
		return err
	}
	ports, err := h.readPorts(r.Context(), serviceID)
	if err != nil {
		return err
	}
	command, err := h.sshDockerCommand(service, template, image, "deploy", storages, ports)
	if err != nil {
		return err
	}
	logFile, statusFile := deploymentLogFiles(serviceID)
	script := "(" + command + ") > " + shellQuote(logFile) + " 2>&1; code=$?; printf '%s' \"$code\" > " + shellQuote(statusFile+".tmp") + "; mv " + shellQuote(statusFile+".tmp") + " " + shellQuote(statusFile)
	launch := ": > " + shellQuote(logFile) + "; rm -f " + shellQuote(statusFile) + "; nohup sh -c " + shellQuote(script) + " </dev/null >/dev/null 2>&1 &"
	_, err = h.executeSSH(r.Context(), serverID, launch)
	return err
}

func deploymentLogFiles(serviceID string) (string, string) {
	base := "/tmp/kymber-service-" + serviceID
	return base + ".log", base + ".exit"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func (h *Handler) readServiceByID(r *http.Request, id string) (databaseService, error) {
	var service databaseService
	var encrypted []byte
	err := h.db.QueryRowContext(r.Context(), "SELECT id, workspace_id, project_id, engine, name, image_version, image, port, data_path, COALESCE(server_id, ''), database_name, database_user, encrypted_config FROM database_services WHERE id = ?", id).Scan(&service.ID, &service.WorkspaceID, &service.ProjectID, &service.Engine, &service.Name, &service.ImageVersion, &service.Image, &service.Port, &service.DataPath, &service.ServerID, &service.DatabaseName, &service.DatabaseUser, &encrypted)
	if err != nil {
		return service, err
	}
	plain, err := secrets.Decrypt(encrypted)
	if err != nil {
		return service, err
	}
	if err := json.Unmarshal(plain, &service.Environment); err != nil {
		return service, err
	}
	return service, nil
}

func (h *Handler) sshDockerCommand(s databaseService, t templates.Template, image, action string, storages []serviceStorage, ports []servicePort) (string, error) {
	name := t.ContainerName(s.Name, s.ID)
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
	if action == "stop" {
		return "docker stop " + quote(name), nil
	}
	if action == "start" {
		return "docker start " + quote(name), nil
	}
	if action == "restart" {
		return "docker restart " + quote(name), nil
	}
	if action == "remove_container" {
		return "docker rm -f " + quote(name), nil
	}
	if action == "remove_container_image" {
		return "docker rm -f " + quote(name) + " >/dev/null 2>&1 || true; docker image rm " + quote(image), nil
	}
	if action == "delete" {
		command := "docker rm -f " + quote(name) + " >/dev/null 2>&1 || true; " + storageDeleteCommand(s.ID) + ":"
		if t.DataPath != "" {
			command += "; docker volume rm " + quote(t.VolumeName(s.ID))
		}
		return command, nil
	}
	if action != "deploy" {
		return "", fmt.Errorf("unsupported action")
	}
	if !templates.ValidDataPath(t.DataPath) {
		return "", fmt.Errorf("invalid data path")
	}
	config, ok := catalog.DockerArgs(s.Engine, s.Environment)
	if !ok {
		return "", fmt.Errorf("unsupported engine")
	}
	var env strings.Builder
	for key, value := range s.Environment {
		env.WriteString("printf '%s\\n' ")
		env.WriteString(quote(key + "=" + value))
		env.WriteString(" >> \"$f\"; ")
	}
	prepare, storageArgs := storageDeployCommand(s.ID, storages)
	portCheck, portArgs := portDeployCommand(ports)
	command := "set -eu; n=" + quote(name) + "; f=$(mktemp); trap 'rm -f \"$f\"' EXIT; " + portCheck + env.String() + prepare + "docker network inspect " + quote("kymber-workspace-"+s.WorkspaceID) + " >/dev/null 2>&1 || docker network create --label kymber.managed=true " + quote("kymber-workspace-"+s.WorkspaceID) + " >/dev/null; docker rm -f \"$n\" >/dev/null 2>&1 || true; docker run --pull=missing -d --name \"$n\" --restart unless-stopped --network " + quote("kymber-workspace-"+s.WorkspaceID) + " --network-alias " + quote(t.InternalHost(s.ID)) + " --label kymber.managed=true --label " + quote("kymber.service="+s.ID)
	if t.DataPath != "" {
		command += " --volume " + quote(t.VolumeName(s.ID)+":"+t.DataPath)
	}
	if t.HealthCmd != "" {
		command += " --health-cmd " + quote(t.HealthCmd) + " --health-interval 10s --health-timeout 5s --health-retries 10 --health-start-period 30s"
	}
	command += storageArgs + portArgs + " --env-file \"$f\" " + quote(image)
	for _, arg := range config {
		command += " " + quote(arg)
	}
	return command, nil
}

func (h *Handler) getService(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok {
		return
	}
	if _, allowed := h.requireWorkspaceRole(w, r, user, workspaceID, true); !allowed {
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
	service, err := h.readService(r, workspaceID, projectID, serviceID, true)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read service")
		return
	}
	writeJSON(w, http.StatusOK, service)
}

func (h *Handler) updateService(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, true) {
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
	var input struct {
		Name         string           `json:"name"`
		ImageVersion string           `json:"image_version"`
		Environment  templates.Config `json:"environment"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	service, err := h.readService(r, workspaceID, projectID, serviceID, true)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read service")
		return
	}
	if service.Status != "draft" {
		writeError(w, http.StatusConflict, "only draft services can be configured")
		return
	}
	if input.Name != "" {
		service.Name = strings.ToLower(strings.TrimSpace(input.Name))
	}
	template := serviceTemplate(service)
	if input.ImageVersion != "" {
		if _, err := template.ImageRef(input.ImageVersion); err != nil {
			writeError(w, http.StatusBadRequest, "invalid image version")
			return
		}
		service.ImageVersion = input.ImageVersion
	}
	for key, value := range input.Environment {
		service.Environment[key] = value
	}
	service.Environment, err = templates.Normalize(template, service.Environment)
	if err != nil || !serviceNamePattern.MatchString(service.Name) {
		writeError(w, http.StatusBadRequest, "invalid service configuration")
		return
	}
	service.DatabaseName = service.Environment[template.Connection.DatabaseField]
	service.DatabaseUser = service.Environment[template.Connection.UserField]
	config, err := secrets.Encrypt(mustJSON(service.Environment))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "service encryption is not configured")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET name = ?, image_version = ?, database_name = ?, database_user = ?, encrypted_config = ? WHERE id = ? AND status = 'draft'", service.Name, service.ImageVersion, service.DatabaseName, service.DatabaseUser, config, serviceID); err != nil {
		writeError(w, http.StatusConflict, "could not update service configuration")
		return
	}
	service.ConnectionURL = template.ConnectionURL(serviceID, service.Environment)
	service.InternalHost = template.InternalHost(serviceID)
	writeJSON(w, http.StatusOK, service)
}

func (h *Handler) readService(r *http.Request, workspaceID, projectID string, serviceID string, includeSecret bool) (databaseService, error) {
	var service databaseService
	var encrypted []byte
	err := h.db.QueryRowContext(r.Context(), `SELECT d.id, d.workspace_id, d.project_id, d.engine, d.name, d.image_version, d.image, d.port, d.data_path, d.logo_url, d.category, d.environment_id, COALESCE(e.name, ''), COALESCE(d.server_id, ''), COALESCE(v.name, ''), d.database_name, d.database_user, d.status, COALESCE(d.last_error, ''), d.encrypted_config FROM database_services d LEFT JOIN servers v ON v.id = d.server_id LEFT JOIN project_environments e ON e.id = d.environment_id WHERE d.id = ? AND d.workspace_id = ? AND d.project_id = ?`, serviceID, workspaceID, projectID).Scan(&service.ID, &service.WorkspaceID, &service.ProjectID, &service.Engine, &service.Name, &service.ImageVersion, &service.Image, &service.Port, &service.DataPath, &service.LogoURL, &service.Category, &service.EnvironmentID, &service.EnvironmentName, &service.ServerID, &service.ServerName, &service.DatabaseName, &service.DatabaseUser, &service.Status, &service.LastError, &encrypted)
	if err != nil {
		return service, err
	}
	service.Category = serviceCategory(service.Category)
	template := serviceTemplate(service)
	service.Image, service.DataPath = template.Image, template.DataPath
	service.InternalHost, service.Port = template.InternalHost(serviceID), template.Port
	if includeSecret {
		plain, err := secrets.Decrypt(encrypted)
		if err != nil {
			return service, err
		}
		var config templates.Config
		if err := json.Unmarshal(plain, &config); err != nil {
			return service, err
		}
		service.Environment = config
		service.ConnectionURL = template.ConnectionURL(serviceID, config)
	}
	return service, nil
}

func serviceTemplate(service databaseService) templates.Template {
	template, _ := catalog.Get(service.Engine)
	return template.ForService(service.Image, service.Port, service.DataPath)
}

func (h *Handler) deleteService(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, true) {
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
	var serverID sql.NullString
	if err := h.db.QueryRowContext(r.Context(), "SELECT server_id FROM database_services WHERE id = ? AND workspace_id = ? AND project_id = ?", serviceID, workspaceID, projectID).Scan(&serverID); err != nil {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if !serverID.Valid {
		if _, err := h.db.ExecContext(r.Context(), "DELETE FROM database_services WHERE id = ? AND status = 'draft'", serviceID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not delete draft service")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !h.requireServerOnline(w, r, serverID.String) {
		return
	}
	var connectionType string
	var sshConfigured bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT connection_type, encrypted_ssh_key IS NOT NULL FROM servers WHERE id = ?", serverID.String).Scan(&connectionType, &sshConfigured); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read server connection")
		return
	}
	if connectionType == "ssh" || sshConfigured {
		if err := h.executeServiceSSH(r, serverID.String, serviceID, "delete", ""); err != nil {
			writeError(w, http.StatusBadGateway, "SSH app deletion failed: "+err.Error())
			return
		}
		if _, err := h.db.ExecContext(r.Context(), "DELETE FROM database_services WHERE id = ?", serviceID); err != nil {
			writeError(w, http.StatusInternalServerError, "container removed but could not delete app record")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := h.queueServiceAction(r, serverID.String, serviceID, "delete", ""); err != nil {
		writeError(w, http.StatusInternalServerError, "could not queue service deletion")
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) authorizeProject(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) bool {
	var exists bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM projects WHERE id = ? AND workspace_id = ?)", projectID, workspaceID).Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check project")
		return false
	}
	if !exists {
		writeError(w, http.StatusNotFound, "project not found")
		return false
	}
	return true
}

func (h *Handler) queueServiceAction(r *http.Request, serverID, serviceID string, action, imageVersion string) error {
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(), "UPDATE database_services SET status = 'queued', image_version = CASE WHEN ? = '' THEN image_version ELSE ? END WHERE id = ?", imageVersion, imageVersion, serviceID); err != nil {
		return err
	}
	taskID, err := database.NewShortID()
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(r.Context(), "INSERT INTO server_tasks (id, server_id, service_id, action) VALUES (?, ?, ?, ?)", taskID, serverID, serviceID, action); err != nil {
		return err
	}
	return tx.Commit()
}

func randomNameSuffix() (string, error) {
	data := make([]byte, 4)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func mustJSON(value any) []byte { data, _ := json.Marshal(value); return data }
