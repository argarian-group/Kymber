package agent

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"kymber/backend/internal/secrets"
	"kymber/backend/internal/templates"
)

type Handler struct{ db *sql.DB }

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

func New(db *sql.DB) *Handler { return &Handler{db: db} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/agent/heartbeat", h.heartbeat)
	mux.HandleFunc("POST /api/agent/tasks/claim", h.claimTask)
	mux.HandleFunc("POST /api/agent/tasks/{taskID}/result", h.completeTask)
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

func (h *Handler) claimTask(w http.ResponseWriter, r *http.Request) {
	serverID, ok := h.authenticate(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), `UPDATE database_services d JOIN server_tasks q ON q.service_id = d.id
		SET d.status = 'failed', d.last_error = 'Server task lease expired too many times.'
		WHERE q.server_id = ? AND q.status = 'running' AND q.attempts >= 5 AND (q.lease_expires_at IS NULL OR q.lease_expires_at < UTC_TIMESTAMP(6))
		AND NOT EXISTS (SELECT 1 FROM server_tasks newer WHERE newer.service_id = q.service_id AND newer.id <> q.id AND newer.created_at >= q.created_at)`, serverID)
	if err == nil {
		_, err = tx.ExecContext(r.Context(), `UPDATE server_tasks SET status = 'failed', result_message = 'Server task lease expired too many times.', completed_at = UTC_TIMESTAMP(6), lease_expires_at = NULL
			WHERE server_id = ? AND status = 'running' AND attempts >= 5 AND (lease_expires_at IS NULL OR lease_expires_at < UTC_TIMESTAMP(6))`, serverID)
	}
	if err == nil {
		_, err = tx.ExecContext(r.Context(), "UPDATE server_tasks SET status = 'queued', claimed_at = NULL, lease_expires_at = NULL WHERE server_id = ? AND status = 'running' AND attempts < 5 AND (lease_expires_at IS NULL OR lease_expires_at < UTC_TIMESTAMP(6))", serverID)
	}
	if err != nil {
		http.Error(w, "could not recover expired task", http.StatusInternalServerError)
		return
	}
	var t task
	var encrypted []byte
	err = tx.QueryRowContext(r.Context(), `SELECT q.id, q.service_id, q.action, d.workspace_id, d.engine, d.name, d.image_version, d.image, d.port, d.data_path, COALESCE(d.environment_id, ''), d.database_name, d.database_user, d.encrypted_config
		FROM server_tasks q JOIN database_services d ON d.id = q.service_id
		WHERE q.server_id = ? AND q.status = 'queued' AND d.category <> 'application' AND d.compose_file IS NULL
		ORDER BY q.created_at LIMIT 1 FOR UPDATE SKIP LOCKED`, serverID).
		Scan(&t.ID, &t.ServiceID, &t.Action, &t.WorkspaceID, &t.Engine, &t.Name, &t.ImageVersion, &t.Image, &t.Port, &t.DataPath, &t.Environment, &t.Database, &t.Username, &encrypted)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.Commit(); err != nil {
			http.Error(w, "could not recover expired task", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		http.Error(w, "could not read task", http.StatusInternalServerError)
		return
	}
	plain, err := secrets.Decrypt(encrypted)
	if err != nil {
		if err := h.failUndeliverableTask(r, tx, t.ID); err != nil {
			http.Error(w, "could not decrypt task configuration", http.StatusInternalServerError)
			return
		}
		http.Error(w, "could not decrypt task configuration", http.StatusInternalServerError)
		return
	}
	if err := json.Unmarshal(plain, &t.Config); err != nil {
		if err := h.failUndeliverableTask(r, tx, t.ID); err != nil {
			http.Error(w, "could not decode task configuration", http.StatusInternalServerError)
			return
		}
		http.Error(w, "could not decode task configuration", http.StatusInternalServerError)
		return
	}
	if _, err := tx.ExecContext(r.Context(), "UPDATE server_tasks SET status = 'running', attempts = attempts + 1, claimed_at = UTC_TIMESTAMP(6), lease_expires_at = UTC_TIMESTAMP(6) + INTERVAL 30 MINUTE WHERE id = ?", t.ID); err != nil {
		http.Error(w, "could not claim task", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "could not claim task", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(t)
}

func (h *Handler) failUndeliverableTask(r *http.Request, tx *sql.Tx, taskID string) error {
	if _, err := tx.ExecContext(r.Context(), "UPDATE server_tasks SET status = 'failed', result_message = 'Task configuration could not be decoded.', completed_at = UTC_TIMESTAMP(6) WHERE id = ?", taskID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(r.Context(), `UPDATE database_services d JOIN server_tasks q ON q.service_id = d.id
		SET d.status = 'failed', d.last_error = 'Task configuration could not be decoded.' WHERE q.id = ?
		AND NOT EXISTS (SELECT 1 FROM server_tasks newer WHERE newer.service_id = q.service_id AND newer.id <> q.id AND newer.created_at >= q.created_at)`, taskID); err != nil {
		return err
	}
	return tx.Commit()
}

func (h *Handler) completeTask(w http.ResponseWriter, r *http.Request) {
	serverID, ok := h.authenticate(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var result result
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8192))
	if err != nil || json.Unmarshal(body, &result) != nil {
		http.Error(w, "invalid task result", http.StatusBadRequest)
		return
	}
	if len(result.Message) > 2048 || len(result.ContainerID) > 128 {
		http.Error(w, "task result is too large", http.StatusBadRequest)
		return
	}
	status, serviceStatus := "succeeded", "running"
	if !result.Success {
		status, serviceStatus = "failed", "failed"
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	var serviceID, action string
	if err := tx.QueryRowContext(r.Context(), "SELECT service_id, action FROM server_tasks WHERE id = ? AND server_id = ? AND status = 'running' AND lease_expires_at > UTC_TIMESTAMP(6) FOR UPDATE", r.PathValue("taskID"), serverID).Scan(&serviceID, &action); err != nil {
		http.Error(w, "task is not claimed by this server", http.StatusConflict)
		return
	}
	if result.Success {
		switch action {
		case "stop":
			serviceStatus = "stopped"
		case "delete":
			serviceStatus = "deleted"
		case "remove_container", "remove_container_image":
			serviceStatus = "stopped"
		}
	}
	if _, err := tx.ExecContext(r.Context(), "UPDATE server_tasks SET status = ?, result_message = ?, completed_at = UTC_TIMESTAMP(6), lease_expires_at = NULL WHERE id = ?", status, result.Message, r.PathValue("taskID")); err != nil {
		http.Error(w, "could not save task result", http.StatusInternalServerError)
		return
	}
	if _, err := tx.ExecContext(r.Context(), `UPDATE database_services SET status = ?, last_error = ? WHERE id = ?
		AND NOT EXISTS (SELECT 1 FROM server_tasks newer JOIN server_tasks current ON current.service_id = newer.service_id
			WHERE current.id = ? AND newer.id <> current.id AND newer.created_at >= current.created_at)`, serviceStatus, nullableMessage(result.Message, result.Success), serviceID, r.PathValue("taskID")); err != nil {
		http.Error(w, "could not update service", http.StatusInternalServerError)
		return
	}
	if result.Success && action == "deploy" {
		if _, err := tx.ExecContext(r.Context(), `UPDATE service_storages SET applied = TRUE WHERE service_id = ?
			AND NOT EXISTS (SELECT 1 FROM server_tasks newer JOIN server_tasks current ON current.service_id = newer.service_id
				WHERE current.id = ? AND newer.id <> current.id AND newer.created_at >= current.created_at)`, serviceID, r.PathValue("taskID")); err != nil {
			http.Error(w, "could not update service storage", http.StatusInternalServerError)
			return
		}
		if _, err := tx.ExecContext(r.Context(), `UPDATE service_ports SET applied = TRUE WHERE service_id = ?
			AND NOT EXISTS (SELECT 1 FROM server_tasks newer JOIN server_tasks current ON current.service_id = newer.service_id
				WHERE current.id = ? AND newer.id <> current.id AND newer.created_at >= current.created_at)`, serviceID, r.PathValue("taskID")); err != nil {
			http.Error(w, "could not update service ports", http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "could not save task result", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func nullableMessage(message string, success bool) any {
	if success {
		return nil
	}
	return message
}
