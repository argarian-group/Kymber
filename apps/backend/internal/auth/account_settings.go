package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/go-sql-driver/mysql"
)

func (h *Handler) checkCurrentPassword(w http.ResponseWriter, r *http.Request, userID uint64, password string) bool {
	if len(password) == 0 || len(password) > 256 {
		writeError(w, http.StatusForbidden, "current password is incorrect")
		return false
	}
	var passwordHash string
	if err := h.db.QueryRowContext(r.Context(), "SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash); err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify password")
		return false
	}
	if !verifyPassword(passwordHash, password) {
		writeError(w, http.StatusForbidden, "current password is incorrect")
		return false
	}
	return true
}

func (h *Handler) getAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	hash := sha256.Sum256([]byte(normalizeEmail(user.Email)))
	writeJSON(w, http.StatusOK, map[string]string{"avatar_url": "https://www.gravatar.com/avatar/" + hex.EncodeToString(hash[:]) + "?d=404&s=56"})
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var input struct {
		Username        string `json:"username"`
		Email           string `json:"email"`
		CurrentPassword string `json:"current_password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Username = normalizeUsername(input.Username)
	if !validUsername(input.Username) {
		writeError(w, http.StatusBadRequest, "username must be 3-32 characters, start with a letter or number, and contain only letters, numbers, _ or -")
		return
	}
	input.Email = normalizeEmail(input.Email)
	if !validEmail(input.Email) {
		writeError(w, http.StatusBadRequest, "valid email is required")
		return
	}
	if !h.checkCurrentPassword(w, r, user.ID, input.CurrentPassword) {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE users SET username = ?, email = ? WHERE id = ?", input.Username, input.Email, user.ID); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			writeError(w, http.StatusConflict, "email or username is already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not update account")
		return
	}
	user.Username, user.Email = input.Username, input.Email
	writeJSON(w, http.StatusOK, map[string]User{"user": user})
}

func (h *Handler) updatePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validatePassword(input.NewPassword); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !h.checkCurrentPassword(w, r, user.ID, input.CurrentPassword) {
		return
	}
	passwordHash, err := hashPassword(input.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		writeError(w, http.StatusUnauthorized, errUnauthenticated.Error())
		return
	}
	token, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, errUnauthenticated.Error())
		return
	}
	digest := sha256.Sum256(token)
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update password")
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(), "UPDATE users SET password_hash = ? WHERE id = ?", passwordHash, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update password")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "DELETE FROM user_sessions WHERE user_id = ? AND token_hash <> ?", user.ID, digest[:]); err != nil {
		writeError(w, http.StatusInternalServerError, "could not end other sessions")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update password")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
