package services

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"kymber/backend/internal/auth"
)

type Handler struct{ db *sql.DB }

const serverStatusSQL = `CASE
	WHEN s.connection_type = 'ssh' THEN 'online'
	WHEN s.connection_type = 'local' AND s.encrypted_ssh_key IS NOT NULL THEN 'online'
	WHEN s.last_seen IS NULL THEN 'pending'
	WHEN s.last_seen >= UTC_TIMESTAMP(6) - INTERVAL 30 SECOND OR EXISTS(SELECT 1 FROM server_tasks t WHERE t.server_id = s.id AND t.status = 'running') THEN 'online'
	ELSE 'offline'
END`

func New(db *sql.DB) *Handler { return &Handler{db: db} }
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	go h.resumeDeployWatches()
	for _, route := range []struct {
		pattern string
		handler http.HandlerFunc
	}{
		{"GET /api/database-templates", h.listDatabaseTemplates},
		{"GET /api/docker-hub/search", h.searchDockerHub},
		{"GET /api/docker-hub/image", h.dockerImageDetails},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/services", h.listServices},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/services", h.createService},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}", h.getService},
		{"PUT /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}", h.updateService},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/{action}", h.serviceAction},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/logs", h.serviceLogs},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/ports", h.listPorts},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/ports", h.createPort},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/ports/{portID}", h.deletePort},
		{"GET /api/servers/{serverID}/terminal", h.serverTerminal},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/terminal", h.serviceTerminal},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/files", h.listFiles},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/files", h.createFile},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/files", h.deleteFile},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/files/content", h.readFile},
		{"PUT /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/files/content", h.writeFile},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/files/move", h.moveFile},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/storages", h.listStorages},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/storages", h.createStorage},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/storages/{storageID}", h.getStorageContent},
		{"PUT /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/storages/{storageID}", h.updateStorageContent},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/storages/{storageID}", h.deleteStorage},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}", h.deleteService},
	} {
		mux.Handle(route.pattern, auth.Authenticate(h.db, route.handler))
	}
}

var entityIDPattern = regexp.MustCompile(`^[0-9a-f]{6}$`)

func pathID(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	id := r.PathValue(name)
	if !entityIDPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return "", false
	}
	return id, true
}
func decodeJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request must contain a single JSON value")
		return false
	}
	return true
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, user auth.User, workspaceID string, write bool) bool {
	if user.IsRoot {
		var exists bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM workspaces WHERE id = ?)", workspaceID).Scan(&exists); err != nil {
			writeError(w, http.StatusInternalServerError, "could not verify workspace")
			return false
		}
		if !exists {
			writeError(w, http.StatusNotFound, "workspace not found")
			return false
		}
		return true
	}
	var role string
	err := h.db.QueryRowContext(r.Context(), "SELECT role FROM workspace_members WHERE workspace_id = ? AND user_id = ?", workspaceID, user.ID).Scan(&role)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "workspace not found")
		return false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify workspace access")
		return false
	}
	if write && role != "owner" && role != "admin" {
		writeError(w, http.StatusForbidden, "workspace write access required")
		return false
	}
	return true
}
func (h *Handler) requireWorkspaceRole(w http.ResponseWriter, r *http.Request, user auth.User, workspaceID string, allowAdmin bool) (string, bool) {
	if !h.authorize(w, r, user, workspaceID, false) {
		return "", false
	}
	if user.IsRoot {
		return "root", true
	}
	var role string
	if err := h.db.QueryRowContext(r.Context(), "SELECT role FROM workspace_members WHERE workspace_id = ? AND user_id = ?", workspaceID, user.ID).Scan(&role); err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify workspace ownership")
		return "", false
	}
	if role != "owner" && !(allowAdmin && role == "admin") {
		msg := "workspace owner access required"
		if allowAdmin {
			msg = "workspace owner or admin access required"
		}
		writeError(w, http.StatusForbidden, msg)
		return "", false
	}
	return role, true
}
func (h *Handler) requireServerOnline(w http.ResponseWriter, r *http.Request, id string) bool {
	var online bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM servers s WHERE s.id = ? AND (s.connection_type = 'ssh' OR "+serverStatusSQL+" = 'online'))", id).Scan(&online); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check server")
		return false
	}
	if !online {
		writeError(w, http.StatusConflict, "server is offline")
		return false
	}
	return true
}
