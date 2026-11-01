import { Panel } from "@/components/ui/Panel";
import { type OverviewTask } from "@/features/dashboard/overview";

export function TaskSummaryPanel({ tasks }: { tasks: OverviewTask[] }) {
  const done = tasks.filter((item) => item.done).length;
  const late = tasks.filter((item) => item.late).length;
  const percent = tasks.length ? Math.round((done / tasks.length) * 100) : 0;
  const stats = [
    { name: "Open", value: tasks.length - done, tone: "" },
    { name: "Late", value: late, tone: late > 0 ? "text-ember" : "" },
    { name: "Done", value: done, tone: "" },
  ];

  return (
    <Panel title="Tasks" className="lg:col-span-2" aside={<span className="text-meta text-ink-soft tabular-nums">{tasks.length} total</span>}>
      <div className="mx-4 mb-3 h-2 overflow-hidden rounded-full bg-white/10 md:mx-6" role="progressbar" aria-label="Tasks done" aria-valuenow={percent} aria-valuemin={0} aria-valuemax={100}>
        <div className="h-full rounded-full bg-seafoam" style={{ width: `${percent}%` }} />
      </div>
      <dl className="grid grid-cols-3 gap-2 px-4 md:px-6">
        {stats.map(({ name, value, tone }) => (
          <div className="min-w-0" key={name}>
            <dt className="truncate text-xs text-ink-soft">{name}</dt>
            <dd className={`font-display text-lg font-semibold tabular-nums ${tone}`}>{value}</dd>
          </div>
        ))}
      </dl>
    </Panel>
  );
}
