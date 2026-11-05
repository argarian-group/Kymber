package services

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"kymber/backend/internal/auth"
)

type deploymentRecord struct {
	ID           string     `json:"id"`
	Action       string     `json:"action"`
	ImageVersion string     `json:"image_version"`
	Status       string     `json:"status"`
	LastError    string     `json:"last_error"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
}

type deploymentDetail struct {
	deploymentRecord
	Output string `json:"output"`
}

func (h *Handler) listDeployments(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, "") {
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
	rows, err := h.db.QueryContext(r.Context(), `SELECT d.id, d.action, d.image_version, d.status, COALESCE(d.last_error, ''), d.started_at, d.finished_at
		FROM service_deployments d JOIN database_services s ON s.id = d.service_id
		WHERE d.service_id = ? AND s.workspace_id = ? AND s.project_id = ?
		ORDER BY d.started_at DESC LIMIT 50`, serviceID, workspaceID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list deployments")
		return
	}
	defer rows.Close()
	deployments := make([]deploymentRecord, 0)
	for rows.Next() {
		var deployment deploymentRecord
		if err := rows.Scan(&deployment.ID, &deployment.Action, &deployment.ImageVersion, &deployment.Status, &deployment.LastError, &deployment.StartedAt, &deployment.FinishedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read deployments")
			return
		}
		deployments = append(deployments, deployment)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read deployments")
		return
	}
	if err := rows.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not close deployments")
		return
	}
	if len(deployments) == 0 {
		var exists bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM database_services WHERE id = ? AND workspace_id = ? AND project_id = ?)", serviceID, workspaceID, projectID).Scan(&exists); err != nil {
			writeError(w, http.StatusInternalServerError, "could not check service")
			return
		}
		if !exists {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"deployments": deployments})
}

func (h *Handler) getDeployment(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, "") {
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
	deploymentID, ok := pathID(w, r, "deploymentID")
	if !ok {
		return
	}
	var deployment deploymentDetail
	err := h.db.QueryRowContext(r.Context(), `SELECT d.id, d.action, d.image_version, d.status, COALESCE(d.last_error, ''), d.started_at, d.finished_at, d.output
		FROM service_deployments d JOIN database_services s ON s.id = d.service_id
		WHERE d.id = ? AND d.service_id = ? AND s.workspace_id = ? AND s.project_id = ?`, deploymentID, serviceID, workspaceID, projectID).
		Scan(&deployment.ID, &deployment.Action, &deployment.ImageVersion, &deployment.Status, &deployment.LastError, &deployment.StartedAt, &deployment.FinishedAt, &deployment.Output)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "deployment not found")
		} else {
			writeError(w, http.StatusInternalServerError, "could not read deployment")
		}
		return
	}
	writeJSON(w, http.StatusOK, deployment)
}
