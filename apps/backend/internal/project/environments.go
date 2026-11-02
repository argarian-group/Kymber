package project

import (
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"kymber/backend/internal/database"
)

type projectEnvironment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func environmentName(raw string) string {
	name := strings.TrimSpace(raw)
	if utf8.RuneCountInString(name) > 64 || strings.ContainsFunc(name, unicode.IsControl) {
		return ""
	}
	return name
}

func (h *Handler) listEnvironments(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, false)
	if !ok {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT id, name FROM project_environments WHERE project_id = ? ORDER BY created_at, id", projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list environments")
		return
	}
	defer rows.Close()
	environments := make([]projectEnvironment, 0)
	for rows.Next() {
		var environment projectEnvironment
		if err := rows.Scan(&environment.ID, &environment.Name); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read environments")
			return
		}
		environments = append(environments, environment)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read environments")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]projectEnvironment{"environments": environments})
}

func (h *Handler) createEnvironment(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, true)
	if !ok {
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	name := environmentName(input.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "environment name must be 1 to 64 characters")
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate environment id")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO project_environments (id, project_id, name) VALUES (?, ?, ?)", id, projectID, name); err != nil {
		writeError(w, http.StatusConflict, "could not create environment; its name may already be used")
		return
	}
	writeJSON(w, http.StatusCreated, projectEnvironment{ID: id, Name: name})
}

func (h *Handler) deleteEnvironment(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, true)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "environmentID")
	if !ok {
		return
	}
	result, err := h.db.ExecContext(r.Context(), "DELETE FROM project_environments WHERE id = ? AND project_id = ? AND NOT EXISTS (SELECT 1 FROM database_services d WHERE d.environment_id = ? AND d.status <> 'deleted')", id, projectID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete environment")
		return
	}
	if deleted, _ := result.RowsAffected(); deleted == 0 {
		writeError(w, http.StatusConflict, "environment not found or still has apps")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
