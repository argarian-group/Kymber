package projects

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
)

const serverStatusSQL = `CASE
	WHEN s.last_seen IS NULL THEN 'pending'
	WHEN s.last_seen >= UTC_TIMESTAMP(6) - INTERVAL 30 SECOND OR EXISTS(SELECT 1 FROM server_tasks t WHERE t.server_id = s.id AND t.status = 'running') THEN 'online'
	ELSE 'offline'
END`

type serverInfo struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	LastSeen  *time.Time `json:"last_seen"`
	CreatedAt time.Time  `json:"created_at"`
	Apps      int        `json:"apps"`
}

func (h *Handler) requireRoot(w http.ResponseWriter, r *http.Request) bool {
	user, _ := auth.UserFromContext(r)
	if !user.IsRoot {
		writeError(w, http.StatusForbidden, "root access required")
		return false
	}
	return true
}

func (h *Handler) listServers(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT s.id, s.name, "+serverStatusSQL+", s.last_seen, s.created_at, (SELECT COUNT(*) FROM database_services d WHERE d.server_id = s.id AND d.status <> 'deleted') FROM servers s ORDER BY s.created_at")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list servers")
		return
	}
	defer rows.Close()
	items := make([]serverInfo, 0)
	for rows.Next() {
		var s serverInfo
		if err := rows.Scan(&s.ID, &s.Name, &s.Status, &s.LastSeen, &s.CreatedAt, &s.Apps); err != nil {
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

func (h *Handler) listDeployTargets(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, true) {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT s.id, s.name FROM servers s WHERE "+serverStatusSQL+" = 'online' ORDER BY s.created_at")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list servers")
		return
	}
	defer rows.Close()
	type target struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	targets := make([]target, 0)
	for rows.Next() {
		var item target
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read servers")
			return
		}
		targets = append(targets, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read servers")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]target{"servers": targets})
}

func (h *Handler) requireServerOnline(w http.ResponseWriter, r *http.Request, id string) bool {
	var online bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM servers s WHERE s.id = ? AND "+serverStatusSQL+" = 'online')", id).Scan(&online); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check server")
		return false
	}
	if !online {
		writeError(w, http.StatusConflict, "server is offline")
		return false
	}
	return true
}

func serverName(w http.ResponseWriter, r *http.Request) (string, bool) {
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return "", false
	}
	name := strings.TrimSpace(input.Name)
	if length := utf8.RuneCountInString(name); length < 1 || length > 120 {
		writeError(w, http.StatusBadRequest, "name must be 1 to 120 characters")
		return "", false
	}
	return name, true
}

func (h *Handler) createServer(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	user, _ := auth.UserFromContext(r)
	name, ok := serverName(w, r)
	if !ok {
		return
	}
	token, digest, err := newServerToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate server token")
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate server id")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO servers (id, name, token_hash, created_by) VALUES (?, ?, ?, ?)", id, name, digest[:], user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not register server")
		return
	}
	if err := h.claimLocalToken(r.Context(), token); err != nil {
		_, _ = h.db.ExecContext(r.Context(), "DELETE FROM servers WHERE id = ?", id)
		writeError(w, http.StatusInternalServerError, "could not provision local server token")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"server": serverInfo{ID: id, Name: name, Status: "pending", CreatedAt: time.Now().UTC()}, "token": token})
}

func newServerToken() (string, [32]byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", [32]byte{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, sha256.Sum256([]byte(token)), nil
}

func (h *Handler) renameServer(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	id, ok := pathID(w, r, "serverID")
	if !ok {
		return
	}
	name, ok := serverName(w, r)
	if !ok {
		return
	}
	var exists bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM servers WHERE id = ?)", id).Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read server")
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE servers SET name = ? WHERE id = ?", name, id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not rename server")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) rotateServerToken(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	id, ok := pathID(w, r, "serverID")
	if !ok {
		return
	}
	token, digest, err := newServerToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate server token")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not replace server token")
		return
	}
	defer tx.Rollback()
	var current []byte
	err = tx.QueryRowContext(r.Context(), "SELECT token_hash FROM servers WHERE id = ? FOR UPDATE", id).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not replace server token")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "UPDATE servers SET token_hash = ?, last_seen = NULL, status = 'pending' WHERE id = ?", digest[:], id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not replace server token")
		return
	}
	if isLocalServer(current) {
		if err := writeTokenFile(os.Getenv("KYMBER_SERVER_TOKEN_FILE"), token); err != nil {
			writeError(w, http.StatusInternalServerError, "could not provision local server token")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not replace server token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *Handler) deleteServer(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	id, ok := pathID(w, r, "serverID")
	if !ok {
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete server")
		return
	}
	defer tx.Rollback()
	var apps int
	if err := tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM database_services WHERE server_id = ? AND status <> 'deleted' FOR UPDATE", id).Scan(&apps); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check server apps")
		return
	}
	if apps > 0 {
		writeError(w, http.StatusConflict, "delete this server's apps first")
		return
	}
	for _, query := range []string{"DELETE FROM server_tasks WHERE server_id = ?", "DELETE FROM database_services WHERE server_id = ? AND status = 'deleted'"} {
		if _, err := tx.ExecContext(r.Context(), query, id); err != nil {
			writeError(w, http.StatusInternalServerError, "could not delete server")
			return
		}
	}
	var current []byte
	err = tx.QueryRowContext(r.Context(), "SELECT token_hash FROM servers WHERE id = ?", id).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete server")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "DELETE FROM servers WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete server")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete server")
		return
	}
	if isLocalServer(current) {
		_ = os.Remove(os.Getenv("KYMBER_SERVER_TOKEN_FILE"))
	}
	w.WriteHeader(http.StatusNoContent)
}
