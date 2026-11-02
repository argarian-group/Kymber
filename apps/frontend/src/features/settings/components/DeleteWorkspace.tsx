import { useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { useProjects, type Workspace } from "@/features/projects/ProjectsContext";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function DeleteWorkspace({ workspace }: { workspace: Workspace }) {
  const { deleteWorkspace } = useProjects();
  const [confirming, setConfirming] = useState(false);
  const { pending, error, run } = useAction();

  async function confirmDelete() {
    if (!(await run(deleteWorkspace))) setConfirming(false);
  }

  return (
    <SettingsSection title={t("Delete workspace")} description={t("Delete its projects first.")}>
      <div className="flex flex-wrap items-center justify-between gap-3 p-4">
        <p className="min-w-0 wrap-break-word">
          {confirming ? t("Delete {name}? This cannot be undone.", { name: workspace.name }) : t("Permanently delete {name}.", { name: workspace.name })}
        </p>
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
            {t("Delete workspace")}
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
