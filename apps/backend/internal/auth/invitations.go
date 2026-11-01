package auth

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-sql-driver/mysql"
)

type invitationAcceptance struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	var input invitationAcceptance
	if !decodeJSON(w, r, &input) {
		return
	}
	token, err := base64.RawURLEncoding.DecodeString(input.Token)
	if err != nil || len(token) != 32 {
		writeError(w, http.StatusGone, "invitation is invalid or expired")
		return
	}
	digest := sha256.Sum256(token)
	user, sessionErr := userFromSession(h.db, r)
	if sessionErr != nil && !errors.Is(sessionErr, errUnauthenticated) {
		writeError(w, http.StatusInternalServerError, "could not verify session")
		return
	}
	hasSession := sessionErr == nil
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not accept invitation")
		return
	}
	defer tx.Rollback()
	var workspaceID string
	var invitedEmail, role string
	err = tx.QueryRowContext(r.Context(), "SELECT workspace_id, email, role FROM workspace_invitations WHERE token_hash = ? AND expires_at > UTC_TIMESTAMP(6) FOR UPDATE", digest[:]).Scan(&workspaceID, &invitedEmail, &role)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusGone, "invitation is invalid or expired")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read invitation")
		return
	}
	var tokenValue string
	var sessionExpiresAt time.Time
	if !hasSession {
		var existingID uint64
		err := tx.QueryRowContext(r.Context(), "SELECT id FROM users WHERE email = ?", invitedEmail).Scan(&existingID)
		if err == nil {
			writeError(w, http.StatusUnauthorized, "log in with the invited email to accept this invitation")
			return
		}
		if !errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusInternalServerError, "could not check invited account")
			return
		}
		credentials := credentials{Username: input.Username, Email: invitedEmail, Password: input.Password}
		if err := validateCredentials(&credentials); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		passwordHash, err := hashPassword(credentials.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not hash password")
			return
		}
		user.ID, err = createUser(r, tx, credentials.Username, credentials.Email, passwordHash, false)
		if err != nil {
			writeUserError(w, err)
			return
		}
		if err := createWorkspace(r, tx, user.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not create personal workspace")
			return
		}
		user.Username, user.Email = credentials.Username, credentials.Email
		tokenValue, sessionExpiresAt, err = createSession(r, tx, user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not create session")
			return
		}
	} else if normalizeEmail(user.Email) != invitedEmail {
		writeError(w, http.StatusForbidden, "invitation belongs to a different email address")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "INSERT INTO workspace_members (workspace_id, user_id, role) VALUES (?, ?, ?)", workspaceID, user.ID, role); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			writeError(w, http.StatusConflict, "account is already a workspace member")
		} else {
			writeError(w, http.StatusInternalServerError, "could not add account to workspace")
		}
		return
	}
	if _, err := tx.ExecContext(r.Context(), "DELETE FROM workspace_invitations WHERE token_hash = ?", digest[:]); err != nil {
		writeError(w, http.StatusInternalServerError, "could not consume invitation")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not complete invitation acceptance")
		return
	}
	if tokenValue != "" {
		setSessionCookie(w, tokenValue, sessionExpiresAt)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"user": user, "workspace_id": workspaceID, "role": role})
}
