package tasks

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
)

type TaskLabel struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type taskLabelPayload struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func validLabelColor(color string) bool {
	switch color {
	case "red", "orange", "yellow", "green", "teal", "blue", "purple", "gray":
		return true
	}
	return false
}

func (h *Handler) loadLabels(ctx context.Context, projectID string) ([]TaskLabel, error) {
	rows, err := h.db.QueryContext(ctx, "SELECT id, name, color FROM project_labels WHERE project_id = ? ORDER BY name, id", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	labels := make([]TaskLabel, 0)
	for rows.Next() {
		var label TaskLabel
		if err := rows.Scan(&label.ID, &label.Name, &label.Color); err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}
	return labels, rows.Err()
}

func taskLabelInput(w http.ResponseWriter, r *http.Request) (taskLabelPayload, bool) {
	var input taskLabelPayload
	if !decodeJSON(w, r, &input) {
		return input, false
	}
	input.Name = strings.TrimSpace(input.Name)
	if length := utf8.RuneCountInString(input.Name); length < 1 || length > 40 {
		writeError(w, http.StatusBadRequest, "name must be 1 to 40 characters")
		return input, false
	}
	if !validLabelColor(input.Color) {
		writeError(w, http.StatusBadRequest, "unknown label color")
		return input, false
	}
	return input, true
}

func (h *Handler) createTaskLabel(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, "")
	if !ok {
		return
	}
	input, ok := taskLabelInput(w, r)
	if !ok {
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate label id")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO project_labels (id, project_id, name, color) VALUES (?, ?, ?, ?)", id, projectID, input.Name, input.Color); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create label")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]TaskLabel{"label": {ID: id, Name: input.Name, Color: input.Color}})
}

func (h *Handler) authorizeTaskLabel(w http.ResponseWriter, r *http.Request, permission string) (string, string, bool) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, permission)
	if !ok {
		return "", "", false
	}
	labelID, ok := pathID(w, r, "labelID")
	if !ok || !h.scopeExists(w, r, "SELECT EXISTS(SELECT 1 FROM project_labels WHERE id = ? AND project_id = ?)", "label not found", labelID, projectID) {
		return "", "", false
	}
	return projectID, labelID, true
}

func (h *Handler) updateTaskLabel(w http.ResponseWriter, r *http.Request) {
	_, labelID, ok := h.authorizeTaskLabel(w, r, "")
	if !ok {
		return
	}
	input, ok := taskLabelInput(w, r)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE project_labels SET name = ?, color = ? WHERE id = ?", input.Name, input.Color, labelID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update label")
		return
	}
	writeJSON(w, http.StatusOK, map[string]TaskLabel{"label": {ID: labelID, Name: input.Name, Color: input.Color}})
}

func (h *Handler) deleteTaskLabel(w http.ResponseWriter, r *http.Request) {
	_, labelID, ok := h.authorizeTaskLabel(w, r, permissions.TasksDelete)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "DELETE FROM project_labels WHERE id = ?", labelID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete label")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
