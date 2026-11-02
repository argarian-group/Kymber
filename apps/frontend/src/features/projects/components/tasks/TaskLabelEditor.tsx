import { type FormEvent, useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { useProjects } from "@/features/projects/ProjectsContext";
import { labelColors, type LabelColor, type TaskLabel } from "@/features/projects/tasks";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function TaskLabelEditor({ label, path, onDone }: { label: TaskLabel | null; path: string; onDone: (changed: boolean) => void }) {
  const { canWrite } = useProjects();
  const [name, setName] = useState(label?.name ?? "");
  const [color, setColor] = useState<LabelColor>(label?.color ?? "teal");
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    const saved = await run(() => api(label ? `${path}/task-labels/${label.id}` : `${path}/task-labels`, { method: label ? "PUT" : "POST", body: { name, color } }));
    if (saved) onDone(true);
  }

  return (
    <form className="grid gap-2 rounded-control bg-sunken p-2" onSubmit={submit}>
      <input className="field" aria-label={t("Label name")} placeholder={t("Label name")} required autoFocus maxLength={40} value={name} onChange={(event) => setName(event.target.value)} />
      <div className="flex flex-wrap gap-1" role="group" aria-label={t("Label color")}>
        {(Object.keys(labelColors) as LabelColor[]).map((option) => (
          <button className={`focus-ring size-7 cursor-pointer rounded-sm ${color === option ? "outline-2 outline-offset-1 outline-ink" : ""}`} style={{ backgroundColor: labelColors[option] }} type="button" aria-label={t(option)} aria-pressed={color === option} onClick={() => setColor(option)} key={option} />
        ))}
      </div>
      <FormError message={error} />
      <div className="flex flex-wrap gap-2">
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {t("Save")}
        </button>
        <button className="btn" type="button" onClick={() => onDone(false)}>
          {t("Cancel")}
        </button>
        {label && canWrite && (
          <button
            className="btn btn-danger"
            type="button"
            disabled={pending}
            onClick={async () => {
              if (await run(() => api(`${path}/task-labels/${label.id}`, { method: "DELETE" }))) onDone(true);
            }}
          >
            {t("Delete")}
          </button>
        )}
      </div>
    </form>
  );
}
