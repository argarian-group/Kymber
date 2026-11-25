package workspaces

import (
	"net/http"
	"strconv"
	"time"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/permissions"
)

type auditEvent struct {
	ID        uint64    `json:"id,string"`
	Username  string    `json:"username"`
	Method    string    `json:"method"`
	Route     string    `json:"route"`
	Path      string    `json:"path"`
	Workspace bool      `json:"workspace"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *Handler) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.WorkspaceManage) {
		return
	}
	before, err := strconv.ParseUint(r.URL.Query().Get("before"), 10, 64)
	if err != nil {
		before = 1<<63 - 1
	}
	rows, err := h.db.QueryContext(r.Context(), `SELECT a.id, COALESCE(u.username, ''), a.method, a.route, a.path, a.workspace_id IS NOT NULL, a.created_at
		FROM audit_events a LEFT JOIN users u ON u.id = a.user_id
		WHERE (a.workspace_id = ? OR (? AND a.workspace_id IS NULL)) AND a.id < ?
		ORDER BY a.id DESC LIMIT 50`, workspaceID, user.IsRoot, before)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list audit events")
		return
	}
	defer rows.Close()
	events := make([]auditEvent, 0)
	for rows.Next() {
		var event auditEvent
		if err := rows.Scan(&event.ID, &event.Username, &event.Method, &event.Route, &event.Path, &event.Workspace, &event.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read audit events")
			return
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read audit events")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]auditEvent{"events": events})
}
