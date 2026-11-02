package tasks

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"kymber/backend/internal/auth"
)

type Handler struct{ db *sql.DB }

func New(db *sql.DB) *Handler { return &Handler{db: db} }
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	for _, route := range []struct {
		pattern string
		handler http.HandlerFunc
	}{
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/board", h.getBoard},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/tasks", h.createTask},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/tasks/{taskID}", h.getTask},
		{"PUT /api/workspaces/{workspaceID}/projects/{projectID}/tasks/{taskID}", h.updateTask},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/tasks/{taskID}/move", h.moveTask},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}/tasks/{taskID}", h.deleteTask},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/tasks/{taskID}/checklists", h.createChecklist},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/tasks/{taskID}/comments", h.createComment},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/task-lists", h.createTaskList},
		{"PUT /api/workspaces/{workspaceID}/projects/{projectID}/task-lists/order", h.reorderTaskLists},
		{"PUT /api/workspaces/{workspaceID}/projects/{projectID}/task-lists/{listID}", h.updateTaskList},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}/task-lists/{listID}", h.deleteTaskList},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/task-labels", h.createTaskLabel},
		{"PUT /api/workspaces/{workspaceID}/projects/{projectID}/task-labels/{labelID}", h.updateTaskLabel},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}/task-labels/{labelID}", h.deleteTaskLabel},
		{"PUT /api/workspaces/{workspaceID}/projects/{projectID}/task-checklists/{checklistID}", h.updateChecklist},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}/task-checklists/{checklistID}", h.deleteChecklist},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/task-checklists/{checklistID}/items", h.createChecklistItem},
		{"PUT /api/workspaces/{workspaceID}/projects/{projectID}/task-checklist-items/{itemID}", h.updateChecklistItem},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}/task-checklist-items/{itemID}", h.deleteChecklistItem},
		{"PUT /api/workspaces/{workspaceID}/projects/{projectID}/task-comments/{commentID}", h.updateComment},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}/task-comments/{commentID}", h.deleteComment},
	} {
		mux.Handle(route.pattern, auth.Authenticate(h.db, route.handler))
	}
}

var idPattern = regexp.MustCompile(`^[0-9a-f]{6}$`)

func pathID(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	id := r.PathValue(name)
	if !idPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return "", false
	}
	return id, true
}
func pathUserID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(r.PathValue("userID"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, "invalid userID")
		return 0, false
	}
	return id, true
}
func decodeJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request must contain a single JSON value")
		return false
	}
	return true
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, user auth.User, workspaceID string, write bool) bool {
	if user.IsRoot {
		var exists bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM workspaces WHERE id = ?)", workspaceID).Scan(&exists); err != nil {
			writeError(w, http.StatusInternalServerError, "could not verify workspace")
			return false
		}
		if !exists {
			writeError(w, http.StatusNotFound, "workspace not found")
			return false
		}
		return true
	}
	var role string
	err := h.db.QueryRowContext(r.Context(), "SELECT role FROM workspace_members WHERE workspace_id = ? AND user_id = ?", workspaceID, user.ID).Scan(&role)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "workspace not found")
		return false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify workspace access")
		return false
	}
	if write && role != "owner" && role != "admin" {
		writeError(w, http.StatusForbidden, "workspace write access required")
		return false
	}
	return true
}
func (h *Handler) authorizeProjectAccess(w http.ResponseWriter, r *http.Request, write bool) (string, string, bool) {
	user, _ := auth.UserFromContext(r)
	wid, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, wid, write) {
		return "", "", false
	}
	pid, ok := pathID(w, r, "projectID")
	if !ok {
		return "", "", false
	}
	var exists bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM projects WHERE id = ? AND workspace_id = ?)", pid, wid).Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify project")
		return "", "", false
	}
	if !exists {
		writeError(w, http.StatusNotFound, "project not found")
		return "", "", false
	}
	return wid, pid, true
}
