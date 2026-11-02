package permissions

import (
	"context"
	"database/sql"
	"strings"
)

const (
	WorkspaceManage = "workspace.manage"
	MembersInvite   = "members.invite"
	MembersManage   = "members.manage"
	ProjectsManage  = "projects.manage"
	AppsManage      = "apps.manage"
	TerminalUse     = "terminal.use"
	TasksDelete     = "tasks.delete"
)

var All = []string{WorkspaceManage, MembersInvite, MembersManage, ProjectsManage, AppsManage, TerminalUse, TasksDelete}

var Builtin = map[string][]string{
	"owner":  All,
	"admin":  {WorkspaceManage, MembersInvite, ProjectsManage, AppsManage, TerminalUse, TasksDelete},
	"member": {},
}

var BuiltinOrder = []string{"owner", "admin", "member"}

type Set map[string]bool

func (s Set) Has(permission string) bool { return s[permission] }

func (s Set) Covers(other Set) bool {
	for permission := range other {
		if !s[permission] {
			return false
		}
	}
	return true
}

func (s Set) List() []string {
	list := make([]string, 0, len(s))
	for _, permission := range All {
		if s[permission] {
			list = append(list, permission)
		}
	}
	return list
}

func Of(list []string) Set {
	set := Set{}
	for _, permission := range list {
		set[permission] = true
	}
	return set
}

func Valid(list []string) bool {
	known := Of(All)
	for _, permission := range list {
		if !known[permission] {
			return false
		}
	}
	return true
}

func Encode(set Set) string { return strings.Join(set.List(), ",") }

func ForRole(role string, custom sql.NullString) Set {
	if role == "root" {
		return Of(All)
	}
	if list, ok := Builtin[role]; ok {
		return Of(list)
	}
	if !custom.Valid || custom.String == "" {
		return Set{}
	}
	return Of(strings.Split(custom.String, ","))
}

type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func Lookup(ctx context.Context, db querier, workspaceID string, userID uint64) (string, Set, error) {
	var role string
	var custom sql.NullString
	err := db.QueryRowContext(ctx, "SELECT m.role, r.permissions FROM workspace_members m LEFT JOIN workspace_roles r ON r.workspace_id = m.workspace_id AND r.name = m.role WHERE m.workspace_id = ? AND m.user_id = ?", workspaceID, userID).Scan(&role, &custom)
	if err != nil {
		return "", nil, err
	}
	return role, ForRole(role, custom), nil
}

func RolePermissions(ctx context.Context, db querier, workspaceID, role string) (Set, bool, error) {
	if list, ok := Builtin[role]; ok {
		return Of(list), true, nil
	}
	var stored string
	err := db.QueryRowContext(ctx, "SELECT permissions FROM workspace_roles WHERE workspace_id = ? AND name = ?", workspaceID, role).Scan(&stored)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return ForRole(role, sql.NullString{String: stored, Valid: true}), true, nil
}
