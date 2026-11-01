package routes

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"kymber/backend/internal/auth"
)

func New(db *sql.DB, secureOnboard bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	auth.New(db, secureOnboard).RegisterRoutes(mux)
	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
