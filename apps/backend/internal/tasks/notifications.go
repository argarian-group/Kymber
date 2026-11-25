package tasks

import (
	"context"
	"database/sql"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"kymber/backend/internal/auth"
)

type Notification struct {
	ID        uint64     `json:"id,string"`
	Kind      string     `json:"kind"`
	Actor     *TaskUser  `json:"actor"`
	TaskID    string     `json:"task_id"`
	TaskKey   string     `json:"task_key"`
	TaskTitle string     `json:"task_title"`
	ProjectID string     `json:"project_id"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

var mentionPattern = regexp.MustCompile(`(?:^|[^A-Za-z0-9_@-])@([A-Za-z0-9_-]{3,32})`)

func notify(ctx context.Context, db queryer, taskID string, actorID uint64, kind string, userIDs []uint64) error {
	for _, id := range userIDs {
		if id == actorID {
			continue
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO notifications (user_id, actor_id, task_id, kind) VALUES (?, ?, ?, ?)", id, actorID, taskID, kind); err != nil {
			return err
		}
	}
	return nil
}

func queryIDs(ctx context.Context, db queryer, query string, args ...any) ([]uint64, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]uint64, 0)
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// notifyComment tells mentioned workspace members, then the task's other assignees.
func notifyComment(ctx context.Context, db queryer, workspaceID, taskID string, actorID uint64, body string) error {
	var names []any
	for _, match := range mentionPattern.FindAllStringSubmatch(body, 20) {
		names = append(names, match[1])
	}
	mentioned := []uint64{}
	if len(names) > 0 {
		var err error
		mentioned, err = queryIDs(ctx, db, "SELECT u.id FROM users u JOIN workspace_members m ON m.user_id = u.id AND m.workspace_id = ? WHERE u.username IN (?"+strings.Repeat(", ?", len(names)-1)+")", append([]any{workspaceID}, names...)...)
		if err != nil {
			return err
		}
	}
	assignees, err := queryIDs(ctx, db, "SELECT user_id FROM task_assignees WHERE task_id = ?", taskID)
	if err != nil {
		return err
	}
	assignees = slices.DeleteFunc(assignees, func(id uint64) bool { return slices.Contains(mentioned, id) })
	if err := notify(ctx, db, taskID, actorID, "mentioned", mentioned); err != nil {
		return err
	}
	return notify(ctx, db, taskID, actorID, "commented", assignees)
}

func (h *Handler) listNotifications(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	rows, err := h.db.QueryContext(r.Context(), `SELECT n.id, n.kind, a.id, a.username, t.id, CONCAT(p.task_prefix, '-', t.number), t.title, p.id, n.read_at, n.created_at
		FROM notifications n JOIN project_tasks t ON t.id = n.task_id JOIN projects p ON p.id = t.project_id LEFT JOIN users a ON a.id = n.actor_id
		WHERE n.user_id = ? ORDER BY n.id DESC LIMIT 50`, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read notifications")
		return
	}
	defer rows.Close()
	notifications := make([]Notification, 0)
	for rows.Next() {
		var n Notification
		var actorID sql.NullInt64
		var actorName sql.NullString
		if err := rows.Scan(&n.ID, &n.Kind, &actorID, &actorName, &n.TaskID, &n.TaskKey, &n.TaskTitle, &n.ProjectID, &n.ReadAt, &n.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read notifications")
			return
		}
		if actorID.Valid {
			n.Actor = &TaskUser{ID: uint64(actorID.Int64), Username: actorName.String}
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read notifications")
		return
	}
	var unread int
	if err := h.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at IS NULL", user.ID).Scan(&unread); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read notifications")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notifications": notifications, "unread": unread})
}

// readNotifications marks one notification as read, or all of them when id is empty.
func (h *Handler) readNotifications(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	var input struct {
		ID string `json:"id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	query, args := "UPDATE notifications SET read_at = CURRENT_TIMESTAMP WHERE user_id = ? AND read_at IS NULL", []any{user.ID}
	if input.ID != "" {
		id, err := strconv.ParseUint(input.ID, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid notification id")
			return
		}
		query, args = query+" AND id = ?", append(args, id)
	}
	if _, err := h.db.ExecContext(r.Context(), query, args...); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update notifications")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
