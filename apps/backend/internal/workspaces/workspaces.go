package workspaces

import (
	"database/sql"
	"errors"
	"net/http"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/permissions"
)

type workspaceScanner interface {
	Scan(...any) error
}

func scanWorkspace(row workspaceScanner) (Workspace, error) {
	var workspace Workspace
	var creatorID sql.NullInt64
	var creatorUsername, creatorEmail, custom sql.NullString
	err := row.Scan(&workspace.ID, &workspace.Name, &workspace.CreatedAt, &workspace.Role, &custom, &creatorID, &creatorUsername, &creatorEmail)
	workspace.Permissions = permissions.ForRole(workspace.Role, custom).List()
	if creatorID.Valid {
		workspace.CreatedBy = &WorkspaceCreator{ID: uint64(creatorID.Int64), Username: creatorUsername.String, Email: creatorEmail.String}
	}
	return workspace, err
}

func (h *Handler) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	query := `SELECT w.id, w.name, w.created_at, m.role, rp.permissions, c.id, c.username, c.email FROM workspaces w JOIN workspace_members m ON m.workspace_id = w.id LEFT JOIN workspace_roles rp ON rp.workspace_id = w.id AND rp.name = m.role LEFT JOIN users c ON c.id = w.created_by WHERE m.user_id = ? ORDER BY w.id`
	args := []any{user.ID}
	if user.IsRoot {
		query = `SELECT w.id, w.name, w.created_at, COALESCE(m.role, 'root'), rp.permissions, c.id, c.username, c.email FROM workspaces w LEFT JOIN workspace_members m ON m.workspace_id = w.id AND m.user_id = ? LEFT JOIN workspace_roles rp ON rp.workspace_id = w.id AND rp.name = m.role LEFT JOIN users c ON c.id = w.created_by ORDER BY w.id`
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
		workspace, err := scanWorkspace(rows)
		if err != nil {
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
	if !ok || !h.authorize(w, r, user, workspaceID, "") {
		return
	}
	workspace, err := scanWorkspace(h.db.QueryRowContext(r.Context(), `SELECT w.id, w.name, w.created_at, COALESCE(m.role, 'root'), rp.permissions, c.id, c.username, c.email FROM workspaces w LEFT JOIN workspace_members m ON m.workspace_id = w.id AND m.user_id = ? LEFT JOIN workspace_roles rp ON rp.workspace_id = w.id AND rp.name = m.role LEFT JOIN users c ON c.id = w.created_by WHERE w.id = ?`, user.ID, workspaceID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read workspace")
		return
	}
	writeJSON(w, http.StatusOK, map[string]Workspace{"workspace": workspace})
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, user auth.User, workspaceID string, permission string) bool {
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
	_, granted, err := permissions.Lookup(r.Context(), h.db, workspaceID, user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "workspace not found")
		return false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify workspace access")
		return false
	}
	if permission != "" && !granted.Has(permission) {
		writeError(w, http.StatusForbidden, "you do not have permission for this action")
		return false
	}
	return true
}
