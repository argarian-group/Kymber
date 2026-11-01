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
	Email    string `json:"email"`
	Password string `json:"password"`
}

func validateCredentials(input *credentials) error {
	input.Email = normalizeEmail(input.Email)
	if !validEmail(input.Email) {
		return errors.New("valid email is required")
	}
	return validatePassword(input.Password)
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

func createUser(r *http.Request, tx *sql.Tx, email, passwordHash string, root bool) (uint64, error) {
	result, err := tx.ExecContext(r.Context(), "INSERT INTO users (email, password_hash, is_root) VALUES (?, ?, ?)", email, passwordHash, root)
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
		writeError(w, http.StatusConflict, "email is already registered")
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
