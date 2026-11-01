package projects

import (
	"database/sql"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"kymber/backend/internal/auth"
)

type WorkspaceMember struct {
	UserID    uint64    `json:"user_id,string"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type memberPayload struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type memberRolePayload struct {
	Role string `json:"role"`
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, false) {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `SELECT u.id, u.username, u.email, m.role, m.created_at FROM workspace_members m JOIN users u ON u.id = m.user_id WHERE m.workspace_id = ? ORDER BY m.created_at, u.id`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list workspace members")
		return
	}
	defer rows.Close()
	members := make([]WorkspaceMember, 0)
	for rows.Next() {
		var member WorkspaceMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.Email, &member.Role, &member.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read workspace members")
			return
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read workspace members")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]WorkspaceMember{"members": members})
}

func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.requireWorkspaceOwner(w, r, user, workspaceID) {
		return
	}
	var input memberPayload
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email || len(input.Email) > 254 || !validMemberRole(input.Role) {
		writeError(w, http.StatusBadRequest, "valid email and role are required")
		return
	}
	var member WorkspaceMember
	err = h.db.QueryRowContext(r.Context(), "SELECT id, username, email FROM users WHERE email = ?", input.Email).Scan(&member.UserID, &member.Username, &member.Email)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not find user")
		return
	}
	var creatorID uint64
	if err := h.db.QueryRowContext(r.Context(), "SELECT COALESCE(created_by, 0) FROM workspaces WHERE id = ?", workspaceID).Scan(&creatorID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify workspace creator")
		return
	}
	if member.UserID == creatorID && input.Role != "owner" {
		writeError(w, http.StatusConflict, "workspace creator must remain an owner")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO workspace_members (workspace_id, user_id, role) VALUES (?, ?, ?)", workspaceID, member.UserID, input.Role); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			writeError(w, http.StatusConflict, "user is already a workspace member")
		} else {
			writeError(w, http.StatusInternalServerError, "could not add workspace member")
		}
		return
	}
	member.Role = input.Role
	writeJSON(w, http.StatusCreated, map[string]WorkspaceMember{"member": member})
}

func (h *Handler) updateMember(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.requireWorkspaceOwner(w, r, user, workspaceID) {
		return
	}
	userID, ok := pathUserID(w, r)
	if !ok {
		return
	}
	var input memberRolePayload
	if !decodeJSON(w, r, &input) {
		return
	}
	if !validMemberRole(input.Role) {
		writeError(w, http.StatusBadRequest, "role must be owner, admin, or member")
		return
	}
	var creatorID uint64
	if err := h.db.QueryRowContext(r.Context(), "SELECT COALESCE(created_by, 0) FROM workspaces WHERE id = ?", workspaceID).Scan(&creatorID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify workspace creator")
		return
	}
	if userID == creatorID && input.Role != "owner" {
		writeError(w, http.StatusConflict, "workspace creator must remain an owner")
		return
	}
	result, err := h.db.ExecContext(r.Context(), "UPDATE workspace_members SET role = ? WHERE workspace_id = ? AND user_id = ?", input.Role, workspaceID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update workspace member")
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update workspace member")
		return
	}
	if changed == 0 {
		var exists bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id = ? AND user_id = ?)", workspaceID, userID).Scan(&exists); err != nil {
			writeError(w, http.StatusInternalServerError, "could not check workspace member")
			return
		}
		if !exists {
			writeError(w, http.StatusNotFound, "workspace member not found")
			return
		}
	}
	member, err := h.findMember(r, workspaceID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read workspace member")
		return
	}
	writeJSON(w, http.StatusOK, map[string]WorkspaceMember{"member": member})
}

func (h *Handler) deleteMember(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.requireWorkspaceOwner(w, r, user, workspaceID) {
		return
	}
	userID, ok := pathUserID(w, r)
	if !ok {
		return
	}
	var creatorID uint64
	if err := h.db.QueryRowContext(r.Context(), "SELECT COALESCE(created_by, 0) FROM workspaces WHERE id = ?", workspaceID).Scan(&creatorID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify workspace creator")
		return
	}
	if userID == creatorID {
		writeError(w, http.StatusConflict, "workspace creator cannot be removed")
		return
	}
	result, err := h.db.ExecContext(r.Context(), "DELETE FROM workspace_members WHERE workspace_id = ? AND user_id = ?", workspaceID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not remove workspace member")
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not remove workspace member")
		return
	}
	if deleted == 0 {
		writeError(w, http.StatusNotFound, "workspace member not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) requireWorkspaceOwner(w http.ResponseWriter, r *http.Request, user auth.User, workspaceID string) bool {
	_, ok := h.requireWorkspaceRole(w, r, user, workspaceID, false)
	return ok
}

func (h *Handler) requireWorkspaceRole(w http.ResponseWriter, r *http.Request, user auth.User, workspaceID string, allowAdmin bool) (string, bool) {
	if !h.authorize(w, r, user, workspaceID, false) {
		return "", false
	}
	if user.IsRoot {
		return "root", true
	}
	var role string
	err := h.db.QueryRowContext(r.Context(), "SELECT role FROM workspace_members WHERE workspace_id = ? AND user_id = ?", workspaceID, user.ID).Scan(&role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify workspace ownership")
		return "", false
	}
	if role != "owner" && !(allowAdmin && role == "admin") {
		message := "workspace owner access required"
		if allowAdmin {
			message = "workspace owner or admin access required"
		}
		writeError(w, http.StatusForbidden, message)
		return "", false
	}
	return role, true
}

func (h *Handler) findMember(r *http.Request, workspaceID string, userID uint64) (WorkspaceMember, error) {
	var member WorkspaceMember
	err := h.db.QueryRowContext(r.Context(), `SELECT u.id, u.username, u.email, m.role, m.created_at FROM workspace_members m JOIN users u ON u.id = m.user_id WHERE m.workspace_id = ? AND u.id = ?`, workspaceID, userID).Scan(&member.UserID, &member.Username, &member.Email, &member.Role, &member.CreatedAt)
	return member, err
}

func validMemberRole(role string) bool {
	return role == "owner" || role == "admin" || role == "member"
}
