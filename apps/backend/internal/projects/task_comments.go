package projects

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
)

type TaskComment struct {
	ID        string        `json:"id"`
	Author    *TaskAssignee `json:"author"`
	Body      string        `json:"body"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type commentPayload struct {
	Body string `json:"body"`
}

func (h *Handler) loadComments(ctx context.Context, taskID string) ([]TaskComment, error) {
	rows, err := h.db.QueryContext(ctx, "SELECT c.id, u.id, u.username, c.body, c.created_at, c.updated_at FROM task_comments c LEFT JOIN users u ON u.id = c.author_id WHERE c.task_id = ? ORDER BY c.created_at, c.id", taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	comments := make([]TaskComment, 0)
	for rows.Next() {
		var comment TaskComment
		var authorID sql.NullInt64
		var authorName sql.NullString
		if err := rows.Scan(&comment.ID, &authorID, &authorName, &comment.Body, &comment.CreatedAt, &comment.UpdatedAt); err != nil {
			return nil, err
		}
		if authorID.Valid {
			comment.Author = &TaskAssignee{ID: uint64(authorID.Int64), Username: authorName.String}
		}
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}

func commentBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	var input commentPayload
	if !decodeJSON(w, r, &input) {
		return "", false
	}
	body := strings.TrimSpace(input.Body)
	if length := utf8.RuneCountInString(body); length < 1 || length > 5000 {
		writeError(w, http.StatusBadRequest, "comment must be 1 to 5000 characters")
		return "", false
	}
	return body, true
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	_, _, taskID, ok := h.authorizeTask(w, r, false)
	if !ok {
		return
	}
	body, ok := commentBody(w, r)
	if !ok {
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate comment id")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO task_comments (id, task_id, author_id, body) VALUES (?, ?, ?, ?)", id, taskID, user.ID, body); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create comment")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) commentAuthor(w http.ResponseWriter, r *http.Request, projectID string) (string, uint64, bool) {
	commentID, ok := pathID(w, r, "commentID")
	if !ok {
		return "", 0, false
	}
	var authorID sql.NullInt64
	err := h.db.QueryRowContext(r.Context(), "SELECT c.author_id FROM task_comments c JOIN project_tasks t ON t.id = c.task_id WHERE c.id = ? AND t.project_id = ?", commentID, projectID).Scan(&authorID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "comment not found")
		return "", 0, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check comment")
		return "", 0, false
	}
	return commentID, uint64(authorID.Int64), true
}

func (h *Handler) updateComment(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	_, projectID, ok := h.authorizeProjectAccess(w, r, false)
	if !ok {
		return
	}
	commentID, authorID, ok := h.commentAuthor(w, r, projectID)
	if !ok {
		return
	}
	if authorID != user.ID {
		writeError(w, http.StatusForbidden, "only the author can edit this comment")
		return
	}
	body, ok := commentBody(w, r)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE task_comments SET body = ? WHERE id = ?", body, commentID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update comment")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, projectID, ok := h.authorizeProjectAccess(w, r, false)
	if !ok {
		return
	}
	commentID, authorID, ok := h.commentAuthor(w, r, projectID)
	if !ok {
		return
	}
	if authorID != user.ID && !h.authorize(w, r, user, workspaceID, true) {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "DELETE FROM task_comments WHERE id = ?", commentID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete comment")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
