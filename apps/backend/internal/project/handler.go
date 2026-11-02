package project

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
)

type Handler struct{ db *sql.DB }

func New(db *sql.DB) *Handler { return &Handler{db: db} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	for _, route := range []struct {
		pattern string
		handler http.HandlerFunc
	}{
		{"GET /api/workspaces/{workspaceID}/projects", h.listProjects},
		{"POST /api/workspaces/{workspaceID}/projects", h.createProject},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}", h.getProject},
		{"PUT /api/workspaces/{workspaceID}/projects/{projectID}", h.updateProject},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}", h.deleteProject},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/logo", h.getProjectLogo},
		{"PUT /api/workspaces/{workspaceID}/projects/{projectID}/logo", h.uploadProjectLogo},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}/logo", h.deleteProjectLogo},
		{"GET /api/workspaces/{workspaceID}/projects/{projectID}/environments", h.listEnvironments},
		{"POST /api/workspaces/{workspaceID}/projects/{projectID}/environments", h.createEnvironment},
		{"DELETE /api/workspaces/{workspaceID}/projects/{projectID}/environments/{environmentID}", h.deleteEnvironment},
	} {
		mux.Handle(route.pattern, auth.Authenticate(h.db, route.handler))
	}
}

type Project struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	CreatedAt   time.Time `json:"created_at"`
	LogoVersion string    `json:"logo_version,omitempty"`
}

type projectPayload struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

var entityIDPattern = regexp.MustCompile(`^[0-9a-f]{6}$`)

func pathID(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	id := r.PathValue(name)
	if !entityIDPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return "", false
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
func projectInput(w http.ResponseWriter, r *http.Request) (projectPayload, bool) {
	var input projectPayload
	if !decodeJSON(w, r, &input) {
		return input, false
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	if n := utf8.RuneCountInString(input.Name); n < 1 || n > 120 {
		writeError(w, http.StatusBadRequest, "name must be 1 to 120 characters")
		return input, false
	}
	if len(input.Slug) < 1 || len(input.Slug) > 120 {
		writeError(w, http.StatusBadRequest, "slug must be 1 to 120 characters")
		return input, false
	}
	for _, c := range []byte(input.Slug) {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			writeError(w, http.StatusBadRequest, "slug may contain only lowercase letters, numbers, and hyphens")
			return input, false
		}
	}
	return input, true
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, user auth.User, workspaceID string, permission string) bool {
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
	_, granted, err := permissions.Lookup(r.Context(), h.db, workspaceID, user.ID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "workspace not found")
		return false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify workspace access")
		return false
	}
	if permission != "" && !granted.Has(permission) {
		writeError(w, http.StatusForbidden, "you do not have permission for this action")
		return false
	}
	return true
}

func createDefaultTaskLists(ctx context.Context, tx *sql.Tx, projectID string) error {
	for i, item := range []struct {
		name string
		done bool
	}{{"To do", false}, {"In progress", false}, {"In review", false}, {"Done", true}} {
		id, err := database.NewShortID()
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO task_lists (id, project_id, name, is_done, position) VALUES (?, ?, ?, ?, ?)", id, projectID, item.name, item.done, i+1); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) authorizeProject(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) bool {
	var exists bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM projects WHERE id = ? AND workspace_id = ?)", projectID, workspaceID).Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify project")
		return false
	}
	if !exists {
		writeError(w, http.StatusNotFound, "project not found")
		return false
	}
	return true
}
