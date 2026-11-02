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
	Domain          string           `json:"domain,omitempty"`
	DataPath        string           `json:"data_path,omitempty"`
	LogoURL         string           `json:"logo_url,omitempty"`
	Category        string           `json:"category"`
	EnvironmentID   string           `json:"environment_id,omitempty"`
	EnvironmentName string           `json:"environment_name,omitempty"`
	ConnectionURL   string           `json:"connection_url,omitempty"`
	Environment     templates.Config `json:"environment,omitempty"`
}

var serviceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)
var serviceDomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)

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
	rows, err := h.db.QueryContext(r.Context(), `SELECT d.id, d.workspace_id, d.project_id, d.engine, d.name, d.image_version, d.image, d.port, COALESCE(d.domain, ''), d.data_path, d.logo_url, d.category, d.environment_id, COALESCE(e.name, ''), COALESCE(d.server_id, ''), COALESCE(v.name, ''), d.database_name, d.database_user, d.status, COALESCE(d.last_error, '') FROM database_services d LEFT JOIN servers v ON v.id = d.server_id LEFT JOIN project_environments e ON e.id = d.environment_id WHERE d.workspace_id = ? AND d.project_id = ? ORDER BY d.created_at`, workspaceID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list services")
		return
	}
	defer rows.Close()
	services := make([]databaseService, 0)
	for rows.Next() {
		var service databaseService
		if err := rows.Scan(&service.ID, &service.WorkspaceID, &service.ProjectID, &service.Engine, &service.Name, &service.ImageVersion, &service.Image, &service.Port, &service.Domain, &service.DataPath, &service.LogoURL, &service.Category, &service.EnvironmentID, &service.EnvironmentName, &service.ServerID, &service.ServerName, &service.DatabaseName, &service.DatabaseUser, &service.Status, &service.LastError); err != nil {
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

func (h *Handler) updateServiceDomain(w http.ResponseWriter, r *http.Request) {
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
	err := h.db.QueryRowContext(r.Context(), `SELECT d.id, d.workspace_id, d.project_id, d.engine, d.name, d.image_version, d.image, d.port, COALESCE(d.domain, ''), d.data_path, d.logo_url, d.category, d.environment_id, COALESCE(e.name, ''), COALESCE(d.server_id, ''), COALESCE(v.name, ''), d.database_name, d.database_user, d.status, COALESCE(d.last_error, ''), d.encrypted_config FROM database_services d LEFT JOIN servers v ON v.id = d.server_id LEFT JOIN project_environments e ON e.id = d.environment_id WHERE d.id = ? AND d.workspace_id = ? AND d.project_id = ?`, serviceID, workspaceID, projectID).Scan(&service.ID, &service.WorkspaceID, &service.ProjectID, &service.Engine, &service.Name, &service.ImageVersion, &service.Image, &service.Port, &service.Domain, &service.DataPath, &service.LogoURL, &service.Category, &service.EnvironmentID, &service.EnvironmentName, &service.ServerID, &service.ServerName, &service.DatabaseName, &service.DatabaseUser, &service.Status, &service.LastError, &encrypted)
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
