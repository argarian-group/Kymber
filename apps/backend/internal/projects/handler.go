package projects

import (
	"database/sql"
	"net/http"

	"kymber/backend/internal/project"
	"kymber/backend/internal/servers"
	"kymber/backend/internal/services"
	"kymber/backend/internal/tasks"
	"kymber/backend/internal/workspaces"
)

type Handler struct {
	db *sql.DB
}

func New(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	workspaces.New(h.db).RegisterRoutes(mux)
	servers.New(h.db).RegisterRoutes(mux)
	tasks.New(h.db).RegisterRoutes(mux)
	services.New(h.db).RegisterRoutes(mux)
	project.New(h.db).RegisterRoutes(mux)
}
