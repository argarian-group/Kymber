package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"

	"github.com/go-sql-driver/mysql"
)

type credentials struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func validateCredentials(input *credentials) error {
	input.Username = normalizeUsername(input.Username)
	if !validUsername(input.Username) {
		return errors.New("username must be 3-32 characters, start with a letter or number, and contain only letters, numbers, _ or -")
	}
	input.Email = normalizeEmail(input.Email)
	if !validEmail(input.Email) {
		return errors.New("valid email is required")
	}
	return validatePassword(input.Password)
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func validUsername(username string) bool {
	if len(username) < 3 || len(username) > 32 {
		return false
	}
	for i, char := range []byte(username) {
		valid := char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || i > 0 && (char == '_' || char == '-')
		if !valid {
			return false
		}
	}
	return true
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request must contain a single JSON value")
		return false
	}
	return true
}

func createUser(r *http.Request, tx *sql.Tx, username, email, passwordHash string, root bool) (uint64, error) {
	result, err := tx.ExecContext(r.Context(), "INSERT INTO users (username, email, password_hash, is_root) VALUES (?, ?, ?, ?)", username, email, passwordHash, root)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return uint64(id), err
}

func createWorkspace(r *http.Request, tx *sql.Tx, userID uint64) error {
	result, err := tx.ExecContext(r.Context(), "INSERT INTO workspaces (name) VALUES ('Personal')")
	if err != nil {
		return err
	}
	workspaceID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(r.Context(), "INSERT INTO workspace_members (workspace_id, user_id, role) VALUES (?, ?, 'owner')", workspaceID, userID)
	return err
}

func writeUserError(w http.ResponseWriter, err error) {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		writeError(w, http.StatusConflict, "email or username is already registered")
		return
	}
	writeError(w, http.StatusInternalServerError, "could not create user")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
