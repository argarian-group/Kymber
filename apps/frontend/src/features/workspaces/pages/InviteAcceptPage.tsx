import { Link, useLocation, useNavigate } from "react-router";
import { FormError } from "@/components/ui/FormError";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { useProjects } from "@/features/projects/ProjectsContext";
import { type Acceptance } from "@/features/workspaces/types";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function InviteAcceptPage() {
  const user = useCurrentUser();
  const { selectWorkspace } = useProjects();
  const navigate = useNavigate();
  const token = useLocation().hash.slice(1);
  const { pending, error, run } = useAction();

  async function accept() {
    await run(async () => {
      const { workspace_id } = await api<Acceptance>("/invitations/accept", { method: "POST", body: { token } });
      selectWorkspace(workspace_id);
      navigate("/projects", { replace: true });
    });
  }

  return (
    <div className="max-w-md px-4 py-8 md:px-6">
      <h1 className="page-title">{t("Join a workspace")}</h1>
      <p className="mt-1 mb-4 text-ink-soft">{t("The invitation must have been created for {email}.", { email: user.email })}</p>
      <FormError message={error} />
      <div className="mt-3 flex flex-wrap gap-2">
        <button className="btn btn-primary" type="button" onClick={accept} disabled={pending}>
          {pending ? t("Accepting…") : t("Accept invitation")}
        </button>
        <Link className="btn" to="/">
          {t("Not now")}
        </Link>
      </div>
    </div>
  );
}
