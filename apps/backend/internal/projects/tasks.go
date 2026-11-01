package projects

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"kymber/backend/internal/database"
)

const taskColumns = `t.id, t.project_id, t.title, t.status, t.priority, u.id, u.username, DATE_FORMAT(t.due_date, '%Y-%m-%d'), t.created_at, t.updated_at
	FROM project_tasks t LEFT JOIN users u ON u.id = t.assignee_id`

type Task struct {
	ID        string        `json:"id"`
	ProjectID string        `json:"project_id"`
	Title     string        `json:"title"`
	Status    string        `json:"status"`
	Priority  string        `json:"priority"`
	Assignee  *TaskAssignee `json:"assignee"`
	DueDate   *string       `json:"due_date"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type TaskAssignee struct {
	ID       uint64 `json:"id,string"`
	Username string `json:"username"`
}

type taskPayload struct {
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Priority   string  `json:"priority"`
	AssigneeID *uint64 `json:"assignee_id,string"`
	DueDate    *string `json:"due_date"`
}

func normalizeTask(input *taskPayload) string {
	input.Title = strings.TrimSpace(input.Title)
	if length := utf8.RuneCountInString(input.Title); length < 1 || length > 200 {
		return "title must be 1 to 200 characters"
	}
	switch input.Status {
	case "todo", "in_progress", "in_review", "done":
	default:
		return "status must be todo, in_progress, in_review, or done"
	}
	switch input.Priority {
	case "urgent", "high", "normal":
	default:
		return "priority must be urgent, high, or normal"
	}
	if input.DueDate != nil {
		if _, err := time.Parse(time.DateOnly, *input.DueDate); err != nil {
			return "due_date must be a valid date in YYYY-MM-DD format"
		}
	}
	return ""
}

func scanTask(row interface{ Scan(...any) error }) (Task, error) {
	var task Task
	var assigneeID sql.NullInt64
	var assigneeName, dueDate sql.NullString
	if err := row.Scan(&task.ID, &task.ProjectID, &task.Title, &task.Status, &task.Priority, &assigneeID, &assigneeName, &dueDate, &task.CreatedAt, &task.UpdatedAt); err != nil {
		return task, err
	}
	if assigneeID.Valid {
		task.Assignee = &TaskAssignee{ID: uint64(assigneeID.Int64), Username: assigneeName.String}
	}
	if dueDate.Valid {
		task.DueDate = &dueDate.String
	}
	return task, nil
}

func (h *Handler) taskInput(w http.ResponseWriter, r *http.Request, workspaceID string) (taskPayload, bool) {
	var input taskPayload
	if !decodeJSON(w, r, &input) {
		return input, false
	}
	if message := normalizeTask(&input); message != "" {
		writeError(w, http.StatusBadRequest, message)
		return input, false
	}
	if input.AssigneeID != nil {
		var member bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id = ? AND user_id = ?)", workspaceID, *input.AssigneeID).Scan(&member); err != nil {
			writeError(w, http.StatusInternalServerError, "could not check task assignee")
			return input, false
		}
		if !member {
			writeError(w, http.StatusBadRequest, "assignee must be a workspace member")
			return input, false
		}
	}
	return input, true
}

func (h *Handler) findTask(r *http.Request, projectID, taskID string) (Task, error) {
	return scanTask(h.db.QueryRowContext(r.Context(), "SELECT "+taskColumns+" WHERE t.id = ? AND t.project_id = ?", taskID, projectID))
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, false)
	if !ok {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT "+taskColumns+" WHERE t.project_id = ? ORDER BY t.created_at, t.id", projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list tasks")
		return
	}
	defer rows.Close()
	tasks := make([]Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not read tasks")
			return
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read tasks")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]Task{"tasks": tasks})
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	workspaceID, projectID, ok := h.authorizeProjectAccess(w, r, false)
	if !ok {
		return
	}
	input, ok := h.taskInput(w, r, workspaceID)
	if !ok {
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate task id")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO project_tasks (id, project_id, title, status, priority, assignee_id, due_date) VALUES (?, ?, ?, ?, ?, ?, ?)", id, projectID, input.Title, input.Status, input.Priority, input.AssigneeID, input.DueDate); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create task")
		return
	}
	task, err := h.findTask(r, projectID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read created task")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]Task{"task": task})
}

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	workspaceID, projectID, ok := h.authorizeProjectAccess(w, r, false)
	if !ok {
		return
	}
	taskID, ok := pathID(w, r, "taskID")
	if !ok {
		return
	}
	input, ok := h.taskInput(w, r, workspaceID)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE project_tasks SET title = ?, status = ?, priority = ?, assignee_id = ?, due_date = ? WHERE id = ? AND project_id = ?", input.Title, input.Status, input.Priority, input.AssigneeID, input.DueDate, taskID, projectID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}
	task, err := h.findTask(r, projectID, taskID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read updated task")
		return
	}
	writeJSON(w, http.StatusOK, map[string]Task{"task": task})
}

func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, true)
	if !ok {
		return
	}
	taskID, ok := pathID(w, r, "taskID")
	if !ok {
		return
	}
	result, err := h.db.ExecContext(r.Context(), "DELETE FROM project_tasks WHERE id = ? AND project_id = ?", taskID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete task")
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete task")
		return
	}
	if deleted == 0 {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
