import { type FormEvent, useState } from "react";
import { Field } from "@/components/ui/Field";
import { useProjects } from "@/features/projects/ProjectsContext";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function EditProject() {
  const { project } = useProjectOutlet();
  const { canWrite, updateProject } = useProjects();
  const [name, setName] = useState(project.name);
  const [slug, setSlug] = useState(project.slug);
  const [saved, setSaved] = useState<string | null>(null);
  const { pending, error, run } = useAction();
  const changed = name.trim() !== project.name || slug.trim() !== project.slug;

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    await run(async () => {
      await updateProject(project.id, { name, slug });
      setSaved(t("Project saved."));
    });
  }

  return (
    <SettingsSection title={t("General")} description={canWrite ? undefined : t("Only admins and owners can edit this project.")}>
      <form onSubmit={submit}>
        <div className="grid items-start gap-4 p-4 sm:grid-cols-2">
          <Field label={t("Name")} required maxLength={120} disabled={!canWrite} value={name} onChange={(event) => setName(event.target.value)} />
          <Field label={t("Slug")} required maxLength={120} disabled={!canWrite} pattern="[a-z0-9\-]+" title={t("Lowercase letters, numbers and hyphens")} hint={t("Lowercase letters, numbers and hyphens.")} value={slug} onChange={(event) => setSlug(event.target.value)} />
        </div>
        {canWrite && (
          <FormFooter error={error} saved={saved}>
            <button className="btn btn-primary" type="submit" disabled={pending || !changed}>
              {pending ? t("Saving…") : t("Save project")}
            </button>
          </FormFooter>
        )}
      </form>
    </SettingsSection>
  );
}
