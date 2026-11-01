import { useState } from "react";

export function TaskDescription({ value, onSave }: { value: string; onSave: (description: string) => void }) {
  const [draft, setDraft] = useState<string | null>(null);

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
        <div className="grid gap-2">
          <textarea className="field min-h-36 py-2" aria-label="Description" autoFocus maxLength={20000} value={draft} onChange={(event) => setDraft(event.target.value)} />
          <div className="flex gap-2">
            <button
              className="btn btn-primary"
              type="button"
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
        <p className="wrap-break-word whitespace-pre-wrap">{value}</p>
      ) : (
        <button className="focus-ring min-h-16 w-full cursor-pointer rounded-control bg-sunken px-3 py-2 text-left text-ink-soft hover:text-ink" type="button" onClick={() => setDraft("")}>
          Add a description
        </button>
      )}
    </section>
  );
}
