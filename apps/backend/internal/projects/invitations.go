package projects

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"kymber/backend/internal/auth"
)

type invitationPayload struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *Handler) createInvitation(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok {
		return
	}
	actorRole, allowed := h.requireWorkspaceRole(w, r, user, workspaceID, true)
	if !allowed {
		return
	}
	var input invitationPayload
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email || len(input.Email) > 254 || !validMemberRole(input.Role) {
		writeError(w, http.StatusBadRequest, "valid email and role are required")
		return
	}
	if actorRole == "admin" && input.Role == "owner" {
		writeError(w, http.StatusForbidden, "only an owner can invite another owner")
		return
	}
	var alreadyMember bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM workspace_members m JOIN users u ON u.id = m.user_id WHERE m.workspace_id = ? AND u.email = ?)", workspaceID, input.Email).Scan(&alreadyMember); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check workspace membership")
		return
	}
	if alreadyMember {
		writeError(w, http.StatusConflict, "user is already a workspace member")
		return
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create invitation")
		return
	}
	token := base64.RawURLEncoding.EncodeToString(secret)
	digest := sha256.Sum256(secret)
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create invitation")
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(), "DELETE FROM workspace_invitations WHERE workspace_id = ? AND email = ?", workspaceID, input.Email); err != nil {
		writeError(w, http.StatusInternalServerError, "could not replace pending invitation")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "INSERT INTO workspace_invitations (token_hash, workspace_id, email, role, invited_by, expires_at) VALUES (?, ?, ?, ?, ?, ?)", digest[:], workspaceID, input.Email, input.Role, user.ID, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create invitation")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not complete invitation")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"invitation": map[string]any{"email": input.Email, "role": input.Role, "token": token, "expires_at": expiresAt}})
}
