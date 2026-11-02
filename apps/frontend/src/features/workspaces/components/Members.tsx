import { useCallback, useState } from "react";
import { NavLink } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { type Permission, useProjects, type Workspace, workspaceLabel } from "@/features/projects/ProjectsContext";
import { InviteForm } from "@/features/workspaces/components/InviteForm";
import { MemberRow } from "@/features/workspaces/components/MemberRow";
import { RoleDialog, RoleList } from "@/features/workspaces/components/Roles";
import { type Member, type Role } from "@/features/workspaces/types";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

type MembersState = { status: "loading" } | { status: "error"; message: string } | { status: "ready"; members: Member[]; roles: Role[] };

export function Members({ workspace, section }: { workspace: Workspace; section: "members" | "roles" }) {
  const user = useCurrentUser();
  const { workspaces, can, retry: reloadWorkspaces } = useProjects();
  const canManageMembers = can("members.manage");
  const canInvite = can("members.invite");
  const [state, setState] = useState<MembersState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);
  const [adding, setAdding] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | "new" | null>(null);

  useLoadEffect(
    () => Promise.all([api<{ members: Member[] }>(`/workspaces/${workspace.id}/members`), api<{ roles: Role[] }>(`/workspaces/${workspace.id}/roles`)]),
    ([{ members }, { roles }]) => setState({ status: "ready", members, roles }),
    (message) => setState({ status: "error", message }),
    [attempt, workspace.id],
  );

  const reload = useCallback(() => {
    setState({ status: "loading" });
    setAttempt((value) => value + 1);
  }, []);

  function handleChanged(member: Member, removed: boolean) {
    if (member.user_id === user.id) {
      reloadWorkspaces();
      return;
    }
    setState((previous) =>
      previous.status !== "ready"
        ? previous
        : { ...previous, members: removed ? previous.members.filter(({ user_id }) => user_id !== member.user_id) : previous.members.map((item) => (item.user_id === member.user_id ? member : item)) },
    );
  }

  const roles = state.status === "ready" ? state.roles : [];
  const covered = (name: string) => roles.find((role) => role.name === name)?.permissions.every((permission) => can(permission as Permission)) ?? false;
  const assignable = roles.filter(({ name }) => covered(name)).map(({ name }) => name);
  const count = state.status === "ready" ? state.members.length : null;
  const tabs = [
    { label: t("Members"), to: "/team" },
    { label: t("Roles"), to: "/team/roles" },
  ];
  const action =
    section === "members"
      ? canInvite &&
        !adding && (
          <button className="btn btn-primary" type="button" onClick={() => setAdding(true)}>
            {t("Invite member")}
          </button>
        )
      : canManageMembers && (
          <button className="btn btn-primary" type="button" onClick={() => setEditingRole("new")}>
            {t("New role")}
          </button>
        );
  return (
    <>
      <PageHeader
        title={t("Team")}
        description={count === null ? workspaceLabel(workspace, workspaces) : (count === 1 ? t("{workspace}, 1 member", { workspace: workspaceLabel(workspace, workspaces) }) : t("{workspace}, {count} members", { workspace: workspaceLabel(workspace, workspaces), count }))}
        actions={action}
      />
      <nav className="mt-4 flex gap-5 overflow-x-auto border-b border-line px-4 md:px-6" aria-label={t("Team sections")}>
        {tabs.map(({ label, to }) => (
          <NavLink
            className={({ isActive }) =>
              `focus-ring flex min-h-10 flex-none items-center border-b-2 text-meta font-medium transition-colors duration-100 ${isActive ? "border-seafoam text-ink" : "border-transparent text-ink-soft hover:text-ink"}`
            }
            to={to}
            end
            key={to}
          >
            {label}
          </NavLink>
        ))}
      </nav>
      {state.status === "loading" && <PageSkeleton />}
      {state.status === "error" && <LoadError title={t("Could not load members")} message={state.message} onRetry={reload} />}
      {state.status === "ready" && section === "members" && (
        <>
          {adding && <InviteForm workspaceId={workspace.id} roles={assignable} canAddDirectly={canManageMembers} onAdded={reload} onClose={() => setAdding(false)} />}
          {state.members.length === 0 ? (
            <EmptyState title={t("No members yet")} description={canInvite ? t("Invite someone to this workspace.") : t("Ask someone who can invite members.")} />
          ) : (
            <ul>
              {state.members.map((member) => (
                <MemberRow member={member} workspaceId={workspace.id} roles={assignable} isCreator={workspace.created_by?.id === member.user_id} canManage={canManageMembers && covered(member.role)} onChanged={handleChanged} key={member.user_id} />
              ))}
            </ul>
          )}
        </>
      )}
      {state.status === "ready" && section === "roles" && (
        <RoleList roles={state.roles} members={state.members} canEdit={(role) => canManageMembers && !role.builtin && covered(role.name)} onEdit={setEditingRole} />
      )}
      {editingRole && (
        <RoleDialog
          role={editingRole === "new" ? null : editingRole}
          workspaceId={workspace.id}
          inUse={editingRole !== "new" && state.status === "ready" && state.members.some(({ role }) => role === editingRole.name)}
          grantable={can}
          onSaved={() => (editingRole !== "new" && editingRole.name === workspace.role ? reloadWorkspaces() : reload())}
          onClose={() => setEditingRole(null)}
        />
      )}
    </>
  );
}
