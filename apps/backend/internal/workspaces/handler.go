package workspaces

import (
	"database/sql"
	"net/http"
	"time"

	"kymber/backend/internal/auth"
)

type Handler struct{ db *sql.DB }

func New(db *sql.DB) *Handler { return &Handler{db: db} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	for _, route := range []struct {
		method  string
		pattern string
		handler http.HandlerFunc
	}{
		{"GET", "/api/workspaces", h.listWorkspaces},
		{"POST", "/api/workspaces", h.createWorkspace},
		{"GET", "/api/workspaces/{workspaceID}", h.getWorkspace},
		{"PUT", "/api/workspaces/{workspaceID}", h.updateWorkspace},
		{"DELETE", "/api/workspaces/{workspaceID}", h.deleteWorkspace},
		{"GET", "/api/workspaces/{workspaceID}/members", h.listMembers},
		{"POST", "/api/workspaces/{workspaceID}/members", h.addMember},
		{"PUT", "/api/workspaces/{workspaceID}/members/{userID}", h.updateMember},
		{"DELETE", "/api/workspaces/{workspaceID}/members/{userID}", h.deleteMember},
		{"POST", "/api/workspaces/{workspaceID}/invitations", h.createInvitation},
		{"GET", "/api/workspaces/{workspaceID}/roles", h.listRoles},
		{"POST", "/api/workspaces/{workspaceID}/roles", h.createRole},
		{"PUT", "/api/workspaces/{workspaceID}/roles/{roleName}", h.updateRole},
		{"DELETE", "/api/workspaces/{workspaceID}/roles/{roleName}", h.deleteRole},
	} {
		mux.Handle(route.method+" "+route.pattern, auth.Authenticate(h.db, route.handler))
	}
}

type Workspace struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Role        string            `json:"role"`
	Permissions []string          `json:"permissions"`
	CreatedAt   time.Time         `json:"created_at"`
	CreatedBy   *WorkspaceCreator `json:"created_by"`
}

type WorkspaceCreator struct {
	ID       uint64 `json:"id,string"`
	Username string `json:"username"`
	Email    string `json:"email"`
}
