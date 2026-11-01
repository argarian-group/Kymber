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
	ID        uint64    `json:"id,string"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Project struct {
	ID          uint64    `json:"id,string"`
	WorkspaceID uint64    `json:"workspace_id,string"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	CreatedAt   time.Time `json:"created_at"`
}

func New(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/workspaces", auth.Authenticate(h.db, http.HandlerFunc(h.listWorkspaces)))
	mux.Handle("POST /api/workspaces", auth.Authenticate(h.db, http.HandlerFunc(h.createWorkspace)))
	mux.Handle("GET /api/workspaces/{workspaceID}", auth.Authenticate(h.db, http.HandlerFunc(h.getWorkspace)))
	mux.Handle("PUT /api/workspaces/{workspaceID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateWorkspace)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteWorkspace)))
	mux.Handle("GET /api/workspaces/{workspaceID}/members", auth.Authenticate(h.db, http.HandlerFunc(h.listMembers)))
	mux.Handle("POST /api/workspaces/{workspaceID}/members", auth.Authenticate(h.db, http.HandlerFunc(h.addMember)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/members/{userID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateMember)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/members/{userID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteMember)))
	mux.Handle("GET /api/workspaces/{workspaceID}/projects", auth.Authenticate(h.db, http.HandlerFunc(h.listProjects)))
	mux.Handle("POST /api/workspaces/{workspaceID}/projects", auth.Authenticate(h.db, http.HandlerFunc(h.createProject)))
	mux.Handle("GET /api/workspaces/{workspaceID}/projects/{projectID}", auth.Authenticate(h.db, http.HandlerFunc(h.getProject)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/projects/{projectID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateProject)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/projects/{projectID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteProject)))
}
