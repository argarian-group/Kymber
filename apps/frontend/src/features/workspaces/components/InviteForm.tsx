import { type FormEvent, useRef, useState } from "react";
import { CopyField } from "@/components/ui/CopyField";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { RoleSelect } from "@/features/workspaces/components/RoleSelect";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function InviteForm({ workspaceId, roles, canAddDirectly, onAdded, onClose }: { workspaceId: string; roles: string[]; canAddDirectly: boolean; onAdded: () => void; onClose: () => void }) {
  const [email, setEmail] = useState("");
  const [role, setRole] = useState(roles.includes("member") ? "member" : (roles[0] ?? ""));
  const { pending, error, run } = useAction();
  const [invite, setInvite] = useState<{ email: string; link: string } | null>(null);
  const formRef = useRef<HTMLFormElement>(null);

  async function createInvite(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      const { invitation } = await api<{ invitation: { email: string; token: string } }>(`/workspaces/${workspaceId}/invitations`, { method: "POST", body: { email, role } });
      setInvite({ email: invitation.email, link: `${window.location.origin}/invite#${invitation.token}` });
    });
  }

  async function addDirectly() {
    if (!formRef.current?.reportValidity()) return;
    await run(async () => {
      await api(`/workspaces/${workspaceId}/members`, { method: "POST", body: { email, role } });
      onAdded();
      onClose();
    });
  }

  if (invite) {
    return (
      <div className="mx-4 mt-4 rounded-panel bg-raised p-4 md:mx-6" role="status">
        <p className="font-medium wrap-break-word">{t("Invite link for {email}", { email: invite.email })}</p>
        <p className="mt-1 text-meta text-ink-soft">{t("Kymber does not send this by email. Copy it now, because it will not be shown again. It expires in 7 days.")}</p>
        <div className="mt-3">
          <CopyField label={t("Invite link")} value={invite.link} />
        </div>
        <button className="btn mt-3" type="button" onClick={onClose}>
          {t("Done")}
        </button>
      </div>
    );
  }

  return (
    <form className="mx-4 mt-4 rounded-panel bg-raised p-4 md:mx-6" ref={formRef} onSubmit={createInvite} aria-label={t("Invite member")}>
      <div className="grid items-end gap-4 sm:grid-cols-[minmax(0,1fr)_10rem]">
        <Field label={t("Email")} type="email" required autoFocus value={email} onChange={(event) => setEmail(event.target.value)} />
        <div className="sm:self-start">
          <span className="mb-1 block text-meta font-medium" aria-hidden="true">
            {t("Role")}
          </span>
          <RoleSelect id="new-member-role" label={t("Role")} value={role} options={roles} onChange={setRole} />
        </div>
      </div>
      <div className="mt-3">
        <FormError message={error} />
      </div>
      <div className="mt-2 flex flex-wrap gap-2">
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? t("Creating…") : t("Create invite link")}
        </button>
        {canAddDirectly && (
          <button className="btn" type="button" onClick={addDirectly} disabled={pending}>
            {t("Add existing account")}
          </button>
        )}
        <button className="btn" type="button" onClick={onClose} disabled={pending}>
          {t("Cancel")}
        </button>
      </div>
    </form>
  );
}
