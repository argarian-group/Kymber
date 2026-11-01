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
	LogoVersion string    `json:"logo_version,omitempty"`
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
	mux.Handle("GET /api/workspaces/{workspaceID}/servers", auth.Authenticate(h.db, http.HandlerFunc(h.listDeployTargets)))
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
	mux.Handle("GET /api/workspaces/{workspaceID}/projects/{projectID}/logo", auth.Authenticate(h.db, http.HandlerFunc(h.getProjectLogo)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/projects/{projectID}/logo", auth.Authenticate(h.db, http.HandlerFunc(h.uploadProjectLogo)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/projects/{projectID}/logo", auth.Authenticate(h.db, http.HandlerFunc(h.deleteProjectLogo)))
	mux.Handle("GET /api/workspaces/{workspaceID}/projects/{projectID}/board", auth.Authenticate(h.db, http.HandlerFunc(h.getBoard)))
	mux.Handle("POST /api/workspaces/{workspaceID}/projects/{projectID}/tasks", auth.Authenticate(h.db, http.HandlerFunc(h.createTask)))
	mux.Handle("GET /api/workspaces/{workspaceID}/projects/{projectID}/tasks/{taskID}", auth.Authenticate(h.db, http.HandlerFunc(h.getTask)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/projects/{projectID}/tasks/{taskID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateTask)))
	mux.Handle("POST /api/workspaces/{workspaceID}/projects/{projectID}/tasks/{taskID}/move", auth.Authenticate(h.db, http.HandlerFunc(h.moveTask)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/projects/{projectID}/tasks/{taskID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteTask)))
	mux.Handle("POST /api/workspaces/{workspaceID}/projects/{projectID}/tasks/{taskID}/checklists", auth.Authenticate(h.db, http.HandlerFunc(h.createChecklist)))
	mux.Handle("POST /api/workspaces/{workspaceID}/projects/{projectID}/tasks/{taskID}/comments", auth.Authenticate(h.db, http.HandlerFunc(h.createComment)))
	mux.Handle("POST /api/workspaces/{workspaceID}/projects/{projectID}/task-lists", auth.Authenticate(h.db, http.HandlerFunc(h.createTaskList)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/projects/{projectID}/task-lists/order", auth.Authenticate(h.db, http.HandlerFunc(h.reorderTaskLists)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/projects/{projectID}/task-lists/{listID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateTaskList)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/projects/{projectID}/task-lists/{listID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteTaskList)))
	mux.Handle("POST /api/workspaces/{workspaceID}/projects/{projectID}/task-labels", auth.Authenticate(h.db, http.HandlerFunc(h.createTaskLabel)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/projects/{projectID}/task-labels/{labelID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateTaskLabel)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/projects/{projectID}/task-labels/{labelID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteTaskLabel)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/projects/{projectID}/task-checklists/{checklistID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateChecklist)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/projects/{projectID}/task-checklists/{checklistID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteChecklist)))
	mux.Handle("POST /api/workspaces/{workspaceID}/projects/{projectID}/task-checklists/{checklistID}/items", auth.Authenticate(h.db, http.HandlerFunc(h.createChecklistItem)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/projects/{projectID}/task-checklist-items/{itemID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateChecklistItem)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/projects/{projectID}/task-checklist-items/{itemID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteChecklistItem)))
	mux.Handle("PUT /api/workspaces/{workspaceID}/projects/{projectID}/task-comments/{commentID}", auth.Authenticate(h.db, http.HandlerFunc(h.updateComment)))
	mux.Handle("DELETE /api/workspaces/{workspaceID}/projects/{projectID}/task-comments/{commentID}", auth.Authenticate(h.db, http.HandlerFunc(h.deleteComment)))
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
