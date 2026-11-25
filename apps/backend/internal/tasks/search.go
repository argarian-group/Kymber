package tasks

import (
	"database/sql"
	"net/http"
	"strings"

	"kymber/backend/internal/auth"
)

type WorkspaceTask struct {
	ID          string  `json:"id"`
	Key         string  `json:"key"`
	Title       string  `json:"title"`
	Priority    string  `json:"priority"`
	DueDate     *string `json:"due_date"`
	ProjectID   string  `json:"project_id"`
	ProjectName string  `json:"project_name"`
	ListName    string  `json:"list_name"`
	Done        bool    `json:"done"`
}

// searchTasks lists active tasks across the workspace: ?mine=1 limits them to the caller's, ?q= matches a title or PREFIX-N.
func (h *Handler) searchTasks(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, "") {
		return
	}
	filter, args := "p.workspace_id = ? AND t.archived_at IS NULL", []any{workspaceID}
	if r.URL.Query().Get("mine") == "1" {
		filter += " AND EXISTS(SELECT 1 FROM task_assignees a WHERE a.task_id = t.id AND a.user_id = ?)"
		args = append(args, user.ID)
	}
	limit := 500
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		filter += " AND (t.title LIKE ? OR CONCAT(p.task_prefix, '-', t.number) = ?)"
		args = append(args, "%"+escapeLike(q)+"%", strings.ToUpper(q))
		limit = 20
	}
	rows, err := h.db.QueryContext(r.Context(), `SELECT t.id, CONCAT(p.task_prefix, '-', t.number), t.title, t.priority, DATE_FORMAT(t.due_date, '%Y-%m-%d'), p.id, p.name, l.name, l.is_done
		FROM project_tasks t JOIN projects p ON p.id = t.project_id JOIN task_lists l ON l.id = t.list_id
		WHERE `+filter+` ORDER BY l.is_done, t.due_date IS NULL, t.due_date, t.updated_at DESC LIMIT ?`, append(args, limit)...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not search tasks")
		return
	}
	defer rows.Close()
	tasks := make([]WorkspaceTask, 0)
	for rows.Next() {
		var task WorkspaceTask
		var dueDate sql.NullString
		if err := rows.Scan(&task.ID, &task.Key, &task.Title, &task.Priority, &dueDate, &task.ProjectID, &task.ProjectName, &task.ListName, &task.Done); err != nil {
			writeError(w, http.StatusInternalServerError, "could not search tasks")
			return
		}
		if dueDate.Valid {
			task.DueDate = &dueDate.String
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not search tasks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

func escapeLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(value)
}
