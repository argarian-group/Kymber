package workspaces

import (
	"errors"
	"net/http"

	"github.com/go-sql-driver/mysql"
	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
)

func (h *Handler) createWorkspace(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	if !user.IsRoot {
		writeError(w, http.StatusForbidden, "only root can create workspaces")
		return
	}
	input, ok := workspaceInput(w, r)
	if !ok {
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate workspace id")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start workspace creation")
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(), "INSERT INTO workspaces (id, name, created_by) VALUES (?, ?, ?)", id, input.Name, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create workspace")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "INSERT INTO workspace_members (workspace_id, user_id, role) VALUES (?, ?, 'owner')", id, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not assign workspace owner")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not complete workspace creation")
		return
	}
	workspace, err := scanWorkspace(h.db.QueryRowContext(r.Context(), `SELECT w.id, w.name, w.created_at, 'owner', NULL, u.id, u.username, u.email FROM workspaces w LEFT JOIN users u ON u.id = w.created_by WHERE w.id = ?`, id))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read created workspace")
		return
	}
	w.Header().Set("Location", "/api/workspaces/"+id)
	writeJSON(w, http.StatusCreated, map[string]Workspace{"workspace": workspace})
}

func (h *Handler) updateWorkspace(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.WorkspaceManage) {
		return
	}
	input, ok := workspaceInput(w, r)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE workspaces SET name = ? WHERE id = ?", input.Name, workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update workspace")
		return
	}
	workspace, err := scanWorkspace(h.db.QueryRowContext(r.Context(), `SELECT w.id, w.name, w.created_at, COALESCE(m.role, 'root'), rp.permissions, c.id, c.username, c.email FROM workspaces w LEFT JOIN workspace_members m ON m.workspace_id = w.id AND m.user_id = ? LEFT JOIN workspace_roles rp ON rp.workspace_id = w.id AND rp.name = m.role LEFT JOIN users c ON c.id = w.created_by WHERE w.id = ?`, user.ID, workspaceID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read updated workspace")
		return
	}
	writeJSON(w, http.StatusOK, map[string]Workspace{"workspace": workspace})
}

func (h *Handler) deleteWorkspace(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, "") {
		return
	}
	if !user.IsRoot {
		var creatorID uint64
		if err := h.db.QueryRowContext(r.Context(), "SELECT COALESCE(created_by, 0) FROM workspaces WHERE id = ?", workspaceID).Scan(&creatorID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not verify workspace creator")
			return
		}
		if creatorID != user.ID {
			writeError(w, http.StatusForbidden, "only the original owner can delete the workspace")
			return
		}
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start workspace deletion")
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(), "DELETE FROM workspace_members WHERE workspace_id = ?", workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not remove workspace members")
		return
	}
	result, err := tx.ExecContext(r.Context(), "DELETE FROM workspaces WHERE id = ?", workspaceID)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1451 {
			writeError(w, http.StatusConflict, "workspace still contains projects")
		} else {
			writeError(w, http.StatusInternalServerError, "could not delete workspace")
		}
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete workspace")
		return
	}
	if deleted == 0 {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not complete workspace deletion")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
