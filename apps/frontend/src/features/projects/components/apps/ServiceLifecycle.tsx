const steps = ["Draft", "Queued", "Working", "Running"];

const positions: Record<string, number> = { draft: 0, queued: 1, deploying: 2, failed: 2, running: 3, stopped: 3 };

export function ServiceLifecycle({ status }: { status: string }) {
  const current = positions[status] ?? 0;
  const labels = steps.map((step, index) => (index === 3 && status === "stopped" ? "Stopped" : index === 2 && status === "failed" ? "Failed" : step));

  return (
    <ol className="grid grid-cols-4 gap-1" aria-label={`Status: ${labels[current]}`}>
      {labels.map((label, index) => {
        const active = index === current;
        const tone = status === "failed" && active ? "bg-ember" : status === "stopped" && active ? "bg-ink-faint" : index <= current ? "bg-seafoam" : "bg-white/12";
        const moving = active && (status === "queued" || status === "deploying");
        return (
          <li className="min-w-0" aria-current={active ? "step" : undefined} key={label}>
            <span className={`block h-1 rounded-xs ${tone} ${moving ? "motion-safe:animate-pulse" : ""}`} aria-hidden="true" />
            <span className={`mt-1.5 block truncate text-xs ${active ? "font-medium text-ink" : "text-ink-faint"}`}>{label}</span>
          </li>
        );
      })}
    </ol>
  );
}
