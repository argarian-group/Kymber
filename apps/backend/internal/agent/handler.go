package agent

import (
	"crypto/sha256"
	"database/sql"
	"net/http"
	"strings"
)

type Handler struct{ db *sql.DB }

func New(db *sql.DB) *Handler { return &Handler{db: db} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/agent/heartbeat", h.heartbeat)
}

func (h *Handler) authenticate(r *http.Request) (string, bool) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		return "", false
	}
	digest := sha256.Sum256([]byte(token))
	var id string
	err := h.db.QueryRowContext(r.Context(), "SELECT id FROM servers WHERE token_hash = ?", digest[:]).Scan(&id)
	return id, err == nil
}

func (h *Handler) heartbeat(w http.ResponseWriter, r *http.Request) {
	id, ok := h.authenticate(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE servers SET status = 'online', last_seen = UTC_TIMESTAMP(6) WHERE id = ?", id); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
