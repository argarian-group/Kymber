import { useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { RoleSelect } from "@/features/workspaces/components/RoleSelect";
import { type Member } from "@/features/workspaces/types";
import { api } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { useAction } from "@/lib/useAction";
import { t, translate } from "@/lib/i18n";

export function MemberRow({ member, workspaceId, isCreator, canManage, onChanged }: { member: Member; workspaceId: string; isCreator: boolean; canManage: boolean; onChanged: (member: Member, removed: boolean) => void }) {
  const user = useCurrentUser();
  const isSelf = member.user_id === user.id;
  const [confirming, setConfirming] = useState(false);
  const { pending, error, run } = useAction();

  function changeRole(role: string) {
    return run(async () => {
      const response = await api<{ member: Member }>(`/workspaces/${workspaceId}/members/${member.user_id}`, { method: "PUT", body: { role } });
      onChanged(response.member, false);
    });
  }

  async function remove() {
    const removed = await run(async () => {
      await api(`/workspaces/${workspaceId}/members/${member.user_id}`, { method: "DELETE" });
      onChanged(member, true);
    });
    if (!removed) setConfirming(false);
  }

  return (
    <li className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-line px-4 py-3 md:px-6">
      <div className="min-w-0 flex-1 basis-48">
        <p className="font-medium wrap-break-word">
          {member.username}
          {isSelf && <span className="ml-2 text-xs font-normal text-ink-soft">{t("You")}</span>}
          {isCreator && <span className="ml-2 text-xs font-normal text-ink-soft">{t("Creator")}</span>}
        </p>
        <p className="text-meta wrap-break-word text-ink-soft">{member.email}</p>
        <FormError message={error} />
      </div>
      <span className="text-meta text-ink-soft tabular-nums">{t("Joined {date}", { date: formatDate(member.created_at) })}</span>
      {canManage && !isCreator ? (
        <>
          <div className="w-28">
            <RoleSelect id={`role-${member.user_id}`} label={t("Role of {name}", { name: member.username })} value={member.role} disabled={pending} onChange={changeRole} />
          </div>
          {confirming ? (
            <span className="flex gap-2">
              <button className="btn btn-danger" type="button" onClick={remove} disabled={pending} aria-label={t("Confirm remove {name}", { name: member.username })}>
                {pending ? t("Removing…") : t("Confirm remove")}
              </button>
              <button className="btn" type="button" onClick={() => setConfirming(false)} disabled={pending}>
                {t("Cancel")}
              </button>
            </span>
          ) : (
            <button className="btn" type="button" onClick={() => setConfirming(true)} aria-label={t("Remove {name}", { name: member.username })}>
              {t("Remove")}
            </button>
          )}
        </>
      ) : (
        <span className="text-meta font-medium">{translate(member.role)}</span>
      )}
    </li>
  );
}
