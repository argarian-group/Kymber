export function ProgressRing({ name, value, tone }: { name: string; value: number; tone: string }) {
  return (
    <span
      className="flex items-center gap-2 text-meta font-medium tabular-nums"
      role="progressbar"
      aria-label={`${name} progress`}
      aria-valuenow={value}
      aria-valuemin={0}
      aria-valuemax={100}
    >
      <svg className={`size-5 flex-none -rotate-90 ${tone}`} viewBox="0 0 16 16" fill="none" strokeWidth="2.5" aria-hidden="true">
        <circle cx="8" cy="8" r="6" className="stroke-white/12" />
        <circle cx="8" cy="8" r="6" stroke="currentColor" pathLength="100" strokeDasharray={`${value} 100`} />
      </svg>
      {value}%
    </span>
  );
}
