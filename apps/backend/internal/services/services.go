package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strings"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
	"kymber/backend/internal/secrets"
	"kymber/backend/internal/templates"
)

type databaseService struct {
	ID              string             `json:"id"`
	WorkspaceID     string             `json:"workspace_id"`
	ProjectID       string             `json:"project_id"`
	Engine          string             `json:"engine"`
	Image           string             `json:"image"`
	ImageVersion    string             `json:"image_version"`
	ServerID        string             `json:"server_id,omitempty"`
	ServerName      string             `json:"server_name,omitempty"`
	Name            string             `json:"name"`
	DatabaseName    string             `json:"database_name"`
	DatabaseUser    string             `json:"database_user"`
	Status          string             `json:"status"`
	LastError       string             `json:"last_error,omitempty"`
	InternalHost    string             `json:"internal_host,omitempty"`
	Port            int                `json:"port,omitempty"`
	Domain          string             `json:"domain,omitempty"`
	DataPath        string             `json:"data_path,omitempty"`
	LogoURL         string             `json:"logo_url,omitempty"`
	Category        string             `json:"category"`
	EnvironmentID   string             `json:"environment_id,omitempty"`
	EnvironmentName string             `json:"environment_name,omitempty"`
	ConnectionURL   string             `json:"connection_url,omitempty"`
	Environment     templates.Config   `json:"environment,omitempty"`
	Compose         bool               `json:"compose,omitempty"`
	ComposeFile     string             `json:"compose_file,omitempty"`
	Containers      []composeContainer `json:"containers,omitempty"`
	ContainersError string             `json:"containers_error,omitempty"`
}

var errServiceActionInProgress = errors.New("service action already in progress")

type composeContainer struct {
	Name          string   `json:"name"`
	ContainerName string   `json:"container_name"`
	Image         string   `json:"image"`
	State         string   `json:"state"`
	Status        string   `json:"status"`
	Ports         []string `json:"ports"`
}

var serviceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)
var serviceDomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)

func (h *Handler) listServices(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, "") {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `SELECT d.id, d.workspace_id, d.project_id, d.engine, d.name, d.image_version, d.image, d.port, COALESCE(d.domain, ''), d.data_path, d.logo_url, d.category, d.environment_id, COALESCE(e.name, ''), COALESCE(d.server_id, ''), COALESCE(v.name, ''), d.database_name, d.database_user, d.status, COALESCE(d.last_error, ''), d.compose_file IS NOT NULL, COALESCE(CAST(d.compose_file AS CHAR), '') FROM database_services d LEFT JOIN servers v ON v.id = d.server_id LEFT JOIN project_environments e ON e.id = d.environment_id WHERE d.workspace_id = ? AND d.project_id = ? AND (d.status <> 'deleted' OR d.compose_file IS NULL) ORDER BY d.created_at`, workspaceID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list services")
		return
	}
	defer rows.Close()
	services := make([]databaseService, 0)
	for rows.Next() {
		var service databaseService
		if err := rows.Scan(&service.ID, &service.WorkspaceID, &service.ProjectID, &service.Engine, &service.Name, &service.ImageVersion, &service.Image, &service.Port, &service.Domain, &service.DataPath, &service.LogoURL, &service.Category, &service.EnvironmentID, &service.EnvironmentName, &service.ServerID, &service.ServerName, &service.DatabaseName, &service.DatabaseUser, &service.Status, &service.LastError, &service.Compose, &service.ComposeFile); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read services")
			return
		}
		if service.Compose {
			service.Image = "Docker Compose stack"
			if service.Status != "draft" && service.Status != "queued" && service.Status != "deploying" {
				service.Containers, err = h.readComposeContainers(r, service)
				if err != nil {
					service.ContainersError = err.Error()
				}
			}
		} else if service.Image == "docker.io/kymber/build-artifacts" {
			service.Port = 0
		} else if template, ok := templates.Get(service.Engine); ok {
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
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates.All(), "git_builds_enabled": gitBuildsEnabled()})
}

func (h *Handler) createService(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
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
		ComposeFile   string           `json:"compose_file"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.ComposeFile != "" {
		h.createComposeService(w, r, user, workspaceID, projectID, input.Name, input.EnvironmentID, input.ComposeFile)
		return
	}
	input.Engine = strings.ToLower(strings.TrimSpace(input.Engine))
	input.Name = strings.ToLower(strings.TrimSpace(input.Name))
	template, supported := templates.Get(input.Engine)
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
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create service")
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), `INSERT INTO database_services (id, workspace_id, project_id, engine, name, image_version, image, port, data_path, logo_url, category, environment_id, database_name, database_user, encrypted_config, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, workspaceID, projectID, input.Engine, input.Name, input.ImageVersion, customImage, input.Port, input.DataPath, logoURL, category, input.EnvironmentID, databaseName, databaseUser, config, user.ID)
	if err != nil {
		writeError(w, http.StatusConflict, "could not create service; its name may already be used")
		return
	}
	for _, port := range template.Ports {
		portID, err := database.NewShortID()
		if err == nil {
			_, err = tx.ExecContext(r.Context(), "INSERT INTO service_ports (id, service_id, host_port, container_port, protocol, bind_address) VALUES (?, ?, ?, ?, 'tcp', '0.0.0.0')", portID, id, port, port)
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not create service ports")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create service")
		return
	}
	service := databaseService{ID: id, WorkspaceID: workspaceID, ProjectID: projectID, Engine: input.Engine, Image: template.Image, ImageVersion: input.ImageVersion, Name: input.Name, DatabaseName: databaseName, DatabaseUser: databaseUser, Status: "draft", InternalHost: template.InternalHost(id), Port: template.Port, DataPath: template.DataPath, LogoURL: logoURL, Category: category, EnvironmentID: input.EnvironmentID, EnvironmentName: environmentName, ConnectionURL: template.ConnectionURL(id, input.Environment), Environment: input.Environment}
	writeJSON(w, http.StatusCreated, service)
}

func (h *Handler) getService(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok {
		return
	}
	if !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
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
	var input struct {
		Name         string           `json:"name"`
		ImageVersion string           `json:"image_version"`
		Environment  templates.Config `json:"environment"`
		ComposeFile  string           `json:"compose_file"`
		Port         *int             `json:"port"`
		DataPath     *string          `json:"data_path"`
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
	if service.Compose {
		if input.ComposeFile == "" || validateComposeFile(input.ComposeFile) != nil {
			writeError(w, http.StatusBadRequest, "invalid Compose file")
			return
		}
		if !strings.Contains(input.ComposeFile, "image:") {
			writeError(w, http.StatusBadRequest, "every Compose service must use an existing image")
			return
		}
		var stackNameCount int
		if err := h.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM database_services WHERE project_id = ? AND environment_id = ? AND name = ? AND id <> ? AND compose_file IS NOT NULL", projectID, service.EnvironmentID, service.Name, serviceID).Scan(&stackNameCount); err != nil || stackNameCount > 0 {
			writeError(w, http.StatusConflict, "stack name is already in use")
			return
		}
		serverID := service.ServerID
		if serverID == "" {
			if err := h.db.QueryRowContext(r.Context(), "SELECT s.id FROM servers s WHERE s.connection_type = 'local' AND "+serverStatusSQL+" = 'online' ORDER BY s.created_at LIMIT 1").Scan(&serverID); err != nil {
				writeError(w, http.StatusConflict, "Compose stacks require an online local server")
				return
			}
		}
		if service.Status != "deploying" && service.Status != "queued" {
			if err := h.validateComposeOnServer(r, serverID, service, input.ComposeFile); err != nil {
				writeError(w, http.StatusBadRequest, "invalid Compose file")
				return
			}
		}
		result, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET compose_file = ? WHERE id = ?", input.ComposeFile, serviceID)
		if err != nil {
			writeError(w, http.StatusConflict, "could not update Compose file")
			return
		}
		if updated, _ := result.RowsAffected(); updated == 0 {
			var exists bool
			if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM database_services WHERE id = ? AND workspace_id = ? AND project_id = ?)", serviceID, workspaceID, projectID).Scan(&exists); err != nil || !exists {
				writeError(w, http.StatusNotFound, "service not found")
				return
			}
		}
		service.ComposeFile = input.ComposeFile
		writeJSON(w, http.StatusOK, service)
		return
	}
	if service.Status != "draft" {
		writeError(w, http.StatusConflict, "only draft services can be configured")
		return
	}
	if service.Image == "docker.io/kymber/build-artifacts" && input.Port != nil && (*input.Port < 0 || *input.Port > 65535) {
		writeError(w, http.StatusBadRequest, "port must be between 0 and 65535")
		return
	}
	if service.Image == "docker.io/kymber/build-artifacts" && input.DataPath != nil && !templates.ValidDataPath(*input.DataPath) {
		writeError(w, http.StatusBadRequest, "invalid data path")
		return
	}
	if input.Name != "" {
		service.Name = strings.ToLower(strings.TrimSpace(input.Name))
	}
	template := serviceTemplate(service)
	if input.ImageVersion != "" {
		if service.Image == "docker.io/kymber/build-artifacts" {
			if input.ImageVersion != service.ImageVersion {
				writeError(w, http.StatusBadRequest, "GitHub image versions are selected by their build job")
				return
			}
		} else if _, err := template.ImageRef(input.ImageVersion); err != nil {
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
	if input.Port != nil && service.Image == "docker.io/kymber/build-artifacts" {
		service.Port = *input.Port
	}
	if input.DataPath != nil && service.Image == "docker.io/kymber/build-artifacts" {
		service.DataPath = *input.DataPath
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET name = ?, image_version = ?, port = ?, data_path = ?, database_name = ?, database_user = ?, encrypted_config = ? WHERE id = ? AND status = 'draft'", service.Name, service.ImageVersion, service.Port, service.DataPath, service.DatabaseName, service.DatabaseUser, config, serviceID); err != nil {
		writeError(w, http.StatusConflict, "could not update service configuration")
		return
	}
	service.ConnectionURL = template.ConnectionURL(serviceID, service.Environment)
	service.InternalHost = template.InternalHost(serviceID)
	writeJSON(w, http.StatusOK, service)
}

func (h *Handler) updateServiceDomain(w http.ResponseWriter, r *http.Request) {
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
	var input struct {
		Domain string `json:"domain"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	domain := strings.ToLower(strings.TrimSpace(input.Domain))
	if domain != "" && (len(domain) > 253 || !serviceDomainPattern.MatchString(domain) || net.ParseIP(domain) != nil) {
		writeError(w, http.StatusBadRequest, "enter a valid public domain name")
		return
	}
	service, err := h.readService(r, workspaceID, projectID, serviceID, false)
	if err != nil {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if service.Status == "queued" || service.Status == "deploying" {
		writeError(w, http.StatusConflict, "wait for the current action to finish")
		return
	}
	if domain != "" && (service.Category != "application" || service.Port < 1 || service.Port > 65535) {
		writeError(w, http.StatusBadRequest, "public domains require an application with an HTTP port")
		return
	}
	if domain != "" && service.ServerID != "" {
		var connectionType string
		if err := h.db.QueryRowContext(r.Context(), "SELECT connection_type FROM servers WHERE id = ?", service.ServerID).Scan(&connectionType); err != nil || connectionType != "local" {
			writeError(w, http.StatusConflict, "public domains currently require the local server with Traefik")
			return
		}
	}
	if domain != "" {
		var panelDomain string
		if err := h.db.QueryRowContext(r.Context(), "SELECT COALESCE(panel_domain, '') FROM platform_settings WHERE id = 1").Scan(&panelDomain); err != nil {
			writeError(w, http.StatusInternalServerError, "could not check panel domain")
			return
		}
		if domain == panelDomain {
			writeError(w, http.StatusConflict, "domain is already assigned to the panel")
			return
		}
	}
	var savedDomain any
	if domain != "" {
		savedDomain = domain
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET domain = ? WHERE id = ? AND NOT EXISTS (SELECT 1 FROM platform_settings WHERE panel_domain = ? AND id = 1)", savedDomain, serviceID, domain); err != nil {
		writeError(w, http.StatusConflict, "could not save domain")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"domain": domain})
}

func (h *Handler) readService(r *http.Request, workspaceID, projectID string, serviceID string, includeSecret bool) (databaseService, error) {
	var service databaseService
	var encrypted []byte
	err := h.db.QueryRowContext(r.Context(), `SELECT d.id, d.workspace_id, d.project_id, d.engine, d.name, d.image_version, d.image, d.port, COALESCE(d.domain, ''), d.data_path, d.logo_url, d.category, d.environment_id, COALESCE(e.name, ''), COALESCE(d.server_id, ''), COALESCE(v.name, ''), d.database_name, d.database_user, d.status, COALESCE(d.last_error, ''), d.encrypted_config, d.compose_file IS NOT NULL, COALESCE(CAST(d.compose_file AS CHAR), '') FROM database_services d LEFT JOIN servers v ON v.id = d.server_id LEFT JOIN project_environments e ON e.id = d.environment_id WHERE d.id = ? AND d.workspace_id = ? AND d.project_id = ?`, serviceID, workspaceID, projectID).Scan(&service.ID, &service.WorkspaceID, &service.ProjectID, &service.Engine, &service.Name, &service.ImageVersion, &service.Image, &service.Port, &service.Domain, &service.DataPath, &service.LogoURL, &service.Category, &service.EnvironmentID, &service.EnvironmentName, &service.ServerID, &service.ServerName, &service.DatabaseName, &service.DatabaseUser, &service.Status, &service.LastError, &encrypted, &service.Compose, &service.ComposeFile)
	if err != nil {
		return service, err
	}
	service.Category = serviceCategory(service.Category)
	if service.Compose {
		service.Image = "Docker Compose stack"
		if service.Status != "draft" && service.Status != "queued" && service.Status != "deploying" {
			service.Containers, err = h.readComposeContainers(r, service)
			if err != nil {
				service.ContainersError = err.Error()
			}
		}
		return service, nil
	}
	if service.Image == "docker.io/kymber/build-artifacts" {
		service.InternalHost = "kymber-service-" + serviceID
		if service.DataPath != "" && !templates.ValidDataPath(service.DataPath) {
			return service, errors.New("invalid stored data path")
		}
		if includeSecret {
			plain, err := secrets.Decrypt(encrypted)
			if err != nil {
				return service, err
			}
			if err := json.Unmarshal(plain, &service.Environment); err != nil {
				return service, err
			}
		}
		return service, nil
	}
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
	template, _ := templates.Get(service.Engine)
	return template.ForService(service.Image, service.Port, service.DataPath)
}

func (h *Handler) deleteService(w http.ResponseWriter, r *http.Request) {
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
	var serverID sql.NullString
	var compose bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT server_id, compose_file IS NOT NULL FROM database_services WHERE id = ? AND workspace_id = ? AND project_id = ?", serviceID, workspaceID, projectID).Scan(&serverID, &compose); err != nil {
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
	if compose {
		if _, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET status = 'deleted', last_error = NULL WHERE id = ?", serviceID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not mark stack deleted")
			return
		}
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
		if compose {
			service, err := h.readService(r, workspaceID, projectID, serviceID, false)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not read stack")
				return
			}
			if err := h.executeComposeSSH(r, serverID.String, service, "delete"); err != nil {
				_, _ = h.db.ExecContext(r.Context(), "UPDATE database_services SET status = 'failed', last_error = ? WHERE id = ?", err.Error(), serviceID)
				writeError(w, http.StatusBadGateway, "Compose stack deletion failed: "+err.Error())
				return
			}
			if _, err := h.db.ExecContext(r.Context(), "DELETE FROM database_services WHERE id = ?", serviceID); err != nil {
				writeError(w, http.StatusInternalServerError, "stack removed but could not delete its record")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
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
	result, err := tx.ExecContext(r.Context(), "UPDATE database_services SET status = 'queued', image_version = CASE WHEN ? = '' THEN image_version ELSE ? END WHERE id = ? AND status NOT IN ('deploying', 'queued')", imageVersion, imageVersion, serviceID)
	if err != nil {
		return err
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		return errServiceActionInProgress
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
