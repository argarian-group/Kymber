package auth

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
)

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var input credentials
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validateCredentials(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var canRegister bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT registration_enabled FROM platform_settings WHERE id = 1").Scan(&canRegister); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check registration setting")
		return
	}
	if !canRegister {
		writeError(w, http.StatusForbidden, "registration is disabled")
		return
	}
	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start registration")
		return
	}
	defer tx.Rollback()
	var enabled bool
	if err := tx.QueryRowContext(r.Context(), "SELECT registration_enabled FROM platform_settings WHERE id = 1 FOR UPDATE").Scan(&enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check registration setting")
		return
	}
	if !enabled {
		writeError(w, http.StatusForbidden, "registration is disabled")
		return
	}
	userID, err := createUser(r, tx, input.Username, input.Email, passwordHash, false)
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := createWorkspace(r, tx, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create workspace")
		return
	}
	token, expiresAt, err := createSession(r, tx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not complete registration")
		return
	}
	setSessionCookie(w, token, expiresAt)
	writeJSON(w, http.StatusCreated, map[string]User{"user": {ID: userID, Username: input.Username, Email: input.Email}})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var input credentials
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Email = normalizeEmail(input.Email)
	if !validEmail(input.Email) || len(input.Password) == 0 || len(input.Password) > 256 {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	var user User
	var passwordHash string
	err := h.db.QueryRowContext(r.Context(), "SELECT id, username, email, is_root, password_hash FROM users WHERE email = ?", input.Email).Scan(&user.ID, &user.Username, &user.Email, &user.IsRoot, &passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not look up user")
		return
	}
	if !verifyPassword(passwordHash, input.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	defer tx.Rollback()
	token, expiresAt, err := createSession(r, tx, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	setSessionCookie(w, token, expiresAt)
	writeJSON(w, http.StatusOK, map[string]User{"user": user})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		if token, err := base64.RawURLEncoding.DecodeString(cookie.Value); err == nil && len(token) == 32 {
			digest := sha256.Sum256(token)
			if _, err := h.db.ExecContext(r.Context(), "DELETE FROM user_sessions WHERE token_hash = ?", digest[:]); err != nil {
				writeError(w, http.StatusInternalServerError, "could not end session")
				return
			}
		}
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if ok {
		writeJSON(w, http.StatusOK, map[string]User{"user": user})
	}
}
func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) (User, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return User{}, false
	}
	token, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil || len(token) != 32 {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return User{}, false
	}
	digest := sha256.Sum256(token)
	var user User
	err = h.db.QueryRowContext(r.Context(), `SELECT u.id, u.username, u.email, u.is_root FROM users u JOIN user_sessions s ON s.user_id = u.id WHERE s.token_hash = ? AND s.expires_at > UTC_TIMESTAMP(6)`, digest[:]).Scan(&user.ID, &user.Username, &user.Email, &user.IsRoot)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return User{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify session")
		return User{}, false
	}
	return user, true
}
