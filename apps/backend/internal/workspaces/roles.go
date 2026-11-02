package workspaces

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-sql-driver/mysql"
	"kymber/backend/internal/auth"
	"kymber/backend/internal/permissions"
)

var roleNamePattern = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N} _-]{0,31}$`)

type WorkspaceRole struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	Builtin     bool     `json:"builtin"`
	Modified    bool     `json:"modified"`
}

type rolePayload struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, "") {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT name, permissions FROM workspace_roles WHERE workspace_id = ? ORDER BY created_at, name", workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list roles")
		return
	}
	defer rows.Close()
	overrides := map[string]string{}
	custom := make([]WorkspaceRole, 0)
	for rows.Next() {
		var role WorkspaceRole
		var stored string
		if err := rows.Scan(&role.Name, &stored); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read roles")
			return
		}
		if _, builtin := permissions.Builtin[role.Name]; builtin {
			overrides[role.Name] = stored
			continue
		}
		role.Permissions = permissions.ForRole(role.Name, sql.NullString{String: stored, Valid: true}).List()
		custom = append(custom, role)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read roles")
		return
	}
	roles := make([]WorkspaceRole, 0, len(permissions.BuiltinOrder)+len(custom))
	for _, name := range permissions.BuiltinOrder {
		stored, modified := overrides[name]
		roles = append(roles, WorkspaceRole{Name: name, Permissions: permissions.ForRole(name, sql.NullString{String: stored, Valid: modified}).List(), Builtin: true, Modified: modified})
	}
	roles = append(roles, custom...)
	writeJSON(w, http.StatusOK, map[string][]WorkspaceRole{"roles": roles})
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok {
		return
	}
	actor, ok := h.requirePermission(w, r, user, workspaceID, permissions.MembersManage)
	if !ok {
		return
	}
	var input rolePayload
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	lower := strings.ToLower(input.Name)
	if _, builtin := permissions.Builtin[lower]; builtin || lower == "root" || !roleNamePattern.MatchString(input.Name) {
		writeError(w, http.StatusBadRequest, "role name must be 1 to 32 letters, numbers, spaces, hyphens or underscores and not a built-in role")
		return
	}
	granted, ok := grantable(w, actor, input.Permissions)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO workspace_roles (workspace_id, name, permissions) VALUES (?, ?, ?)", workspaceID, input.Name, permissions.Encode(granted)); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			writeError(w, http.StatusConflict, "a role with this name already exists")
		} else {
			writeError(w, http.StatusInternalServerError, "could not create role")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]WorkspaceRole{"role": {Name: input.Name, Permissions: granted.List()}})
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, name, actor, ok := h.customRoleRequest(w, r, user)
	if !ok {
		return
	}
	var input rolePayload
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Name != "" && input.Name != name {
		writeError(w, http.StatusBadRequest, "roles cannot be renamed")
		return
	}
	granted, ok := grantable(w, actor, input.Permissions)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO workspace_roles (workspace_id, name, permissions) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE permissions = VALUES(permissions)", workspaceID, name, permissions.Encode(granted)); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update role")
		return
	}
	_, builtin := permissions.Builtin[name]
	writeJSON(w, http.StatusOK, map[string]WorkspaceRole{"role": {Name: name, Permissions: granted.List(), Builtin: builtin, Modified: builtin}})
}

func (h *Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, name, _, ok := h.customRoleRequest(w, r, user)
	if !ok {
		return
	}
	if _, builtin := permissions.Builtin[name]; builtin {
		if _, err := h.db.ExecContext(r.Context(), "DELETE FROM workspace_roles WHERE workspace_id = ? AND name = ?", workspaceID, name); err != nil {
			writeError(w, http.StatusInternalServerError, "could not restore role")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var inUse bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id = ? AND role = ?) OR EXISTS(SELECT 1 FROM workspace_invitations WHERE workspace_id = ? AND role = ? AND expires_at > UTC_TIMESTAMP(6))", workspaceID, name, workspaceID, name).Scan(&inUse); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check role usage")
		return
	}
	if inUse {
		writeError(w, http.StatusConflict, "move members and pending invitations to another role first")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "DELETE FROM workspace_roles WHERE workspace_id = ? AND name = ?", workspaceID, name); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete role")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) customRoleRequest(w http.ResponseWriter, r *http.Request, user auth.User) (string, string, permissions.Set, bool) {
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok {
		return "", "", nil, false
	}
	actor, ok := h.requirePermission(w, r, user, workspaceID, permissions.MembersManage)
	if !ok {
		return "", "", nil, false
	}
	name := r.PathValue("roleName")
	if !permissions.Editable(name) {
		writeError(w, http.StatusBadRequest, "the owner role always has full access")
		return "", "", nil, false
	}
	current, exists, err := permissions.RolePermissions(r.Context(), h.db, workspaceID, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read role")
		return "", "", nil, false
	}
	if !exists {
		writeError(w, http.StatusNotFound, "role not found")
		return "", "", nil, false
	}
	if !actor.Covers(current) {
		writeError(w, http.StatusForbidden, "this role has permissions you do not have")
		return "", "", nil, false
	}
	return workspaceID, name, actor, true
}

func grantable(w http.ResponseWriter, actor permissions.Set, list []string) (permissions.Set, bool) {
	if !permissions.Valid(list) {
		writeError(w, http.StatusBadRequest, "unknown permission")
		return nil, false
	}
	granted := permissions.Of(list)
	if !actor.Covers(granted) {
		writeError(w, http.StatusForbidden, "you cannot grant permissions you do not have")
		return nil, false
	}
	return granted, true
}
