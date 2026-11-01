import { Trash2 } from "lucide-react";
import { AddInline } from "@/features/projects/components/tasks/AddInline";
import { type Checklist, type ChecklistItem } from "@/features/projects/tasks";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { FormError } from "@/components/ui/FormError";

export function TaskChecklistCard({ checklist, path, onChanged }: { checklist: Checklist; path: string; onChanged: () => void }) {
  const { error, run } = useAction();
  const done = checklist.items.filter((item) => item.done).length;
  const percent = checklist.items.length ? Math.round((done / checklist.items.length) * 100) : 0;
  const change = (action: () => Promise<unknown>) =>
    run(async () => {
      await action();
      onChanged();
    });
  const update = (item: ChecklistItem, input: { title: string; done: boolean }) => change(() => api(`${path}/task-checklist-items/${item.id}`, { method: "PUT", body: input }));

  return (
    <section aria-label={checklist.title}>
      <div className="flex items-center gap-2">
        <input
          className="min-w-0 flex-1 rounded-sm bg-transparent font-medium focus-ring"
          aria-label="Checklist title"
          maxLength={120}
          defaultValue={checklist.title}
          onBlur={(event) => {
            const title = event.target.value.trim();
            if (title && title !== checklist.title) change(() => api(`${path}/task-checklists/${checklist.id}`, { method: "PUT", body: { title } }));
            else event.target.value = checklist.title;
          }}
        />
        <button className="btn min-h-7 border-transparent px-1.5" type="button" aria-label={`Delete checklist ${checklist.title}`} onClick={() => change(() => api(`${path}/task-checklists/${checklist.id}`, { method: "DELETE" }))}>
          <Trash2 aria-hidden="true" size={14} />
        </button>
      </div>
      <div className="mt-1.5 flex items-center gap-2 text-xs text-ink-soft tabular-nums">
        <span className="w-8">{percent}%</span>
        <span className="h-1.5 flex-1 overflow-hidden rounded-full bg-white/10">
          <span className="block h-full bg-seafoam" style={{ width: `${percent}%` }} />
        </span>
      </div>
      <ul className="mt-2 grid gap-0.5">
        {checklist.items.map((item) => (
          <li className="flex min-h-9 items-center gap-2 rounded-control pl-1 hover:bg-white/5" key={item.id}>
            <input className="size-4 flex-none cursor-pointer accent-seafoam" type="checkbox" aria-label={`Done: ${item.title}`} checked={item.done} onChange={(event) => update(item, { title: item.title, done: event.target.checked })} />
            <input
              className={`min-w-0 flex-1 rounded-sm bg-transparent py-1 focus-ring ${item.done ? "text-ink-faint line-through" : ""}`}
              aria-label="Item title"
              maxLength={300}
              defaultValue={item.title}
              onBlur={(event) => {
                const title = event.target.value.trim();
                if (title && title !== item.title) update(item, { title, done: item.done });
                else event.target.value = item.title;
              }}
            />
            <button className="btn min-h-7 border-transparent px-1.5" type="button" aria-label={`Delete item ${item.title}`} onClick={() => change(() => api(`${path}/task-checklist-items/${item.id}`, { method: "DELETE" }))}>
              <Trash2 aria-hidden="true" size={14} />
            </button>
          </li>
        ))}
      </ul>
      <FormError message={error} />
      <div className="mt-1">
        <AddInline
          label="Add item"
          placeholder="Item title"
          maxLength={300}
          keepOpen
          onAdd={async (title) => {
            await api(`${path}/task-checklists/${checklist.id}/items`, { method: "POST", body: { title, done: false } });
            onChanged();
          }}
        />
      </div>
    </section>
  );
}
