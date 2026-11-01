import { Search } from "lucide-react";
import { priorities } from "@/features/dashboard/priorities";
import { type Board, type TaskFilters } from "@/features/projects/tasks";

type TaskFiltersBarProps = {
  board: Board;
  filters: TaskFilters;
  onChange: (name: keyof TaskFilters, value: string) => void;
  onClear: () => void;
};

export function TaskFiltersBar({ board, filters, onChange, onClear }: TaskFiltersBarProps) {
  const people = new Map(board.tasks.flatMap(({ assignees }) => assignees).map((person) => [person.id, person.username]));
  const active = Object.values(filters).some(Boolean);
  const select = "field w-auto min-w-0 flex-1 basis-[45%] cursor-pointer sm:flex-none sm:basis-auto";
  return (
    <div className="flex flex-wrap items-center gap-2" role="search">
      <label className="relative w-full sm:w-56">
        <Search className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-ink-soft" aria-hidden="true" size={14} />
        <span className="sr-only">Search tasks</span>
        <input className="field pl-8" type="search" placeholder="Search tasks" value={filters.search} onChange={(event) => onChange("search", event.target.value)} />
      </label>
      <select className={select} aria-label="Filter by assignee" value={filters.assignee} onChange={(event) => onChange("assignee", event.target.value)}>
        <option value="">Anyone</option>
        <option value="none">Unassigned</option>
        {[...people].map(([id, username]) => (
          <option value={id} key={id}>
            {username}
          </option>
        ))}
      </select>
      <select className={select} aria-label="Filter by label" value={filters.label} onChange={(event) => onChange("label", event.target.value)}>
        <option value="">Any label</option>
        {board.labels.map(({ id, name }) => (
          <option value={id} key={id}>
            {name}
          </option>
        ))}
      </select>
      <select className={select} aria-label="Filter by priority" value={filters.priority} onChange={(event) => onChange("priority", event.target.value)}>
        <option value="">Any priority</option>
        {Object.entries(priorities).map(([id, { label }]) => (
          <option value={id} key={id}>
            {label}
          </option>
        ))}
      </select>
      <select className={select} aria-label="Filter by due date" value={filters.due} onChange={(event) => onChange("due", event.target.value)}>
        <option value="">Any due date</option>
        <option value="late">Late</option>
        <option value="week">Next 7 days</option>
        <option value="none">No due date</option>
      </select>
      {active && (
        <button className="text-button text-meta" type="button" onClick={onClear}>
          Clear filters
        </button>
      )}
    </div>
  );
}
