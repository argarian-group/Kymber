package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"kymber/backend/internal/database"
)

const apiTokenPrefix = "kym_"

type apiToken struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

func userFromAPIToken(db *sql.DB, r *http.Request, header string) (User, error) {
	value, ok := strings.CutPrefix(header, "Bearer "+apiTokenPrefix)
	if !ok {
		return User{}, errUnauthenticated
	}
	token, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(token) != 32 {
		return User{}, errUnauthenticated
	}
	digest := sha256.Sum256(token)
	var user User
	err = db.QueryRowContext(r.Context(), "SELECT u.id, u.username, u.email, u.is_root FROM users u JOIN api_tokens t ON t.user_id = u.id WHERE t.token_hash = ?", digest[:]).Scan(&user.ID, &user.Username, &user.Email, &user.IsRoot)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, errUnauthenticated
	}
	if err == nil {
		_, _ = db.ExecContext(r.Context(), "UPDATE api_tokens SET last_used_at = UTC_TIMESTAMP(6) WHERE token_hash = ? AND (last_used_at IS NULL OR last_used_at < UTC_TIMESTAMP(6) - INTERVAL 1 MINUTE)", digest[:])
	}
	return user, err
}

// sessionUser refuses API tokens so a leaked token cannot mint or revoke tokens.
func (h *Handler) sessionUser(w http.ResponseWriter, r *http.Request) (User, bool) {
	if r.Header.Get("Authorization") != "" {
		writeError(w, http.StatusForbidden, "API tokens can only be managed from a signed-in session")
		return User{}, false
	}
	return h.currentUser(w, r)
}

func (h *Handler) listAPITokens(w http.ResponseWriter, r *http.Request) {
	user, ok := h.sessionUser(w, r)
	if !ok {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT id, name, created_at, last_used_at FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC", user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list API tokens")
		return
	}
	defer rows.Close()
	tokens := make([]apiToken, 0)
	for rows.Next() {
		var token apiToken
		if err := rows.Scan(&token.ID, &token.Name, &token.CreatedAt, &token.LastUsedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read API tokens")
			return
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read API tokens")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]apiToken{"tokens": tokens})
}

func (h *Handler) createAPIToken(w http.ResponseWriter, r *http.Request) {
	user, ok := h.sessionUser(w, r)
	if !ok {
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 80 {
		writeError(w, http.StatusBadRequest, "name must be 1 to 80 characters")
		return
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate API token")
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate API token")
		return
	}
	digest := sha256.Sum256(secret)
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO api_tokens (id, user_id, name, token_hash) VALUES (?, ?, ?, ?)", id, user.ID, input.Name, digest[:]); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save API token")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": apiToken{ID: id, Name: input.Name, CreatedAt: time.Now().UTC()}, "secret": apiTokenPrefix + base64.RawURLEncoding.EncodeToString(secret)})
}

func (h *Handler) deleteAPIToken(w http.ResponseWriter, r *http.Request) {
	user, ok := h.sessionUser(w, r)
	if !ok {
		return
	}
	result, err := h.db.ExecContext(r.Context(), "DELETE FROM api_tokens WHERE id = ? AND user_id = ?", r.PathValue("tokenID"), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not revoke API token")
		return
	}
	if deleted, _ := result.RowsAffected(); deleted == 0 {
		writeError(w, http.StatusNotFound, "API token not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
