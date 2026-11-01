export function PriorityBars({ level, tone }: { level: number; tone: string }) {
  return (
    <svg className="size-4" viewBox="0 0 16 16" aria-hidden="true">
      {[1, 2, 3].map((bar) => (
        <rect className={bar <= level ? tone : "fill-white/15"} x={bar * 4 - 3} y={12 - bar * 3} width="3" height={bar * 3 + 1} rx="0.5" key={bar} />
      ))}
    </svg>
  );
}
