import { useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { useProjects, type Workspace } from "@/features/projects/ProjectsContext";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { useAction } from "@/lib/useAction";

export function DeleteWorkspace({ workspace }: { workspace: Workspace }) {
  const { deleteWorkspace } = useProjects();
  const [confirming, setConfirming] = useState(false);
  const { pending, error, run } = useAction();

  async function confirmDelete() {
    if (!(await run(deleteWorkspace))) setConfirming(false);
  }

  return (
    <SettingsSection title="Delete workspace" description="Delete its projects first.">
      <div className="flex flex-wrap items-center justify-between gap-3 p-4">
        <p className="min-w-0 wrap-break-word">
          {confirming ? `Delete ${workspace.name}? This cannot be undone.` : `Permanently delete ${workspace.name}.`}
        </p>
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
            Delete workspace
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
