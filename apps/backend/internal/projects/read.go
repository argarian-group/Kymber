package projects

import (
	"database/sql"
	"errors"
	"net/http"

	"kymber/backend/internal/auth"
)

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, false) {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT id, workspace_id, name, slug, created_at FROM projects WHERE workspace_id = ? ORDER BY id", workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list projects")
		return
	}
	defer rows.Close()
	projects := make([]Project, 0)
	for rows.Next() {
		var project Project
		if err := rows.Scan(&project.ID, &project.WorkspaceID, &project.Name, &project.Slug, &project.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read projects")
			return
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read projects")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]Project{"projects": projects})
}

func (h *Handler) getProject(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, false) {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok {
		return
	}
	project, err := h.findProject(r, workspaceID, projectID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read project")
		return
	}
	writeJSON(w, http.StatusOK, map[string]Project{"project": project})
}

func (h *Handler) findProject(r *http.Request, workspaceID, projectID string) (Project, error) {
	var project Project
	err := h.db.QueryRowContext(r.Context(), "SELECT id, workspace_id, name, slug, created_at FROM projects WHERE id = ? AND workspace_id = ?", projectID, workspaceID).Scan(&project.ID, &project.WorkspaceID, &project.Name, &project.Slug, &project.CreatedAt)
	return project, err
}
