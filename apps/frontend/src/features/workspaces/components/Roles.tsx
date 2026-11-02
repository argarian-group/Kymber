import { type FormEvent, useState } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { Modal } from "@/components/ui/Modal";
import { type Permission, permissionList } from "@/features/projects/ProjectsContext";
import { type Member, type Role } from "@/features/workspaces/types";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t, translate } from "@/lib/i18n";

const permissionText: Record<Permission, { label: string; hint: string }> = {
  "workspace.manage": { label: t("Rename the workspace"), hint: t("Change the workspace name in settings.") },
  "members.invite": { label: t("Invite members"), hint: t("Create invite links with roles they can grant.") },
  "members.manage": { label: t("Manage members and roles"), hint: t("Add and remove members, change their roles and edit roles.") },
  "projects.manage": { label: t("Manage projects"), hint: t("Create, rename and delete projects and their environments.") },
  "apps.manage": { label: t("Manage apps"), hint: t("Create, configure, deploy and delete apps, and browse their files.") },
  "terminal.use": { label: t("Use app terminals"), hint: t("Open a shell inside running app containers.") },
  "tasks.delete": { label: t("Delete tasks"), hint: t("Delete tasks, lists, labels and other people's comments. Everyone can create and edit tasks.") },
};

const permissionGroups: { title: string; permissions: Permission[] }[] = [
  { title: t("Workspace and team"), permissions: ["workspace.manage", "members.invite", "members.manage"] },
  { title: t("Projects and apps"), permissions: ["projects.manage", "apps.manage", "terminal.use"] },
  { title: t("Tasks"), permissions: ["tasks.delete"] },
];

function summary(role: Role) {
  if (role.permissions.length === 0) return t("View only");
  if (role.permissions.length === permissionList.length) return t("Full access");
  return role.permissions.map((permission) => permissionText[permission as Permission]?.label ?? permission).join(", ");
}

function memberCount(count: number) {
  if (count === 0) return t("No members");
  return count === 1 ? t("1 member") : t("{count} members", { count });
}

export function RoleList({ roles, members, canEdit, onEdit }: { roles: Role[]; members: Member[]; canEdit: (role: Role) => boolean; onEdit: (role: Role) => void }) {
  return (
    <>
      <p className="px-4 pt-4 text-meta text-ink-soft md:px-6">{t("Every member can view the workspace. A role decides what they can change. The owner role always has full access.")}</p>
      <ul className="mt-4 border-t border-line">
        {roles.map((role) => (
          <li className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-line px-4 py-3 md:px-6" key={role.name}>
            <div className="min-w-0 flex-1 basis-60">
              <p className="font-medium wrap-break-word">
                {translate(role.name)}
                {role.builtin && <span className="ml-2 text-xs font-normal text-ink-soft">{role.modified ? t("Built-in, changed") : t("Built-in")}</span>}
              </p>
              <p className="text-meta wrap-break-word text-ink-soft">{summary(role)}</p>
            </div>
            <span className="text-meta text-ink-soft tabular-nums">{memberCount(members.filter((member) => member.role === role.name).length)}</span>
            {canEdit(role) && (
              <button className="btn" type="button" onClick={() => onEdit(role)} aria-label={t("Edit {name}", { name: translate(role.name) })}>
                {t("Edit")}
              </button>
            )}
          </li>
        ))}
      </ul>
    </>
  );
}

export function RoleDialog({ role, workspaceId, inUse, grantable, onSaved, onClose }: { role: Role | null; workspaceId: string; inUse: boolean; grantable: (permission: Permission) => boolean; onSaved: () => void; onClose: () => void }) {
  const [name, setName] = useState(role?.name ?? "");
  const [granted, setGranted] = useState<string[]>(role?.permissions ?? []);
  const [confirming, setConfirming] = useState(false);
  const { pending, error, run } = useAction();
  const url = role ? `/workspaces/${workspaceId}/roles/${encodeURIComponent(role.name)}` : `/workspaces/${workspaceId}/roles`;
  const title = role ? t("Edit role {name}", { name: translate(role.name) }) : t("New role");

  async function submit(event: FormEvent) {
    event.preventDefault();
    const saved = await run(() => api(url, { method: role ? "PUT" : "POST", body: role ? { permissions: granted } : { name, permissions: granted } }));
    if (saved) {
      onSaved();
      onClose();
    }
  }

  async function remove() {
    if (await run(() => api(url, { method: "DELETE" }))) {
      onSaved();
      onClose();
    } else setConfirming(false);
  }

  return (
    <Modal label={title} className="max-w-lg" onClose={onClose}>
      <form className="grid gap-4 p-4" onSubmit={submit}>
        <h2 className="section-title">{title}</h2>
        {!role && <Field label={t("Role name")} required autoFocus maxLength={32} hint={t("Names cannot be changed later.")} value={name} onChange={(event) => setName(event.target.value)} />}
        {permissionGroups.map((group) => (
          <fieldset className="grid gap-1" key={group.title}>
            <legend className="mb-1 text-xs font-medium tracking-wide text-ink-soft uppercase">{group.title}</legend>
            {group.permissions.map((permission) => (
              <label className={`flex items-start gap-2 py-1 ${grantable(permission) ? "cursor-pointer" : "opacity-60"}`} key={permission}>
                <input
                  className="mt-1 size-4 flex-none cursor-pointer accent-seafoam disabled:cursor-not-allowed"
                  type="checkbox"
                  checked={granted.includes(permission)}
                  disabled={pending || !grantable(permission)}
                  onChange={(event) => setGranted(event.target.checked ? [...granted, permission] : granted.filter((item) => item !== permission))}
                />
                <span>
                  <span className="block text-meta font-medium">{permissionText[permission].label}</span>
                  <span className="block text-xs text-ink-soft">{grantable(permission) ? permissionText[permission].hint : t("You do not have this permission, so you cannot grant it.")}</span>
                </span>
              </label>
            ))}
          </fieldset>
        ))}
        <FormError message={error} />
        <div className="flex flex-wrap items-center gap-2">
          <button className="btn btn-primary" type="submit" disabled={pending}>
            {role ? t("Save") : t("Create role")}
          </button>
          <button className="btn" type="button" onClick={onClose} disabled={pending}>
            {t("Cancel")}
          </button>
          {role?.builtin && role.modified && (
            <button className="btn ml-auto" type="button" disabled={pending} onClick={remove}>
              {t("Restore defaults")}
            </button>
          )}
          {role && !role.builtin && !inUse && (
            <button className="btn btn-danger ml-auto" type="button" disabled={pending} onClick={confirming ? remove : () => setConfirming(true)}>
              {confirming ? t("Confirm delete") : t("Delete role")}
            </button>
          )}
        </div>
        {role && !role.builtin && inUse && <p className="text-xs text-ink-soft">{t("Move its members to another role before deleting it.")}</p>}
      </form>
    </Modal>
  );
}
