package project

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-sql-driver/mysql"
	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
)

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, true) {
		return
	}
	input, ok := projectInput(w, r)
	if !ok {
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate project id")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create project")
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), "INSERT INTO projects (id, workspace_id, name, slug) VALUES (?, ?, ?, ?)", id, workspaceID, input.Name, input.Slug)
	if err != nil {
		writeProjectError(w, err)
		return
	}
	if err := createDefaultTaskLists(r.Context(), tx, id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create task lists")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create project")
		return
	}
	project, err := h.findProject(r, workspaceID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read created project")
		return
	}
	w.Header().Set("Location", "/api/workspaces/"+workspaceID+"/projects/"+project.ID)
	writeJSON(w, http.StatusCreated, map[string]Project{"project": project})
}

func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, true) {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok {
		return
	}
	input, ok := projectInput(w, r)
	if !ok {
		return
	}
	result, err := h.db.ExecContext(r.Context(), "UPDATE projects SET name = ?, slug = ? WHERE id = ? AND workspace_id = ?", input.Name, input.Slug, projectID, workspaceID)
	if err != nil {
		writeProjectError(w, err)
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update project")
		return
	}
	if changed == 0 {
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
		return
	}
	project, err := h.findProject(r, workspaceID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read updated project")
		return
	}
	writeJSON(w, http.StatusOK, map[string]Project{"project": project})
}

func (h *Handler) deleteProject(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, true) {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok {
		return
	}
	result, err := h.db.ExecContext(r.Context(), "DELETE FROM projects WHERE id = ? AND workspace_id = ?", projectID, workspaceID)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1451 {
			writeError(w, http.StatusConflict, "project has dependent resources")
		} else {
			writeError(w, http.StatusInternalServerError, "could not delete project")
		}
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete project")
		return
	}
	if deleted == 0 {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeProjectError(w http.ResponseWriter, err error) {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		writeError(w, http.StatusConflict, "project slug already exists in workspace")
		return
	}
	writeError(w, http.StatusInternalServerError, "could not save project")
}
