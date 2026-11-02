import { Trash2 } from "lucide-react";
import { useState, type FormEvent } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { LoadError } from "@/components/ui/LoadError";
import { useEnvironments } from "@/features/projects/useEnvironments";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { t } from "@/lib/i18n";

export function ProjectEnvironments() {
  const { environments, error: loadError, path, reload } = useEnvironments();
  const [name, setName] = useState("");
  const { pending, error, run } = useAction();

  async function create(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      await api(path, { method: "POST", body: { name } });
      setName("");
      reload();
    });
  }

  async function remove(id: string) {
    await run(async () => {
      await api(`${path}/${id}`, { method: "DELETE" });
      reload();
    });
  }

  if (loadError) return <LoadError title={t("Could not load environments")} message={loadError} onRetry={reload} />;

  return (
    <SettingsSection title={t("Environments")} description={t("Apps in an environment share a private network. They cannot reach apps in other environments or apps without one.")}>
      <div className="grid gap-4 p-4">
        {environments && environments.length > 0 && (
          <ul className="grid gap-2">
            {environments.map(({ id, name }) => (
              <li className="flex items-center justify-between gap-3 rounded-panel border border-line p-2 pl-3" key={id}>
                <span className="min-w-0 break-words font-medium">{name}</span>
                <button className="btn flex-none px-2" type="button" disabled={pending} aria-label={t("Delete {name}", { name })} onClick={() => remove(id)}>
                  <Trash2 aria-hidden="true" size={16} />
                </button>
              </li>
            ))}
          </ul>
        )}
        <form className="flex flex-wrap items-end gap-2" onSubmit={create}>
          <div className="min-w-48 flex-1">
            <Field label={t("New environment")} required maxLength={64} placeholder="production" value={name} onChange={(event) => setName(event.target.value)} />
          </div>
          <button className="btn btn-primary" type="submit" disabled={pending}>
            {t("Add")}
          </button>
        </form>
        <FormError message={error} />
      </div>
    </SettingsSection>
  );
}
