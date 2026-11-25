import { type KeyboardEvent, useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { Modal } from "@/components/ui/Modal";
import { useProjects } from "@/features/projects/ProjectsContext";
import { taskLink, type WorkspaceTask } from "@/features/tasks/workspaceTasks";
import { api, errorMessage } from "@/lib/api";
import { t } from "@/lib/i18n";

function Palette({ workspaceId, onClose }: { workspaceId: string; onClose: () => void }) {
  const navigate = useNavigate();
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<WorkspaceTask[]>([]);
  const [active, setActive] = useState(0);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const q = query.trim();
    if (!q) {
      setResults([]);
      return;
    }
    let current = true;
    const timer = window.setTimeout(() => {
      api<{ tasks: WorkspaceTask[] }>(`/workspaces/${workspaceId}/tasks?q=${encodeURIComponent(q)}`)
        .then(({ tasks }) => { if (current) { setResults(tasks); setActive(0); setError(null); } })
        .catch((failure) => { if (current) setError(errorMessage(failure)); });
    }, 150);
    return () => { current = false; window.clearTimeout(timer); };
  }, [query, workspaceId]);

  function open(task: WorkspaceTask) {
    onClose();
    navigate(taskLink(task));
  }

  function onKeyDown(event: KeyboardEvent) {
    if (event.key === "ArrowDown") setActive((index) => Math.min(index + 1, results.length - 1));
    else if (event.key === "ArrowUp") setActive((index) => Math.max(index - 1, 0));
    else if (event.key === "Enter" && results[active]) open(results[active]);
    else return;
    event.preventDefault();
  }

  return (
    <Modal label={t("Search tasks")} className="mt-[15vh] mb-auto max-w-xl" onClose={onClose}>
      <input
        className="w-full border-b border-line bg-transparent px-4 py-3 outline-none"
        autoFocus
        role="combobox"
        aria-expanded={results.length > 0}
        aria-controls="task-palette-results"
        aria-activedescendant={results[active] ? `palette-${results[active].id}` : undefined}
        placeholder={t("Search tasks by title or key")}
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        onKeyDown={onKeyDown}
      />
      {error && <p className="p-4 text-meta text-ember" role="alert">{error}</p>}
      {query.trim() && !error && results.length === 0 && <p className="p-4 text-meta text-ink-soft">{t("No tasks found.")}</p>}
      <ul id="task-palette-results" role="listbox" className="max-h-80 overflow-y-auto empty:hidden">
        {results.map((task, index) => (
          <li id={`palette-${task.id}`} role="option" aria-selected={index === active} key={task.id}>
            <button className={`flex w-full cursor-pointer items-center gap-3 px-4 py-2 text-left ${index === active ? "bg-white/10" : ""}`} type="button" tabIndex={-1} onMouseEnter={() => setActive(index)} onClick={() => open(task)}>
              <span className="font-mono text-xs text-ink-faint">{task.key}</span>
              <span className={`min-w-0 flex-1 truncate ${task.done ? "text-ink-faint line-through" : ""}`}>{task.title}</span>
              <span className="truncate text-xs text-ink-soft">{task.project_name}</span>
            </button>
          </li>
        ))}
      </ul>
    </Modal>
  );
}

export function TaskPalette() {
  const { state } = useProjects();
  const workspaceId = state.status === "ready" ? state.workspace?.id : undefined;
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const onKey = (event: globalThis.KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setOpen(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  return open && workspaceId ? <Palette workspaceId={workspaceId} onClose={() => setOpen(false)} /> : null;
}
