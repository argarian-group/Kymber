import { useEffect, useRef, useState } from "react";
import { RichTextEditor } from "@/components/richtext/RichTextEditor";
import { RichTextView } from "@/components/richtext/RichTextView";
import { descriptionLimit } from "@/features/projects/tasks";

export function TaskDescription({ value, onSave }: { value: string; onSave: (description: string) => void }) {
  const [draft, setDraft] = useState<string | null>(null);
  const unsaved = useRef<{ draft: string | null; value: string; onSave: (description: string) => void }>({ draft, value, onSave });
  unsaved.current = { draft, value, onSave };

  useEffect(
    () => () => {
      const { draft: pending, value: saved, onSave: save } = unsaved.current;
      if (pending !== null && pending !== saved && pending.length <= descriptionLimit) save(pending);
    },
    [],
  );

  return (
    <section aria-labelledby="task-description">
      <div className="mb-2 flex items-center justify-between">
        <h3 id="task-description" className="font-medium">
          Description
        </h3>
        {draft === null && value && (
          <button className="btn" type="button" onClick={() => setDraft(value)}>
            Edit
          </button>
        )}
      </div>
      {draft !== null ? (
        <div className="grid grid-cols-1 gap-2">
          <RichTextEditor label="Description" placeholder="Add details, steps, links, tables…" initial={value} limit={descriptionLimit} size="min-h-48" autoFocus value={draft} onChange={setDraft} />
          <div className="flex gap-2">
            <button
              className="btn btn-primary"
              type="button"
              disabled={draft.length > descriptionLimit}
              onClick={() => {
                onSave(draft);
                setDraft(null);
              }}
            >
              Save
            </button>
            <button className="btn" type="button" onClick={() => setDraft(null)}>
              Cancel
            </button>
          </div>
        </div>
      ) : value ? (
        <RichTextView html={value} onToggleTask={onSave} />
      ) : (
        <button className="focus-ring min-h-16 w-full cursor-pointer rounded-control bg-sunken px-3 py-2 text-left text-ink-soft hover:text-ink" type="button" onClick={() => setDraft("")}>
          Add a description
        </button>
      )}
    </section>
  );
}
