import { FormError } from "@/components/ui/FormError";
import { t } from "@/lib/i18n";
import { MetricsGrid, type ChartSpec } from "./MetricsGrid";
import { formatPercent, formatRate } from "./metrics";
import { useMetrics } from "./useMetrics";

const charts: ChartSpec[] = [
  { key: "cpu", title: t("CPU"), format: formatPercent, max: 100 },
  { key: "memory", title: t("Memory"), format: formatPercent, max: 100 },
  { key: "disk", title: t("Disk"), format: formatPercent, max: 100 },
  { key: "network_in", title: t("Network in"), format: formatRate },
];

export function ServerMetrics({ serverId }: { serverId: string }) {
  const { metrics, error } = useMetrics(`/servers/${serverId}/metrics`, "1h");

  if (error && !metrics) return <FormError message={error} />;
  if (!metrics) return <p className="text-meta text-ink-soft">{t("Loading metrics…")}</p>;
  return <MetricsGrid metrics={metrics} charts={charts} className="grid gap-2 sm:grid-cols-2 xl:grid-cols-4" height={56} />;
}
