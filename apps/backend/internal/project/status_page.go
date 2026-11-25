package project

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/permissions"
)

func (h *Handler) setStatusPage(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.ProjectsManage) {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok {
		return
	}
	var input struct {
		Enabled bool `json:"enabled"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	var token any
	if input.Enabled {
		secret := make([]byte, 16)
		if _, err := rand.Read(secret); err != nil {
			writeError(w, http.StatusInternalServerError, "could not generate status page link")
			return
		}
		token = base64.RawURLEncoding.EncodeToString(secret)
	}
	result, err := h.db.ExecContext(r.Context(), "UPDATE projects SET status_page_token = ? WHERE id = ? AND workspace_id = ?", token, projectID, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update status page")
		return
	}
	if changed, _ := result.RowsAffected(); changed == 0 && input.Enabled {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status_page_token": token})
}
