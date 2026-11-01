package projects

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
)

type serverInfo struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	LastSeen  *time.Time `json:"last_seen"`
	CreatedAt time.Time  `json:"created_at"`
}

func (h *Handler) listServers(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	if !user.IsRoot {
		writeError(w, http.StatusForbidden, "root access required")
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT id, name, status, last_seen, created_at FROM servers ORDER BY created_at")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list servers")
		return
	}
	defer rows.Close()
	items := make([]serverInfo, 0)
	for rows.Next() {
		var s serverInfo
		if err := rows.Scan(&s.ID, &s.Name, &s.Status, &s.LastSeen, &s.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read servers")
			return
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read servers")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]serverInfo{"servers": items})
}

func (h *Handler) createServer(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	if !user.IsRoot {
		writeError(w, http.StatusForbidden, "root access required")
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if len(input.Name) < 1 || len(input.Name) > 120 {
		writeError(w, http.StatusBadRequest, "name must be 1 to 120 characters")
		return
	}
	var count int
	if err := h.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM servers").Scan(&count); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check servers")
		return
	}
	if count > 0 {
		writeError(w, http.StatusConflict, "only one local server is supported in this version")
		return
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate server token")
		return
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate server id")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO servers (id, name, token_hash, created_by) VALUES (?, ?, ?, ?)", id, input.Name, digest[:], user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not register server")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"server": serverInfo{ID: id, Name: input.Name, Status: "pending", CreatedAt: time.Now().UTC()}, "token": token})
}
