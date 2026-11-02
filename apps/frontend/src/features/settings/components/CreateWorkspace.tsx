import { type FormEvent, useState } from "react";
import { Field } from "@/components/ui/Field";
import { useProjects } from "@/features/projects/ProjectsContext";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function CreateWorkspace() {
  const { createWorkspace } = useProjects();
  const [name, setName] = useState("");
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      await createWorkspace(name);
      setName("");
    });
  }

  return (
    <SettingsSection title={t("New workspace")} description={t("You become its owner and switch to it.")}>
      <form onSubmit={submit}>
        <div className="p-4">
          <Field label={t("Workspace name")} required maxLength={120} value={name} onChange={(event) => setName(event.target.value)} />
        </div>
        <FormFooter error={error} saved={null}>
          <button className="btn btn-primary" type="submit" disabled={pending}>
            {pending ? t("Creating…") : t("Create workspace")}
          </button>
        </FormFooter>
      </form>
    </SettingsSection>
  );
}
