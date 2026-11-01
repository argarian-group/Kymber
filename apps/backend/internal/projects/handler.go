package projects

import (
	"database/sql"
	"net/http"
	"time"

	"kymber/backend/internal/auth"
)

type Handler struct {
	db *sql.DB
}

type Workspace struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Project struct {
	ID          uint64    `json:"id"`
	WorkspaceID uint64    `json:"workspace_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	CreatedAt   time.Time `json:"created_at"`
}

func New(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/workspaces", auth.Authenticate(h.db, http.HandlerFunc(h.listWorkspaces)))
	mux.Handle("GET /api/workspaces/{workspaceID}/projects", auth.Authenticate(h.db, http.HandlerFunc(h.listProjects)))
	mux.Handle("POST /api/workspaces/{workspaceID}/projects", auth.Authenticate(h.db, http.HandlerFunc(h.createProject)))
	mux.Handle("GET /api/workspaces/{workspaceID}/projects/{projectID}", auth.Authenticate(h.db, http.HandlerFunc(h.getProject)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/projects/{projectID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateProject)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/projects/{projectID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteProject)))
}
