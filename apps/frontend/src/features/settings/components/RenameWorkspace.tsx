import { type FormEvent, useState } from "react";
import { Field } from "@/components/ui/Field";
import { useProjects, type Workspace } from "@/features/projects/ProjectsContext";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function RenameWorkspace({ workspace }: { workspace: Workspace }) {
  const { renameWorkspace, can } = useProjects();
  const canWrite = can("workspace.manage");
  const [name, setName] = useState(workspace.name);
  const [saved, setSaved] = useState<string | null>(null);
  const { pending, error, run } = useAction();
  const changed = name.trim() !== workspace.name;

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    await run(async () => {
      await renameWorkspace(name);
      setName(name.trim());
      setSaved(t("Workspace renamed."));
    });
  }

  return (
    <SettingsSection title={t("General")} description={[!canWrite && t("Your role cannot rename this workspace."), workspace.created_by && t("Created by {name}.", { name: workspace.created_by.username })].filter(Boolean).join(" ")}>
      <form onSubmit={submit}>
        <div className="p-4">
          <Field label={t("Workspace name")} required maxLength={120} disabled={!canWrite} value={name} onChange={(event) => setName(event.target.value)} />
        </div>
        {canWrite && (
          <FormFooter error={error} saved={saved}>
            <button className="btn btn-primary" type="submit" disabled={pending || !changed}>
              {pending ? t("Saving…") : t("Save name")}
            </button>
          </FormFooter>
        )}
      </form>
    </SettingsSection>
  );
}
