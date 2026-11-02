import { LineChart } from "@/components/ui/LineChart";
import type { MetricsResponse } from "./metrics";

export type ChartSpec = { key: string; title: string; format: (value: number) => string; max?: number };

export function MetricsGrid({ metrics, charts, className = "grid gap-3 sm:grid-cols-2 xl:grid-cols-3", height }: { metrics: MetricsResponse; charts: ChartSpec[]; className?: string; height?: number }) {
  return (
    <div className={className}>
      {charts.map(({ key, title, format, max }) => (
        <LineChart title={title} points={metrics.series[key] ?? []} format={format} max={max} height={height} key={key} />
      ))}
    </div>
  );
}
