package project

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/permissions"
)

// setGitHubWebhook creates or removes the secret GitHub signs deliveries with; the secret is only shown in this response.
func (h *Handler) setGitHubWebhook(w http.ResponseWriter, r *http.Request) {
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
	var secret any
	if input.Enabled {
		raw := make([]byte, 24)
		if _, err := rand.Read(raw); err != nil {
			writeError(w, http.StatusInternalServerError, "could not generate webhook secret")
			return
		}
		secret = hex.EncodeToString(raw)
	}
	result, err := h.db.ExecContext(r.Context(), "UPDATE projects SET github_webhook_secret = ? WHERE id = ? AND workspace_id = ?", secret, projectID, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update webhook")
		return
	}
	if changed, _ := result.RowsAffected(); changed == 0 && input.Enabled {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"secret": secret})
}
