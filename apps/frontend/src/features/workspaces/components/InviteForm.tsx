import { type FormEvent, useRef, useState } from "react";
import { CopyField } from "@/components/ui/CopyField";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { roles, RoleSelect } from "@/features/workspaces/components/RoleSelect";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

export function InviteForm({ workspaceId, canAddDirectly, onAdded, onClose }: { workspaceId: string; canAddDirectly: boolean; onAdded: () => void; onClose: () => void }) {
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("member");
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
        <p className="font-medium wrap-break-word">Invite link for {invite.email}</p>
        <p className="mt-1 text-meta text-ink-soft">Kymber does not send this by email. Copy it now, because it will not be shown again. It expires in 7 days.</p>
        <div className="mt-3">
          <CopyField label="Invite link" value={invite.link} />
        </div>
        <button className="btn mt-3" type="button" onClick={onClose}>
          Done
        </button>
      </div>
    );
  }

  return (
    <form className="mx-4 mt-4 rounded-panel bg-raised p-4 md:mx-6" ref={formRef} onSubmit={createInvite} aria-label="Invite member">
      <div className="grid items-end gap-4 sm:grid-cols-[minmax(0,1fr)_10rem]">
        <Field label="Email" type="email" required autoFocus value={email} onChange={(event) => setEmail(event.target.value)} />
        <div className="sm:self-start">
          <span className="mb-1 block text-meta font-medium" aria-hidden="true">
            Role
          </span>
          <RoleSelect id="new-member-role" label="Role" value={role} options={canAddDirectly ? roles : roles.filter((item) => item !== "owner")} onChange={setRole} />
        </div>
      </div>
      <div className="mt-3">
        <FormError message={error} />
      </div>
      <div className="mt-2 flex flex-wrap gap-2">
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? "Creating…" : "Create invite link"}
        </button>
        {canAddDirectly && (
          <button className="btn" type="button" onClick={addDirectly} disabled={pending}>
            Add existing account
          </button>
        )}
        <button className="btn" type="button" onClick={onClose} disabled={pending}>
          Cancel
        </button>
      </div>
    </form>
  );
}
