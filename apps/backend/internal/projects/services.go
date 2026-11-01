package projects

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	ID            string           `json:"id"`
	WorkspaceID   string           `json:"workspace_id"`
	ProjectID     string           `json:"project_id"`
	Engine        string           `json:"engine"`
	Image         string           `json:"image"`
	ImageVersion  string           `json:"image_version"`
	ServerID      string           `json:"server_id,omitempty"`
	ServerName    string           `json:"server_name,omitempty"`
	Name          string           `json:"name"`
	DatabaseName  string           `json:"database_name"`
	DatabaseUser  string           `json:"database_user"`
	Status        string           `json:"status"`
	LastError     string           `json:"last_error,omitempty"`
	InternalHost  string           `json:"internal_host,omitempty"`
	Port          int              `json:"port,omitempty"`
	ConnectionURL string           `json:"connection_url,omitempty"`
	Environment   templates.Config `json:"environment,omitempty"`
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
	rows, err := h.db.QueryContext(r.Context(), `SELECT d.id, d.workspace_id, d.project_id, d.engine, d.name, d.image_version, COALESCE(d.server_id, ''), COALESCE(v.name, ''), d.database_name, d.database_user, d.status, COALESCE(d.last_error, '') FROM database_services d LEFT JOIN servers v ON v.id = d.server_id WHERE d.workspace_id = ? AND d.project_id = ? ORDER BY d.created_at`, workspaceID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list services")
		return
	}
	defer rows.Close()
	services := make([]databaseService, 0)
	for rows.Next() {
		var service databaseService
		if err := rows.Scan(&service.ID, &service.WorkspaceID, &service.ProjectID, &service.Engine, &service.Name, &service.ImageVersion, &service.ServerID, &service.ServerName, &service.DatabaseName, &service.DatabaseUser, &service.Status, &service.LastError); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read services")
			return
		}
		if template, ok := catalog.Get(service.Engine); ok {
			service.Image = template.Image
		}
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
		Engine       string           `json:"engine"`
		Name         string           `json:"name"`
		ImageVersion string           `json:"image_version"`
		Environment  templates.Config `json:"environment"`
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
	_, err = h.db.ExecContext(r.Context(), `INSERT INTO database_services (id, workspace_id, project_id, engine, name, image_version, database_name, database_user, encrypted_config, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, workspaceID, projectID, input.Engine, input.Name, input.ImageVersion, databaseName, databaseUser, config, user.ID)
	if err != nil {
		writeError(w, http.StatusConflict, "could not create service; its name may already be used")
		return
	}
	service := databaseService{ID: id, WorkspaceID: workspaceID, ProjectID: projectID, Engine: input.Engine, Image: template.Image, ImageVersion: input.ImageVersion, Name: input.Name, DatabaseName: databaseName, DatabaseUser: databaseUser, Status: "draft", InternalHost: template.InternalHost(id), Port: template.Port, ConnectionURL: template.ConnectionURL(id, input.Environment), Environment: input.Environment}
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
		template, _ := catalog.Get(service.Engine)
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
		} else if err := h.db.QueryRowContext(r.Context(), "SELECT s.id FROM servers s WHERE "+serverStatusSQL+" = 'online' ORDER BY s.created_at LIMIT 1").Scan(&nullableServerID); errors.Is(err, sql.ErrNoRows) {
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
	if !h.requireServerOnline(w, r, nullableServerID.String) {
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
	if err := h.queueServiceAction(r, serverID, serviceID, action, imageVersion); err != nil {
		writeError(w, http.StatusInternalServerError, "could not queue service action")
		return
	}
	w.WriteHeader(http.StatusAccepted)
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
	template, _ := catalog.Get(service.Engine)
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
	service.Image = template.Image
	service.InternalHost, service.Port = template.InternalHost(serviceID), template.Port
	writeJSON(w, http.StatusOK, service)
}

func (h *Handler) readService(r *http.Request, workspaceID, projectID string, serviceID string, includeSecret bool) (databaseService, error) {
	var service databaseService
	var encrypted []byte
	err := h.db.QueryRowContext(r.Context(), `SELECT d.id, d.workspace_id, d.project_id, d.engine, d.name, d.image_version, COALESCE(d.server_id, ''), COALESCE(v.name, ''), d.database_name, d.database_user, d.status, COALESCE(d.last_error, ''), d.encrypted_config FROM database_services d LEFT JOIN servers v ON v.id = d.server_id WHERE d.id = ? AND d.workspace_id = ? AND d.project_id = ?`, serviceID, workspaceID, projectID).Scan(&service.ID, &service.WorkspaceID, &service.ProjectID, &service.Engine, &service.Name, &service.ImageVersion, &service.ServerID, &service.ServerName, &service.DatabaseName, &service.DatabaseUser, &service.Status, &service.LastError, &encrypted)
	if err != nil {
		return service, err
	}
	template, _ := catalog.Get(service.Engine)
	service.Image = template.Image
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
