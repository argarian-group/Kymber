package auth

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"kymber/backend/internal/logger"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func recordAudit(db *sql.DB, r *http.Request, user User, status int) {
	var workspaceID any
	if id := r.PathValue("workspaceID"); id != "" {
		workspaceID = id
	}
	route := strings.TrimPrefix(r.Pattern, r.Method+" ")
	_, err := db.ExecContext(context.WithoutCancel(r.Context()), "INSERT INTO audit_events (workspace_id, user_id, method, route, path, status) SELECT ?, ?, ?, ?, ?, ? FROM DUAL WHERE ? IS NULL OR EXISTS(SELECT 1 FROM workspaces WHERE id = ?)", workspaceID, user.ID, r.Method, truncate(route, 255), truncate(r.URL.Path, 512), status, workspaceID, workspaceID)
	if err != nil {
		logger.Warn("could not record audit event: %v", err)
	}
}

func truncate(value string, limit int) string {
	if len(value) > limit {
		return value[:limit]
	}
	return value
}
