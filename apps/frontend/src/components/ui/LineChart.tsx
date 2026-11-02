import { useState } from "react";
import { language, t } from "@/lib/i18n";

type Point = { t: number; v: number };

const width = 600;

export function LineChart({ title, points, format, max, height = 96 }: { title: string; points: Point[]; format: (value: number) => string; max?: number; height?: number }) {
  const [hover, setHover] = useState<number | null>(null);
  const latest = points.at(-1);
  const top = max ?? (Math.max(...points.map(({ v }) => v), 0) * 1.15 || 1);
  const first = points[0]?.t ?? 0;
  const span = (latest?.t ?? 1) - first || 1;
  const x = (point: Point) => ((point.t - first) / span) * width;
  const y = (point: Point) => height - (Math.min(point.v, top) / top) * height;
  const line = points.map((point, index) => `${index ? "L" : "M"}${x(point).toFixed(1)},${y(point).toFixed(1)}`).join("");
  const active = hover === null ? null : points[hover];
  const time = (point: Point) => new Date(point.t * 1000).toLocaleString(language, { hour: "2-digit", minute: "2-digit", day: span > 86400 ? "2-digit" : undefined, month: span > 86400 ? "short" : undefined });

  function track(event: React.PointerEvent<SVGSVGElement>) {
    if (points.length === 0) return;
    const box = event.currentTarget.getBoundingClientRect();
    const target = first + ((event.clientX - box.left) / box.width) * span;
    let nearest = 0;
    points.forEach((point, index) => {
      if (Math.abs(point.t - target) < Math.abs(points[nearest].t - target)) nearest = index;
    });
    setHover(nearest);
  }

  return (
    <figure className="min-w-0 rounded-panel bg-raised p-3">
      <figcaption className="flex items-baseline justify-between gap-3">
        <span className="truncate text-xs text-ink-soft">{title}</span>
        <span className="font-display text-base font-semibold tabular-nums">{active ? format(active.v) : latest ? format(latest.v) : "—"}</span>
      </figcaption>
      <p className="h-4 text-xs text-ink-faint tabular-nums">{active ? time(active) : ""}</p>
      {points.length < 2 ? (
        <p className="flex items-center text-xs text-ink-faint" style={{ height }}>
          {t("No data yet")}
        </p>
      ) : (
        <svg
          className="block w-full touch-none overflow-visible text-seafoam"
          viewBox={`0 0 ${width} ${height}`}
          preserveAspectRatio="none"
          style={{ height }}
          role="img"
          aria-label={t("{title}: {value} now, up to {peak} in this period", { title, value: format(latest!.v), peak: format(Math.max(...points.map(({ v }) => v))) })}
          onPointerMove={track}
          onPointerLeave={() => setHover(null)}
        >
          <line x1="0" x2={width} y1={height} y2={height} className="stroke-line-strong" vectorEffect="non-scaling-stroke" />
          <line x1="0" x2={width} y1={height / 2} y2={height / 2} className="stroke-line" strokeDasharray="2 4" vectorEffect="non-scaling-stroke" />
          <path d={`${line}L${width},${height}L0,${height}Z`} fill="currentColor" opacity="0.12" />
          <path d={line} fill="none" stroke="currentColor" strokeWidth="2" strokeLinejoin="round" vectorEffect="non-scaling-stroke" />
          {active && <line x1={x(active)} x2={x(active)} y1="0" y2={height} className="stroke-ink-faint" vectorEffect="non-scaling-stroke" />}
        </svg>
      )}
    </figure>
  );
}
