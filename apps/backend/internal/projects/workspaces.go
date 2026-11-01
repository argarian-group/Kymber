package projects

import (
	"database/sql"
	"errors"
	"net/http"

	"kymber/backend/internal/auth"
)

func (h *Handler) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	query := `SELECT w.id, w.name, w.created_at, m.role FROM workspaces w JOIN workspace_members m ON m.workspace_id = w.id WHERE m.user_id = ? ORDER BY w.id`
	args := []any{user.ID}
	if user.IsRoot {
		query = `SELECT w.id, w.name, w.created_at, COALESCE(m.role, 'root') FROM workspaces w LEFT JOIN workspace_members m ON m.workspace_id = w.id AND m.user_id = ? ORDER BY w.id`
		args = []any{user.ID}
	}
	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list workspaces")
		return
	}
	defer rows.Close()
	workspaces := make([]Workspace, 0)
	for rows.Next() {
		var workspace Workspace
		if err := rows.Scan(&workspace.ID, &workspace.Name, &workspace.CreatedAt, &workspace.Role); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read workspaces")
			return
		}
		workspaces = append(workspaces, workspace)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read workspaces")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]Workspace{"workspaces": workspaces})
}

func (h *Handler) getWorkspace(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, false) {
		return
	}
	var workspace Workspace
	err := h.db.QueryRowContext(r.Context(), `SELECT w.id, w.name, w.created_at, COALESCE(m.role, 'root') FROM workspaces w LEFT JOIN workspace_members m ON m.workspace_id = w.id AND m.user_id = ? WHERE w.id = ?`, user.ID, workspaceID).Scan(&workspace.ID, &workspace.Name, &workspace.CreatedAt, &workspace.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read workspace")
		return
	}
	writeJSON(w, http.StatusOK, map[string]Workspace{"workspace": workspace})
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, user auth.User, workspaceID uint64, write bool) bool {
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
	if errors.Is(err, sql.ErrNoRows) {
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
