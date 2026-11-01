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
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Role      string            `json:"role"`
	CreatedAt time.Time         `json:"created_at"`
	CreatedBy *WorkspaceCreator `json:"created_by"`
}

type WorkspaceCreator struct {
	ID       uint64 `json:"id,string"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type Project struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	CreatedAt   time.Time `json:"created_at"`
}

func New(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/database-templates", auth.Authenticate(h.db, http.HandlerFunc(h.listDatabaseTemplates)))
	mux.Handle("GET /api/workspaces", auth.Authenticate(h.db, http.HandlerFunc(h.listWorkspaces)))
	mux.Handle("POST /api/workspaces", auth.Authenticate(h.db, http.HandlerFunc(h.createWorkspace)))
	mux.Handle("GET /api/workspaces/{workspaceID}", auth.Authenticate(h.db, http.HandlerFunc(h.getWorkspace)))
	mux.Handle("PUT /api/workspaces/{workspaceID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateWorkspace)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteWorkspace)))
	mux.Handle("GET /api/workspaces/{workspaceID}/members", auth.Authenticate(h.db, http.HandlerFunc(h.listMembers)))
	mux.Handle("POST /api/workspaces/{workspaceID}/members", auth.Authenticate(h.db, http.HandlerFunc(h.addMember)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/members/{userID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateMember)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/members/{userID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteMember)))
	mux.Handle("POST /api/workspaces/{workspaceID}/invitations", auth.Authenticate(h.db, http.HandlerFunc(h.createInvitation)))
	mux.Handle("GET /api/workspaces/{workspaceID}/projects", auth.Authenticate(h.db, http.HandlerFunc(h.listProjects)))
	mux.Handle("POST /api/workspaces/{workspaceID}/projects", auth.Authenticate(h.db, http.HandlerFunc(h.createProject)))
	mux.Handle("GET /api/workspaces/{workspaceID}/projects/{projectID}", auth.Authenticate(h.db, http.HandlerFunc(h.getProject)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/projects/{projectID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateProject)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/projects/{projectID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteProject)))
	mux.Handle("GET /api/workspaces/{workspaceID}/projects/{projectID}/services", auth.Authenticate(h.db, http.HandlerFunc(h.listServices)))
	mux.Handle("POST /api/workspaces/{workspaceID}/projects/{projectID}/services", auth.Authenticate(h.db, http.HandlerFunc(h.createService)))
	mux.Handle("GET /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}", auth.Authenticate(h.db, http.HandlerFunc(h.getService)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateService)))
	mux.Handle("POST /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}/{action}", auth.Authenticate(h.db, http.HandlerFunc(h.serviceAction)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/projects/{projectID}/services/{serviceID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteService)))
	mux.Handle("GET /api/servers", auth.Authenticate(h.db, http.HandlerFunc(h.listServers)))
	mux.Handle("POST /api/servers", auth.Authenticate(h.db, http.HandlerFunc(h.createServer)))
	mux.Handle("PUT /api/servers/{serverID}", auth.Authenticate(h.db, http.HandlerFunc(h.renameServer)))
	mux.Handle("POST /api/servers/{serverID}/token", auth.Authenticate(h.db, http.HandlerFunc(h.rotateServerToken)))
	mux.Handle("DELETE /api/servers/{serverID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteServer)))
}
