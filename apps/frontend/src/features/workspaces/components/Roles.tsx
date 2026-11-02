import { type FormEvent, useState } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { type Permission, permissionList } from "@/features/projects/ProjectsContext";
import { type Role } from "@/features/workspaces/types";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t, translate } from "@/lib/i18n";

const permissionLabels: Record<Permission, string> = {
  "workspace.manage": t("Rename the workspace"),
  "members.invite": t("Invite members"),
  "members.manage": t("Manage members and roles"),
  "projects.manage": t("Create, edit and delete projects"),
  "apps.manage": t("Create, configure and deploy apps"),
  "terminal.use": t("Open app terminals"),
  "tasks.delete": t("Delete tasks, lists, labels and others' comments"),
};

function PermissionChecks({ idPrefix, value, grantable, disabled, onChange }: { idPrefix: string; value: string[]; grantable: (permission: Permission) => boolean; disabled?: boolean; onChange?: (value: string[]) => void }) {
  return (
    <ul className="grid gap-1 sm:grid-cols-2">
      {permissionList.map((permission) => (
        <li key={permission}>
          <label className="flex min-h-8 cursor-pointer items-center gap-2 text-meta" htmlFor={`${idPrefix}-${permission}`}>
            <input
              className="size-4 flex-none cursor-pointer accent-seafoam"
              type="checkbox"
              id={`${idPrefix}-${permission}`}
              checked={value.includes(permission)}
              disabled={disabled || !onChange || !grantable(permission)}
              onChange={(event) => onChange?.(event.target.checked ? [...value, permission] : value.filter((item) => item !== permission))}
            />
            <span>{permissionLabels[permission]}</span>
          </label>
        </li>
      ))}
    </ul>
  );
}

function RoleRow({ role, workspaceId, editable, grantable, onChanged }: { role: Role; workspaceId: string; editable: boolean; grantable: (permission: Permission) => boolean; onChanged: (name: string) => void }) {
  const [draft, setDraft] = useState(role.permissions);
  const [confirming, setConfirming] = useState(false);
  const { pending, error, run } = useAction();
  const url = `/workspaces/${workspaceId}/roles/${encodeURIComponent(role.name)}`;
  const dirty = draft.length !== role.permissions.length || draft.some((item) => !role.permissions.includes(item));

  return (
    <li className="border-b border-line px-4 py-3 md:px-6">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <p className="min-w-0 flex-1 font-medium wrap-break-word">
          {translate(role.name)}
          {role.builtin && <span className="ml-2 text-xs font-normal text-ink-soft">{t("Built-in")}</span>}
        </p>
        {editable && (
          <span className="flex gap-2">
            {dirty && (
              <button className="btn btn-primary" type="button" disabled={pending} onClick={() => run(async () => { await api(url, { method: "PUT", body: { permissions: draft } }); onChanged(role.name); })}>
                {t("Save")}
              </button>
            )}
            <button className="btn btn-danger" type="button" disabled={pending} onClick={confirming ? () => run(async () => { await api(url, { method: "DELETE" }); onChanged(role.name); }) : () => setConfirming(true)} aria-label={confirming ? t("Confirm delete {name}", { name: role.name }) : t("Delete {name}", { name: role.name })}>
              {confirming ? t("Confirm delete") : t("Delete")}
            </button>
          </span>
        )}
      </div>
      <div className="mt-2">
        <PermissionChecks idPrefix={`role-${role.name}`} value={editable ? draft : role.permissions} grantable={grantable} disabled={pending} onChange={editable ? setDraft : undefined} />
      </div>
      <FormError message={error} />
    </li>
  );
}

function NewRoleForm({ workspaceId, grantable, onCreated, onClose }: { workspaceId: string; grantable: (permission: Permission) => boolean; onCreated: (name: string) => void; onClose: () => void }) {
  const [name, setName] = useState("");
  const [granted, setGranted] = useState<string[]>([]);
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    const created = await run(async () => {
      await api(`/workspaces/${workspaceId}/roles`, { method: "POST", body: { name, permissions: granted } });
    });
    if (created) {
      onCreated(name);
      onClose();
    }
  }

  return (
    <form className="mx-4 mt-4 rounded-panel bg-raised p-4 md:mx-6" onSubmit={submit} aria-label={t("New role")}>
      <Field label={t("Role name")} required autoFocus maxLength={32} value={name} onChange={(event) => setName(event.target.value)} />
      <fieldset className="mt-3">
        <legend className="mb-1 text-meta font-medium">{t("Permissions")}</legend>
        <PermissionChecks idPrefix="new-role" value={granted} grantable={grantable} disabled={pending} onChange={setGranted} />
      </fieldset>
      <div className="mt-3">
        <FormError message={error} />
      </div>
      <div className="mt-2 flex flex-wrap gap-2">
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? t("Creating…") : t("Create role")}
        </button>
        <button className="btn" type="button" onClick={onClose} disabled={pending}>
          {t("Cancel")}
        </button>
      </div>
    </form>
  );
}

export function Roles({ roles, workspaceId, canManage, grantable, onChanged }: { roles: Role[]; workspaceId: string; canManage: boolean; grantable: (permission: Permission) => boolean; onChanged: (name: string) => void }) {
  const [adding, setAdding] = useState(false);
  const covered = (role: Role) => role.permissions.every((permission) => grantable(permission as Permission));

  return (
    <section className="mt-8" aria-labelledby="roles-heading">
      <div className="flex flex-wrap items-center justify-between gap-2 px-4 md:px-6">
        <div>
          <h2 className="font-medium" id="roles-heading">
            {t("Roles")}
          </h2>
          <p className="text-meta text-ink-soft">{t("Every member can view the workspace. Roles add what they can change.")}</p>
        </div>
        {canManage && !adding && (
          <button className="btn" type="button" onClick={() => setAdding(true)}>
            {t("New role")}
          </button>
        )}
      </div>
      {adding && <NewRoleForm workspaceId={workspaceId} grantable={grantable} onCreated={onChanged} onClose={() => setAdding(false)} />}
      <ul className="mt-4 border-t border-line">
        {roles.map((role) => (
          <RoleRow role={role} workspaceId={workspaceId} editable={canManage && !role.builtin && covered(role)} grantable={grantable} onChanged={onChanged} key={`${role.name}-${role.permissions.join()}`} />
        ))}
      </ul>
    </section>
  );
}
