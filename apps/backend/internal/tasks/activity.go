package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"time"
)

type TaskActivity struct {
	ID        uint64          `json:"id,string"`
	Actor     *TaskUser       `json:"actor"`
	Kind      string          `json:"kind"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type activityEvent struct {
	kind    string
	payload map[string]any
}

type queryer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func recordActivity(ctx context.Context, db queryer, taskID string, actorID uint64, events ...activityEvent) error {
	for _, event := range events {
		payload, err := json.Marshal(event.payload)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO task_activity (task_id, actor_id, kind, payload) VALUES (?, ?, ?, ?)", taskID, actorID, event.kind, payload); err != nil {
			return err
		}
	}
	return nil
}

type taskSnapshot struct {
	title, priority, dueDate, description string
	assignees, labels                     []string
}

func loadSnapshot(ctx context.Context, db queryer, taskID string) (taskSnapshot, error) {
	var s taskSnapshot
	err := db.QueryRowContext(ctx, "SELECT title, priority, COALESCE(DATE_FORMAT(due_date, '%Y-%m-%d'), ''), description FROM project_tasks WHERE id = ?", taskID).Scan(&s.title, &s.priority, &s.dueDate, &s.description)
	if err != nil {
		return s, err
	}
	if s.assignees, err = queryStrings(ctx, db, "SELECT u.username FROM task_assignees a JOIN users u ON u.id = a.user_id WHERE a.task_id = ? ORDER BY u.username", taskID); err != nil {
		return s, err
	}
	s.labels, err = queryStrings(ctx, db, "SELECT l.name FROM task_labels tl JOIN project_labels l ON l.id = tl.label_id WHERE tl.task_id = ? ORDER BY l.name", taskID)
	return s, err
}

func queryStrings(ctx context.Context, db queryer, query string, args ...any) ([]string, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func diffSnapshots(before, after taskSnapshot) []activityEvent {
	var events []activityEvent
	change := func(kind, from, to string) {
		if from != to {
			events = append(events, activityEvent{kind, map[string]any{"from": from, "to": to}})
		}
	}
	change("title", before.title, after.title)
	change("priority", before.priority, after.priority)
	change("due_date", before.dueDate, after.dueDate)
	if before.description != after.description {
		events = append(events, activityEvent{"description", map[string]any{}})
	}
	members := func(kind string, from, to []string) {
		added, removed := missing(to, from), missing(from, to)
		if len(added) > 0 || len(removed) > 0 {
			events = append(events, activityEvent{kind, map[string]any{"added": added, "removed": removed}})
		}
	}
	members("assignees", before.assignees, after.assignees)
	members("labels", before.labels, after.labels)
	return events
}

// missing returns the values in a that are not in b.
func missing(a, b []string) []string {
	result := make([]string, 0)
	for _, value := range a {
		if !slices.Contains(b, value) {
			result = append(result, value)
		}
	}
	return result
}

func (h *Handler) loadActivity(ctx context.Context, taskID string) ([]TaskActivity, error) {
	rows, err := h.db.QueryContext(ctx, "SELECT a.id, u.id, u.username, a.kind, a.payload, a.created_at FROM task_activity a LEFT JOIN users u ON u.id = a.actor_id WHERE a.task_id = ? ORDER BY a.id", taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	activity := make([]TaskActivity, 0)
	for rows.Next() {
		var item TaskActivity
		var actorID sql.NullInt64
		var actorName sql.NullString
		var payload []byte
		if err := rows.Scan(&item.ID, &actorID, &actorName, &item.Kind, &payload, &item.CreatedAt); err != nil {
			return nil, err
		}
		if actorID.Valid {
			item.Actor = &TaskUser{ID: uint64(actorID.Int64), Username: actorName.String}
		}
		item.Payload = payload
		activity = append(activity, item)
	}
	return activity, rows.Err()
}
