package tasks

import (
	"context"
	"net/http"
	"time"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
)

type TaskTimeEntry struct {
	ID        string     `json:"id"`
	User      TaskUser   `json:"user"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
}

func (h *Handler) loadTimeEntries(ctx context.Context, taskID string) ([]TaskTimeEntry, error) {
	rows, err := h.db.QueryContext(ctx, "SELECT e.id, u.id, u.username, e.started_at, e.ended_at FROM task_time_entries e JOIN users u ON u.id = e.user_id WHERE e.task_id = ? ORDER BY e.started_at DESC, e.id", taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]TaskTimeEntry, 0)
	for rows.Next() {
		var entry TaskTimeEntry
		if err := rows.Scan(&entry.ID, &entry.User.ID, &entry.User.Username, &entry.StartedAt, &entry.EndedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// startTimer stops whatever timer the user has running, on any task, and starts one on this task.
func (h *Handler) startTimer(w http.ResponseWriter, r *http.Request) {
	_, projectID, taskID, ok := h.authorizeTask(w, r, "")
	if !ok {
		return
	}
	user, _ := auth.UserFromContext(r)
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start timer")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start timer")
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(), "UPDATE task_time_entries SET ended_at = CURRENT_TIMESTAMP WHERE user_id = ? AND ended_at IS NULL", user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not start timer")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "INSERT INTO task_time_entries (id, task_id, user_id) VALUES (?, ?, ?)", id, taskID, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not start timer")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not start timer")
		return
	}
	h.writeTaskDetail(w, r, http.StatusOK, projectID, taskID)
}

func (h *Handler) stopTimer(w http.ResponseWriter, r *http.Request) {
	_, projectID, taskID, ok := h.authorizeTask(w, r, "")
	if !ok {
		return
	}
	user, _ := auth.UserFromContext(r)
	if _, err := h.db.ExecContext(r.Context(), "UPDATE task_time_entries SET ended_at = CURRENT_TIMESTAMP WHERE task_id = ? AND user_id = ? AND ended_at IS NULL", taskID, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not stop timer")
		return
	}
	h.writeTaskDetail(w, r, http.StatusOK, projectID, taskID)
}

func (h *Handler) createTimeEntry(w http.ResponseWriter, r *http.Request) {
	_, projectID, taskID, ok := h.authorizeTask(w, r, "")
	if !ok {
		return
	}
	var input struct {
		Minutes int    `json:"minutes"`
		Date    string `json:"date"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Minutes < 1 || input.Minutes > 24*60 {
		writeError(w, http.StatusBadRequest, "minutes must be between 1 and 1440")
		return
	}
	length := time.Duration(input.Minutes) * time.Minute
	start := time.Now().UTC().Add(-length)
	if input.Date != "" {
		day, err := time.Parse(time.DateOnly, input.Date)
		if err != nil || day.After(time.Now()) {
			writeError(w, http.StatusBadRequest, "date must be a past date in YYYY-MM-DD format")
			return
		}
		// noon UTC keeps the entry on the same calendar day in every timezone
		start = day.Add(12 * time.Hour)
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not log time")
		return
	}
	user, _ := auth.UserFromContext(r)
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO task_time_entries (id, task_id, user_id, started_at, ended_at) VALUES (?, ?, ?, ?, ?)", id, taskID, user.ID, start, start.Add(length)); err != nil {
		writeError(w, http.StatusInternalServerError, "could not log time")
		return
	}
	h.writeTaskDetail(w, r, http.StatusCreated, projectID, taskID)
}

func (h *Handler) deleteTimeEntry(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, projectID, ok := h.authorizeProjectAccess(w, r, "")
	if !ok {
		return
	}
	entryID, ok := pathID(w, r, "entryID")
	if !ok {
		return
	}
	var ownerID uint64
	err := h.db.QueryRowContext(r.Context(), "SELECT e.user_id FROM task_time_entries e JOIN project_tasks t ON t.id = e.task_id WHERE e.id = ? AND t.project_id = ?", entryID, projectID).Scan(&ownerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "time entry not found")
		return
	}
	if ownerID != user.ID && !h.authorize(w, r, user, workspaceID, permissions.TasksDelete) {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "DELETE FROM task_time_entries WHERE id = ?", entryID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete time entry")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
