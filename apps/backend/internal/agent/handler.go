package agent

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"kymber/backend/internal/secrets"
	"kymber/backend/internal/templates"
)

type Handler struct{ db *sql.DB }

var taskIDPattern = regexp.MustCompile(`^[0-9a-f]{6}$`)

func New(db *sql.DB) *Handler { return &Handler{db: db} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/agent/heartbeat", h.heartbeat)
	mux.HandleFunc("POST /api/agent/tasks/claim", h.claim)
	mux.HandleFunc("POST /api/agent/tasks/{taskID}/result", h.result)
}

func (h *Handler) authenticate(r *http.Request) (string, bool) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		return "", false
	}
	digest := sha256.Sum256([]byte(token))
	var id string
	err := h.db.QueryRowContext(r.Context(), "SELECT id FROM servers WHERE token_hash = ?", digest[:]).Scan(&id)
	return id, err == nil
}

func (h *Handler) heartbeat(w http.ResponseWriter, r *http.Request) {
	id, ok := h.authenticate(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE servers SET status = 'online', last_seen = UTC_TIMESTAMP(6) WHERE id = ?", id); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) claim(w http.ResponseWriter, r *http.Request) {
	id, ok := h.authenticate(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE servers SET status = 'online', last_seen = UTC_TIMESTAMP(6) WHERE id = ?", id); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	var taskID, serviceID string
	var action, engine, name, imageVersion, image, dataPath, dbName, dbUser string
	var port int
	var workspaceID string
	var encryptedConfig []byte
	err = tx.QueryRowContext(r.Context(), `SELECT t.id, t.service_id, t.action, s.workspace_id, s.engine, s.name, s.image_version, s.image, s.port, s.data_path, s.database_name, s.database_user, s.encrypted_config FROM server_tasks t JOIN database_services s ON s.id = t.service_id WHERE t.server_id = ? AND t.status = 'queued' ORDER BY t.created_at LIMIT 1 FOR UPDATE`, id).Scan(&taskID, &serviceID, &action, &workspaceID, &engine, &name, &imageVersion, &image, &port, &dataPath, &dbName, &dbUser, &encryptedConfig)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if _, err := tx.ExecContext(r.Context(), "UPDATE server_tasks SET status = 'running', attempts = attempts + 1, claimed_at = UTC_TIMESTAMP(6) WHERE id = ?", taskID); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if _, err := tx.ExecContext(r.Context(), "UPDATE database_services SET status = 'deploying', last_error = NULL WHERE id = ?", serviceID); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	plain, err := secrets.Decrypt(encryptedConfig)
	if err != nil {
		http.Error(w, "service secret unavailable", http.StatusInternalServerError)
		return
	}
	var config templates.Config
	if err := json.Unmarshal(plain, &config); err != nil {
		http.Error(w, "service config unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": taskID, "service_id": serviceID, "workspace_id": workspaceID, "action": action, "engine": engine, "name": name, "image_version": imageVersion, "image": image, "port": port, "data_path": dataPath, "database_name": dbName, "database_user": dbUser, "config": config})
}

func (h *Handler) result(w http.ResponseWriter, r *http.Request) {
	serverID, ok := h.authenticate(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	taskID := r.PathValue("taskID")
	if !taskIDPattern.MatchString(taskID) {
		http.Error(w, "invalid task", http.StatusBadRequest)
		return
	}
	var input struct {
		Success     bool   `json:"success"`
		Message     string `json:"message"`
		ContainerID string `json:"container_id"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || len(input.Message) > 2048 || len(input.ContainerID) > 128 {
		http.Error(w, "invalid result", http.StatusBadRequest)
		return
	}
	status, serviceStatus := "failed", "failed"
	var action, taskStatus string
	if err := h.db.QueryRowContext(r.Context(), "SELECT action, status FROM server_tasks WHERE id = ? AND server_id = ?", taskID, serverID).Scan(&action, &taskStatus); err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if taskStatus != "running" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if input.Success {
		status, serviceStatus = "completed", "running"
		if action == "stop" {
			serviceStatus = "stopped"
		}
		if action == "delete" {
			serviceStatus = "deleted"
		}
		if action == "remove_container" || action == "remove_container_image" {
			serviceStatus = "stopped"
		}
	}
	if _, err := h.db.ExecContext(r.Context(), `UPDATE server_tasks SET status = ?, result_message = ?, completed_at = UTC_TIMESTAMP(6) WHERE id = ? AND server_id = ? AND status = 'running'`, status, input.Message, taskID, serverID); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	containerID := input.ContainerID
	if input.Success && (action == "delete" || action == "remove_container" || action == "remove_container_image") {
		containerID = ""
	}
	_, err := h.db.ExecContext(r.Context(), `UPDATE database_services s JOIN server_tasks t ON t.service_id = s.id SET s.status = ?, s.last_error = ?, s.container_id = NULLIF(?, '') WHERE t.id = ? AND t.server_id = ?`, serviceStatus, nullableError(input), containerID, taskID, serverID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func nullableError(input struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ContainerID string `json:"container_id"`
}) any {
	if input.Success || input.Message == "" {
		return nil
	}
	return input.Message
}
