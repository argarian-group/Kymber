import { type FormEvent, useState } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { useProjects } from "@/features/projects/ProjectsContext";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

function slugify(name: string) {
  return name
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 120)
    .replace(/-+$/, "");
}

export function CreateProjectForm({ onClose }: { onClose: () => void }) {
  const { createProject } = useProjects();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const { pending, error, setError, run } = useAction();
  const suggestedSlug = slugify(name);

  async function submit(event: FormEvent) {
    event.preventDefault();
    const finalSlug = slug.trim() || suggestedSlug;
    if (!finalSlug) {
      setError(t("Enter a slug using lowercase letters, numbers and hyphens."));
      return;
    }
    await run(async () => {
      await createProject({ name, slug: finalSlug });
      onClose();
    });
  }

  return (
    <form className="mx-4 mt-4 grid gap-4 rounded-panel bg-raised p-4 md:mx-6 md:grid-cols-2" onSubmit={submit} aria-label={t("New project")}>
      <Field label={t("Name")} required autoFocus maxLength={120} value={name} onChange={(event) => setName(event.target.value)} />
      <Field label={t("Slug")} maxLength={120} pattern="[a-z0-9\-]+" title={t("Lowercase letters, numbers and hyphens")} placeholder={suggestedSlug} hint={t("Lowercase letters, numbers and hyphens.")} value={slug} onChange={(event) => setSlug(event.target.value)} />
      <div className="md:col-span-2">
        <FormError message={error} />
        <div className="mt-2 flex gap-2">
          <button className="btn btn-primary" type="submit" disabled={pending}>
            {pending ? t("Creating…") : t("Create project")}
          </button>
          <button className="btn" type="button" onClick={onClose} disabled={pending}>
            {t("Cancel")}
          </button>
        </div>
      </div>
    </form>
  );
}
