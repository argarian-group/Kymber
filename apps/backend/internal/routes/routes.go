package routes

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"kymber/backend/internal/agent"
	"kymber/backend/internal/auth"
	"kymber/backend/internal/middleware"
	"kymber/backend/internal/monitoring"
	"kymber/backend/internal/projects"
)

func New(db *sql.DB, secureOnboard bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	auth.New(db, secureOnboard).RegisterRoutes(mux)
	projects.New(db).RegisterRoutes(mux)
	agent.New(db).RegisterRoutes(mux)
	monitoring.New(db).RegisterRoutes(mux)
	return middleware.RateLimit(mux)
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
