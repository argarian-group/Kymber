package tasks

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
)

const maxTaskRelations = 50

type Task struct {
	ID             string     `json:"id"`
	Number         int        `json:"number"`
	Key            string     `json:"key"`
	ProjectID      string     `json:"project_id"`
	ListID         string     `json:"list_id"`
	Position       int        `json:"position"`
	Title          string     `json:"title"`
	HasDescription bool       `json:"has_description"`
	Priority       string     `json:"priority"`
	StartDate      *string    `json:"start_date"`
	DueDate        *string    `json:"due_date"`
	Assignees      []TaskUser `json:"assignees"`
	LabelIDs       []string   `json:"label_ids"`
	ChecklistDone  int        `json:"checklist_done"`
	ChecklistTotal int        `json:"checklist_total"`
	CommentCount   int        `json:"comment_count"`
	ArchivedAt     *time.Time `json:"archived_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type TaskUser struct {
	ID       uint64 `json:"id,string"`
	Username string `json:"username"`
}

type TaskDetail struct {
	Task
	Description string          `json:"description"`
	Checklists  []TaskChecklist `json:"checklists"`
	Comments    []TaskComment   `json:"comments"`
	Activity    []TaskActivity  `json:"activity"`
	TimeEntries []TaskTimeEntry `json:"time_entries"`
	DeployedAt  *time.Time      `json:"deployed_at"`
}

type taskCreatePayload struct {
	ListID string `json:"list_id"`
	Title  string `json:"title"`
}

type taskPayload struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	StartDate   *string  `json:"start_date"`
	DueDate     *string  `json:"due_date"`
	AssigneeIDs []string `json:"assignee_ids"`
	LabelIDs    []string `json:"label_ids"`
}

type taskMovePayload struct {
	ListID   string `json:"list_id"`
	Position int    `json:"position"`
}

func normalizeTask(input *taskPayload) string {
	input.Title = strings.TrimSpace(input.Title)
	if length := utf8.RuneCountInString(input.Title); length < 1 || length > 200 {
		return "title must be 1 to 200 characters"
	}
	if utf8.RuneCountInString(input.Description) > 20000 {
		return "description must be 20000 characters or fewer"
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
	if input.StartDate != nil {
		if _, err := time.Parse(time.DateOnly, *input.StartDate); err != nil {
			return "start_date must be a valid date in YYYY-MM-DD format"
		}
		if input.DueDate != nil && *input.StartDate > *input.DueDate {
			return "start_date must be on or before due_date"
		}
	}
	if len(input.AssigneeIDs) > maxTaskRelations || len(input.LabelIDs) > maxTaskRelations {
		return "too many assignees or labels"
	}
	return ""
}

func insertAt(ids []string, id string, position int) []string {
	position = max(0, min(position, len(ids)))
	result := make([]string, 0, len(ids)+1)
	result = append(result, ids[:position]...)
	result = append(result, id)
	return append(result, ids[position:]...)
}

// loadTasks returns one task when taskID is set, otherwise the project's archived or active tasks.
func (h *Handler) loadTasks(ctx context.Context, projectID, taskID string, archived bool) ([]Task, error) {
	filter, args := "t.project_id = ?", []any{projectID}
	switch {
	case taskID != "":
		filter, args = filter+" AND t.id = ?", append(args, taskID)
	case archived:
		filter += " AND t.archived_at IS NOT NULL"
	default:
		filter += " AND t.archived_at IS NULL"
	}
	rows, err := h.db.QueryContext(ctx, `SELECT t.id, t.number, CONCAT(p.task_prefix, '-', t.number), t.project_id, t.list_id, t.position, t.title, t.description <> '', t.priority, DATE_FORMAT(t.start_date, '%Y-%m-%d'), DATE_FORMAT(t.due_date, '%Y-%m-%d'),
		(SELECT COUNT(*) FROM task_checklist_items i JOIN task_checklists c ON c.id = i.checklist_id WHERE c.task_id = t.id AND i.done),
		(SELECT COUNT(*) FROM task_checklist_items i JOIN task_checklists c ON c.id = i.checklist_id WHERE c.task_id = t.id),
		(SELECT COUNT(*) FROM task_comments m WHERE m.task_id = t.id),
		t.created_at, t.updated_at, t.archived_at
		FROM project_tasks t JOIN projects p ON p.id = t.project_id WHERE `+filter+` ORDER BY t.position, t.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := make([]Task, 0)
	index := make(map[string]int)
	for rows.Next() {
		task := Task{Assignees: []TaskUser{}, LabelIDs: []string{}}
		var startDate, dueDate sql.NullString
		if err := rows.Scan(&task.ID, &task.Number, &task.Key, &task.ProjectID, &task.ListID, &task.Position, &task.Title, &task.HasDescription, &task.Priority, &startDate, &dueDate, &task.ChecklistDone, &task.ChecklistTotal, &task.CommentCount, &task.CreatedAt, &task.UpdatedAt, &task.ArchivedAt); err != nil {
			return nil, err
		}
		if startDate.Valid {
			task.StartDate = &startDate.String
		}
		if dueDate.Valid {
			task.DueDate = &dueDate.String
		}
		index[task.ID] = len(tasks)
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	assignees, err := h.db.QueryContext(ctx, `SELECT a.task_id, u.id, u.username FROM task_assignees a JOIN users u ON u.id = a.user_id JOIN project_tasks t ON t.id = a.task_id WHERE `+filter+` ORDER BY u.username`, args...)
	if err != nil {
		return nil, err
	}
	defer assignees.Close()
	for assignees.Next() {
		var id string
		var assignee TaskUser
		if err := assignees.Scan(&id, &assignee.ID, &assignee.Username); err != nil {
			return nil, err
		}
		tasks[index[id]].Assignees = append(tasks[index[id]].Assignees, assignee)
	}
	if err := assignees.Err(); err != nil {
		return nil, err
	}

	labels, err := h.db.QueryContext(ctx, `SELECT tl.task_id, tl.label_id FROM task_labels tl JOIN project_labels l ON l.id = tl.label_id JOIN project_tasks t ON t.id = tl.task_id WHERE `+filter+` ORDER BY l.name, l.id`, args...)
	if err != nil {
		return nil, err
	}
	defer labels.Close()
	for labels.Next() {
		var id, labelID string
		if err := labels.Scan(&id, &labelID); err != nil {
			return nil, err
		}
		tasks[index[id]].LabelIDs = append(tasks[index[id]].LabelIDs, labelID)
	}
	return tasks, labels.Err()
}

func (h *Handler) getBoard(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, "")
	if !ok {
		return
	}
	lists, err := h.loadLists(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read task lists")
		return
	}
	labels, err := h.loadLabels(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read task labels")
		return
	}
	tasks, err := h.loadTasks(r.Context(), projectID, "", r.URL.Query().Get("archived") == "1")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read tasks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lists": lists, "labels": labels, "tasks": tasks})
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, "")
	if !ok {
		return
	}
	taskID, ok := pathID(w, r, "taskID")
	if !ok {
		return
	}
	h.writeTaskDetail(w, r, http.StatusOK, projectID, taskID)
}

func (h *Handler) writeTaskDetail(w http.ResponseWriter, r *http.Request, status int, projectID, taskID string) {
	tasks, err := h.loadTasks(r.Context(), projectID, taskID, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read task")
		return
	}
	if len(tasks) == 0 {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	detail := TaskDetail{Task: tasks[0]}
	if err := h.db.QueryRowContext(r.Context(), "SELECT description FROM project_tasks WHERE id = ?", taskID).Scan(&detail.Description); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read task")
		return
	}
	if detail.Checklists, err = h.loadChecklists(r.Context(), taskID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read task checklists")
		return
	}
	if detail.Comments, err = h.loadComments(r.Context(), taskID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read task comments")
		return
	}
	if detail.Activity, err = h.loadActivity(r.Context(), taskID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read task activity")
		return
	}
	if detail.TimeEntries, err = h.loadTimeEntries(r.Context(), taskID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read time entries")
		return
	}
	if detail.DeployedAt, err = h.deployedAt(r.Context(), taskID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read task deploys")
		return
	}
	writeJSON(w, status, map[string]TaskDetail{"task": detail})
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, "")
	if !ok {
		return
	}
	var input taskCreatePayload
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if length := utf8.RuneCountInString(input.Title); length < 1 || length > 200 {
		writeError(w, http.StatusBadRequest, "title must be 1 to 200 characters")
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate task id")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create task")
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(), "UPDATE projects SET task_counter = LAST_INSERT_ID(task_counter + 1) WHERE id = ?", projectID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not number task")
		return
	}
	result, err := tx.ExecContext(r.Context(), `INSERT INTO project_tasks (id, project_id, list_id, number, position, title, priority)
		SELECT ?, l.project_id, l.id, LAST_INSERT_ID(), COALESCE((SELECT MAX(t.position) FROM project_tasks t WHERE t.list_id = l.id), 0) + 1, ?, 'normal'
		FROM task_lists l WHERE l.id = ? AND l.project_id = ?`, id, input.Title, input.ListID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create task")
		return
	}
	if created, err := result.RowsAffected(); err != nil || created == 0 {
		writeError(w, http.StatusBadRequest, "list not found in this project")
		return
	}
	user, _ := auth.UserFromContext(r)
	if err := recordActivity(r.Context(), tx, id, user.ID, activityEvent{"created", map[string]any{}}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create task")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create task")
		return
	}
	h.writeTaskDetail(w, r, http.StatusCreated, projectID, id)
}

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	workspaceID, projectID, taskID, ok := h.authorizeTask(w, r, "")
	if !ok {
		return
	}
	var input taskPayload
	if !decodeJSON(w, r, &input) {
		return
	}
	if message := normalizeTask(&input); message != "" {
		writeError(w, http.StatusBadRequest, message)
		return
	}
	assigneeIDs := make([]uint64, 0, len(input.AssigneeIDs))
	for _, raw := range input.AssigneeIDs {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid assignee id")
			return
		}
		var member bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id = ? AND user_id = ?)", workspaceID, id).Scan(&member); err != nil {
			writeError(w, http.StatusInternalServerError, "could not check task assignee")
			return
		}
		if !member {
			writeError(w, http.StatusBadRequest, "assignee must be a workspace member")
			return
		}
		assigneeIDs = append(assigneeIDs, id)
	}
	for _, id := range input.LabelIDs {
		var found bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM project_labels WHERE id = ? AND project_id = ?)", id, projectID).Scan(&found); err != nil {
			writeError(w, http.StatusInternalServerError, "could not check task label")
			return
		}
		if !found {
			writeError(w, http.StatusBadRequest, "label not found in this project")
			return
		}
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}
	defer tx.Rollback()
	before, err := loadSnapshot(r.Context(), tx, taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}
	previousAssignees, err := queryIDs(r.Context(), tx, "SELECT user_id FROM task_assignees WHERE task_id = ?", taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}
	if err := saveTask(r.Context(), tx, taskID, input, assigneeIDs); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}
	after, err := loadSnapshot(r.Context(), tx, taskID)
	if err == nil {
		user, _ := auth.UserFromContext(r)
		err = recordActivity(r.Context(), tx, taskID, user.ID, diffSnapshots(before, after)...)
		if err == nil {
			added := slices.DeleteFunc(slices.Clone(assigneeIDs), func(id uint64) bool { return slices.Contains(previousAssignees, id) })
			err = notify(r.Context(), tx, taskID, user.ID, "assigned", added)
		}
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}
	h.writeTaskDetail(w, r, http.StatusOK, projectID, taskID)
}

func saveTask(ctx context.Context, tx *sql.Tx, taskID string, input taskPayload, assigneeIDs []uint64) error {
	if _, err := tx.ExecContext(ctx, "UPDATE project_tasks SET title = ?, description = ?, priority = ?, start_date = ?, due_date = ? WHERE id = ?", input.Title, input.Description, input.Priority, input.StartDate, input.DueDate, taskID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM task_assignees WHERE task_id = ?", taskID); err != nil {
		return err
	}
	for _, id := range assigneeIDs {
		if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO task_assignees (task_id, user_id) VALUES (?, ?)", taskID, id); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM task_labels WHERE task_id = ?", taskID); err != nil {
		return err
	}
	for _, id := range input.LabelIDs {
		if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO task_labels (task_id, label_id) VALUES (?, ?)", taskID, id); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) moveTask(w http.ResponseWriter, r *http.Request) {
	_, projectID, taskID, ok := h.authorizeTask(w, r, "")
	if !ok {
		return
	}
	var input taskMovePayload
	if !decodeJSON(w, r, &input) {
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not move task")
		return
	}
	defer tx.Rollback()
	var locked string
	err = tx.QueryRowContext(r.Context(), "SELECT id FROM task_lists WHERE id = ? AND project_id = ? FOR UPDATE", input.ListID, projectID).Scan(&locked)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusBadRequest, "list not found in this project")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not move task")
		return
	}
	var fromListID, fromList, toList string
	if err := tx.QueryRowContext(r.Context(), "SELECT l.id, l.name FROM project_tasks t JOIN task_lists l ON l.id = t.list_id WHERE t.id = ?", taskID).Scan(&fromListID, &fromList); err != nil {
		writeError(w, http.StatusInternalServerError, "could not move task")
		return
	}
	if err := tx.QueryRowContext(r.Context(), "SELECT name FROM task_lists WHERE id = ?", input.ListID).Scan(&toList); err != nil {
		writeError(w, http.StatusInternalServerError, "could not move task")
		return
	}
	rows, err := tx.QueryContext(r.Context(), "SELECT id FROM project_tasks WHERE list_id = ? AND id <> ? ORDER BY position, id", input.ListID, taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not move task")
		return
	}
	siblings := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "could not move task")
			return
		}
		siblings = append(siblings, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not move task")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "UPDATE project_tasks SET list_id = ? WHERE id = ?", input.ListID, taskID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not move task")
		return
	}
	if fromListID != input.ListID {
		user, _ := auth.UserFromContext(r)
		if err := recordActivity(r.Context(), tx, taskID, user.ID, activityEvent{"moved", map[string]any{"from": fromList, "to": toList}}); err != nil {
			writeError(w, http.StatusInternalServerError, "could not move task")
			return
		}
	}
	for position, id := range insertAt(siblings, taskID, input.Position) {
		if _, err := tx.ExecContext(r.Context(), "UPDATE project_tasks SET position = ?, updated_at = updated_at WHERE id = ?", position+1, id); err != nil {
			writeError(w, http.StatusInternalServerError, "could not move task")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not move task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	_, _, taskID, ok := h.authorizeTask(w, r, permissions.TasksDelete)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "DELETE FROM project_tasks WHERE id = ?", taskID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) authorizeTask(w http.ResponseWriter, r *http.Request, permission string) (string, string, string, bool) {
	workspaceID, projectID, ok := h.authorizeProjectAccess(w, r, permission)
	if !ok {
		return "", "", "", false
	}
	taskID, ok := pathID(w, r, "taskID")
	if !ok || !h.scopeExists(w, r, "SELECT EXISTS(SELECT 1 FROM project_tasks WHERE id = ? AND project_id = ?)", "task not found", taskID, projectID) {
		return "", "", "", false
	}
	return workspaceID, projectID, taskID, true
}

func (h *Handler) scopeExists(w http.ResponseWriter, r *http.Request, query, notFound string, args ...any) bool {
	var found bool
	if err := h.db.QueryRowContext(r.Context(), query, args...).Scan(&found); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check access")
		return false
	}
	if !found {
		writeError(w, http.StatusNotFound, notFound)
		return false
	}
	return true
}

func (h *Handler) archiveTask(w http.ResponseWriter, r *http.Request) {
	h.setArchived(w, r, true)
}

func (h *Handler) restoreTask(w http.ResponseWriter, r *http.Request) {
	h.setArchived(w, r, false)
}

func (h *Handler) setArchived(w http.ResponseWriter, r *http.Request, archive bool) {
	_, projectID, taskID, ok := h.authorizeTask(w, r, "")
	if !ok {
		return
	}
	query, kind := "UPDATE project_tasks SET archived_at = CURRENT_TIMESTAMP, updated_at = updated_at WHERE id = ? AND archived_at IS NULL", "archived"
	if !archive {
		query, kind = "UPDATE project_tasks SET archived_at = NULL, updated_at = updated_at WHERE id = ? AND archived_at IS NOT NULL", "restored"
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(r.Context(), query, taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}
	if changed, _ := result.RowsAffected(); changed == 1 {
		user, _ := auth.UserFromContext(r)
		if err := recordActivity(r.Context(), tx, taskID, user.ID, activityEvent{kind, map[string]any{}}); err != nil {
			writeError(w, http.StatusInternalServerError, "could not update task")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}
	h.writeTaskDetail(w, r, http.StatusOK, projectID, taskID)
}
