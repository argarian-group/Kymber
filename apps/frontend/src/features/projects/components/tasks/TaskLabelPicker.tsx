import { Pencil } from "lucide-react";
import { useState } from "react";
import { TaskLabelChip } from "@/features/projects/components/tasks/TaskLabelChip";
import { TaskLabelEditor } from "@/features/projects/components/tasks/TaskLabelEditor";
import { type TaskLabel } from "@/features/projects/tasks";
import { t } from "@/lib/i18n";

type TaskLabelPickerProps = {
  labels: TaskLabel[];
  selected: string[];
  path: string;
  onToggle: (labelId: string, checked: boolean) => void;
  onChanged: () => void;
};

export function TaskLabelPicker({ labels, selected, path, onToggle, onChanged }: TaskLabelPickerProps) {
  const [editing, setEditing] = useState<string | "new" | null>(null);
  const done = (changed: boolean) => {
    setEditing(null);
    if (changed) onChanged();
  };

  return (
    <fieldset>
      <legend className="mb-1 text-meta font-medium">{t("Labels")}</legend>
      <ul className="grid gap-1">
        {labels.map((label) =>
          editing === label.id ? (
            <li key={label.id}>
              <TaskLabelEditor label={label} path={path} onDone={done} />
            </li>
          ) : (
            <li className="flex items-center gap-2" key={label.id}>
              <label className="flex min-h-8 min-w-0 flex-1 cursor-pointer items-center gap-2">
                <input className="size-4 flex-none cursor-pointer accent-seafoam" type="checkbox" checked={selected.includes(label.id)} onChange={(event) => onToggle(label.id, event.target.checked)} />
                <TaskLabelChip label={label} />
              </label>
              <button className="btn min-h-7 border-transparent px-1.5" type="button" aria-label={t("Edit label {name}", { name: label.name })} onClick={() => setEditing(label.id)}>
                <Pencil aria-hidden="true" size={13} />
              </button>
            </li>
          ),
        )}
      </ul>
      <div className="mt-1">
        {editing === "new" ? (
          <TaskLabelEditor label={null} path={path} onDone={done} />
        ) : (
          <button className="text-button text-meta" type="button" onClick={() => setEditing("new")}>
            {t("New label")}
          </button>
        )}
      </div>
    </fieldset>
  );
}
