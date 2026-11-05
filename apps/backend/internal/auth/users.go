package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-sql-driver/mysql"
)

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT id, username, email, is_root FROM users ORDER BY username, id")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list users")
		return
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.IsRoot); err != nil {
			writeError(w, http.StatusInternalServerError, "could not list users")
			return
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not list users")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]User{"users": users})
}

func (h *Handler) createManagedUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	var input struct {
		credentials
		IsRoot bool `json:"is_root"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validateCredentials(&input.credentials); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}
	defer tx.Rollback()
	userID, err := createUser(r, tx, input.Username, input.Email, passwordHash, input.IsRoot)
	if err != nil {
		writeManagedUserError(w, err)
		return
	}
	if err := createWorkspace(r, tx, userID, input.Username); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create workspace")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]User{"user": {ID: userID, Username: input.Username, Email: input.Email, IsRoot: input.IsRoot}})
}

func (h *Handler) updateManagedUser(w http.ResponseWriter, r *http.Request) {
	_, ok := h.rootUser(w, r)
	if !ok {
		return
	}
	userID, ok := managedUserID(w, r)
	if !ok {
		return
	}
	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Username = normalizeUsername(input.Username)
	input.Email = normalizeEmail(input.Email)
	if !validUsername(input.Username) {
		writeError(w, http.StatusBadRequest, "username must be 3-32 characters, start with a letter or number, and contain only letters, numbers, _ or -")
		return
	}
	if !validEmail(input.Email) {
		writeError(w, http.StatusBadRequest, "valid email is required")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE users SET username = ?, email = ? WHERE id = ?", input.Username, input.Email, userID); err != nil {
		writeManagedUserError(w, err)
		return
	}
	var user User
	if err := h.db.QueryRowContext(r.Context(), "SELECT id, username, email, is_root FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.IsRoot); err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]User{"user": user})
}

func (h *Handler) deleteManagedUser(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.rootUser(w, r)
	if !ok {
		return
	}
	userID, ok := managedUserID(w, r)
	if !ok {
		return
	}
	if actor.ID == userID {
		writeError(w, http.StatusForbidden, "cannot delete your own account")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete user")
		return
	}
	defer tx.Rollback()
	var targetRoot bool
	if err := tx.QueryRowContext(r.Context(), "SELECT is_root FROM users WHERE id = ? FOR UPDATE", userID).Scan(&targetRoot); errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "could not find user")
		return
	}
	if targetRoot {
		var rootCount int
		if err := tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM users WHERE is_root = TRUE").Scan(&rootCount); err != nil {
			writeError(w, http.StatusInternalServerError, "could not verify root accounts")
			return
		}
		if rootCount <= 1 {
			writeError(w, http.StatusForbidden, "cannot delete the last root account")
			return
		}
	}
	if _, err := tx.ExecContext(r.Context(), "UPDATE workspaces SET created_by = NULL WHERE created_by = ?", userID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update user workspaces")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "DELETE FROM workspace_invitations WHERE invited_by = ?", userID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete user invitations")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "UPDATE servers SET created_by = ? WHERE created_by = ?", actor.ID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update user servers")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "UPDATE database_services SET created_by = ? WHERE created_by = ?", actor.ID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update user services")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "DELETE FROM users WHERE id = ?", userID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete user")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) rootUser(w http.ResponseWriter, r *http.Request) (User, bool) {
	user, ok := h.currentUser(w, r)
	if !ok || !user.IsRoot {
		if ok {
			writeError(w, http.StatusForbidden, "root access required")
		}
		return User{}, false
	}
	return user, true
}

func managedUserID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(r.PathValue("userID"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return 0, false
	}
	return id, true
}

func writeManagedUserError(w http.ResponseWriter, err error) {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		writeError(w, http.StatusConflict, "email or username is already registered")
		return
	}
	writeError(w, http.StatusInternalServerError, "could not save user")
}
