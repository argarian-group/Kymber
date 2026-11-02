import { useState } from "react";
import { Link } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormError } from "@/components/ui/FormError";
import { MetricsGrid, type ChartSpec } from "@/features/monitoring/MetricsGrid";
import { formatBytes, formatPercent, formatRate, formatRequests, formatSeconds, metricRanges, type MetricRange } from "@/features/monitoring/metrics";
import { useMetrics } from "@/features/monitoring/useMetrics";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { t } from "@/lib/i18n";

const resourceCharts: ChartSpec[] = [
  { key: "cpu", title: t("CPU (% of one core)"), format: formatPercent },
  { key: "memory", title: t("Memory"), format: formatBytes },
  { key: "network_in", title: t("Network in"), format: formatRate },
  { key: "network_out", title: t("Network out"), format: formatRate },
];

const httpCharts: ChartSpec[] = [
  { key: "requests", title: t("Requests"), format: formatRequests },
  { key: "errors", title: t("Server errors (5xx)"), format: formatPercent, max: 100 },
  { key: "latency_p95", title: t("Response time (p95)"), format: formatSeconds },
];

export function AppMetricsPage() {
  const { service, path, canWrite } = useAppOutlet();
  const [range, setRange] = useState<MetricRange>("1h");
  const { metrics, error } = useMetrics(service.server_id ? `${path}/${service.id}/metrics` : null, range);

  if (!service.server_id) return <EmptyState title={t("No metrics yet")} description={t("Metrics appear here after the first deploy.")} />;

  return (
    <div className="grid gap-6">
      <div className="flex flex-wrap gap-1" role="group" aria-label={t("Period")}>
        {metricRanges.map((value) => (
          <button className={`btn min-h-8 px-2.5 ${range === value ? "btn-primary" : ""}`} type="button" aria-pressed={range === value} onClick={() => setRange(value)} key={value}>
            {value}
          </button>
        ))}
      </div>
      {error && !metrics && <FormError message={error} />}
      {!metrics && !error && <p className="text-meta text-ink-soft">{t("Loading metrics…")}</p>}
      {metrics && (
        <>
          <AppSection title={t("Resources")} description={t("Usage of this app's container, refreshed every 30 seconds.")}>
            <MetricsGrid metrics={metrics} charts={resourceCharts} className="grid gap-3 p-3 sm:grid-cols-2" />
          </AppSection>
          <AppSection title={t("HTTP traffic")} description={metrics.http ? t("Requests that reach this app through its public domain.") : undefined}>
            {metrics.http ? (
              <MetricsGrid metrics={metrics} charts={httpCharts} className="grid gap-3 p-3 sm:grid-cols-2 xl:grid-cols-3" />
            ) : (
              <p className="p-4 text-meta text-ink-soft">
                {t("HTTP metrics need a public domain.")}{" "}
                {canWrite && (
                  <Link className="text-button" to="../connection">
                    {t("Set a domain")}
                  </Link>
                )}
              </p>
            )}
          </AppSection>
        </>
      )}
    </div>
  );
}
