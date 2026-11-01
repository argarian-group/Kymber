import { useState } from "react";
import { useNavigate } from "react-router";
import { FormError } from "@/components/ui/FormError";
import { useProjects } from "@/features/projects/ProjectsContext";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { useAction } from "@/lib/useAction";

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
    <SettingsSection title="Delete project">
      <div className="flex flex-wrap items-center justify-between gap-3 p-4">
        <p className="min-w-0 wrap-break-word">{confirming ? `Delete ${project.name}? This cannot be undone.` : `Permanently delete ${project.name}.`}</p>
        {confirming ? (
          <span className="flex gap-2">
            <button className="btn btn-danger" type="button" onClick={confirmDelete} disabled={pending}>
              {pending ? "Deleting…" : "Confirm delete"}
            </button>
            <button className="btn" type="button" onClick={() => setConfirming(false)} disabled={pending}>
              Cancel
            </button>
          </span>
        ) : (
          <button className="btn btn-danger" type="button" onClick={() => setConfirming(true)}>
            Delete project
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
