package tasks

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"unicode/utf8"

	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
)

var defaultTaskLists = []struct {
	name   string
	isDone bool
}{
	{"To do", false},
	{"In progress", false},
	{"In review", false},
	{"Done", true},
}

type TaskList struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsDone   bool   `json:"is_done"`
	Position int    `json:"position"`
}

type taskListPayload struct {
	Name   string `json:"name"`
	IsDone bool   `json:"is_done"`
}

type taskListOrderPayload struct {
	IDs []string `json:"ids"`
}

func sameIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, id := range a {
		counts[id]++
	}
	for _, id := range b {
		if counts[id] == 0 {
			return false
		}
		counts[id]--
	}
	return true
}

func createDefaultTaskLists(ctx context.Context, tx *sql.Tx, projectID string) error {
	for index, list := range defaultTaskLists {
		id, err := database.NewShortID()
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO task_lists (id, project_id, name, is_done, position) VALUES (?, ?, ?, ?, ?)", id, projectID, list.name, list.isDone, index+1); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) loadLists(ctx context.Context, projectID string) ([]TaskList, error) {
	rows, err := h.db.QueryContext(ctx, "SELECT id, name, is_done, position FROM task_lists WHERE project_id = ? ORDER BY position, id", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lists := make([]TaskList, 0)
	for rows.Next() {
		var list TaskList
		if err := rows.Scan(&list.ID, &list.Name, &list.IsDone, &list.Position); err != nil {
			return nil, err
		}
		lists = append(lists, list)
	}
	return lists, rows.Err()
}

func taskListInput(w http.ResponseWriter, r *http.Request) (taskListPayload, bool) {
	var input taskListPayload
	if !decodeJSON(w, r, &input) {
		return input, false
	}
	input.Name = strings.TrimSpace(input.Name)
	if length := utf8.RuneCountInString(input.Name); length < 1 || length > 80 {
		writeError(w, http.StatusBadRequest, "name must be 1 to 80 characters")
		return input, false
	}
	return input, true
}

func (h *Handler) createTaskList(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, "")
	if !ok {
		return
	}
	input, ok := taskListInput(w, r)
	if !ok {
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate list id")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO task_lists (id, project_id, name, is_done, position) SELECT ?, ?, ?, ?, COALESCE(MAX(position), 0) + 1 FROM task_lists WHERE project_id = ?", id, projectID, input.Name, input.IsDone, projectID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create list")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) updateTaskList(w http.ResponseWriter, r *http.Request) {
	projectID, listID, ok := h.authorizeTaskList(w, r, "")
	if !ok {
		return
	}
	input, ok := taskListInput(w, r)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE task_lists SET name = ?, is_done = ? WHERE id = ? AND project_id = ?", input.Name, input.IsDone, listID, projectID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update list")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) reorderTaskLists(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, "")
	if !ok {
		return
	}
	var input taskListOrderPayload
	if !decodeJSON(w, r, &input) {
		return
	}
	lists, err := h.loadLists(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read task lists")
		return
	}
	current := make([]string, len(lists))
	for index, list := range lists {
		current[index] = list.ID
	}
	if !sameIDs(current, input.IDs) {
		writeError(w, http.StatusConflict, "the lists changed, reload and try again")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not reorder lists")
		return
	}
	defer tx.Rollback()
	for index, id := range input.IDs {
		if _, err := tx.ExecContext(r.Context(), "UPDATE task_lists SET position = ? WHERE id = ? AND project_id = ?", index+1, id, projectID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not reorder lists")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not reorder lists")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteTaskList(w http.ResponseWriter, r *http.Request) {
	projectID, listID, ok := h.authorizeTaskList(w, r, permissions.TasksDelete)
	if !ok {
		return
	}
	result, err := h.db.ExecContext(r.Context(), "DELETE FROM task_lists WHERE id = ? AND project_id = ? AND NOT EXISTS (SELECT 1 FROM project_tasks WHERE list_id = ?)", listID, projectID, listID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete list")
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete list")
		return
	}
	if deleted == 0 {
		writeError(w, http.StatusConflict, "move or delete the tasks in this list first")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) authorizeTaskList(w http.ResponseWriter, r *http.Request, permission string) (string, string, bool) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, permission)
	if !ok {
		return "", "", false
	}
	listID, ok := pathID(w, r, "listID")
	if !ok || !h.scopeExists(w, r, "SELECT EXISTS(SELECT 1 FROM task_lists WHERE id = ? AND project_id = ?)", "list not found", listID, projectID) {
		return "", "", false
	}
	return projectID, listID, true
}
