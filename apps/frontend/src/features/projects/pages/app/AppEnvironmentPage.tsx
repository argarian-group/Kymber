import { useState, type FormEvent } from "react";
import { Navigate } from "react-router";
import { CopyField } from "@/components/ui/CopyField";
import { EmptyState } from "@/components/ui/EmptyState";
import { Field } from "@/components/ui/Field";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t, translate } from "@/lib/i18n";

export function AppEnvironmentPage() {
  const { service, detail, template, path, canWrite, onChanged } = useAppOutlet();
  const [edits, setEdits] = useState<Record<string, string>>({});
  const [saved, setSaved] = useState<string | null>(null);
  const { pending, error, run } = useAction();

  if (!canWrite) return <Navigate to=".." replace />;
  if (!detail || !template) return <PageSkeleton />;

  const environment = detail.environment ?? {};
  const draft = service.status === "draft";
  const changed = Object.entries(edits).some(([name, value]) => value !== environment[name]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    await run(async () => {
      await api(`${path}/${service.id}`, { method: "PUT", body: { environment: edits } });
      setEdits({});
      setSaved(t("Saved."));
      onChanged();
    });
  }

  if (template.fields.length === 0) return <EmptyState title={t("No environment variables")} />;

  return (
    <AppSection title={t("Environment variables")} description={draft ? t("These go into the container at deploy.") : t("Variables can only change before the first deploy.")}>
      {draft ? (
        <form onSubmit={submit}>
          <div className="grid items-start gap-4 p-4 sm:grid-cols-2">
            {template.fields.map((field) => (
              <Field
                label={translate(field.label)}
                type={field.secret ? "password" : "text"}
                autoComplete="off"
                required={field.required}
                minLength={field.min || undefined}
                maxLength={field.max || undefined}
                pattern={field.pattern?.replace(/^\^|\$$/g, "")}
                hint={field.name}
                value={edits[field.name] ?? environment[field.name] ?? ""}
                onChange={(event) => setEdits({ ...edits, [field.name]: event.target.value })}
                key={field.name}
              />
            ))}
          </div>
          <FormFooter error={error} saved={saved}>
            <button className="btn btn-primary" type="submit" disabled={pending || !changed}>
              {pending ? t("Saving…") : t("Save variables")}
            </button>
          </FormFooter>
        </form>
      ) : (
        <div className="grid gap-4 p-4">
          {template.fields.map((field) => (
            <CopyField label={field.name} value={environment[field.name] ?? ""} secret={field.secret} key={field.name} />
          ))}
        </div>
      )}
    </AppSection>
  );
}
