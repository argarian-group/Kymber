import { useState } from "react";
import { useNavigate } from "react-router";
import { FormError } from "@/components/ui/FormError";
import { useProjects } from "@/features/projects/ProjectsContext";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function DeleteProject() {
  const { project } = useProjectOutlet();
  const { deleteProject } = useProjects();
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState(false);
  const { pending, error, run } = useAction();

  async function confirmDelete() {
    const deleted = await run(async () => {
      await deleteProject(project.id);
      navigate("/projects", { replace: true });
    });
    if (!deleted) setConfirming(false);
  }

  return (
    <SettingsSection title={t("Delete project")}>
      <div className="flex flex-wrap items-center justify-between gap-3 p-4">
        <p className="min-w-0 wrap-break-word">{confirming ? t("Delete {name}? This cannot be undone.", { name: project.name }) : t("Permanently delete {name}.", { name: project.name })}</p>
        {confirming ? (
          <span className="flex gap-2">
            <button className="btn btn-danger" type="button" onClick={confirmDelete} disabled={pending}>
              {pending ? t("Deleting…") : t("Confirm delete")}
            </button>
            <button className="btn" type="button" onClick={() => setConfirming(false)} disabled={pending}>
              {t("Cancel")}
            </button>
          </span>
        ) : (
          <button className="btn btn-danger" type="button" onClick={() => setConfirming(true)}>
            {t("Delete project")}
          </button>
        )}
      </div>
      {error && (
        <div className="border-t border-line px-4 py-3">
          <FormError message={error} />
        </div>
      )}
    </SettingsSection>
  );
}
