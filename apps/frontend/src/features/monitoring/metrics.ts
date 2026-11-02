import { language, t, type TranslationKey } from "@/lib/i18n";

export type Point = { t: number; v: number };

export type MetricsResponse = { series: Record<string, Point[]>; http?: boolean };

export type MetricRange = "1h" | "6h" | "24h" | "7d";

export const metricRanges: MetricRange[] = ["1h", "6h", "24h", "7d"];

export type Alert = { name: string; server_id?: string; server_name?: string; service_id?: string; service_name?: string; project_id?: string; since: string };

const alertNames: Record<string, TranslationKey> = {
  ServerDown: "Monitoring is not responding",
  DiskAlmostFull: "Disk almost full",
  MemoryHigh: "Memory almost full",
  AppRestartLoop: "Restarting repeatedly",
  AppHttpErrors: "Many server errors (5xx)",
};

export function alertLabel(name: string) {
  return alertNames[name] ? t(alertNames[name]) : name;
}

const units = ["B", "KB", "MB", "GB", "TB"];

export function formatBytes(value: number) {
  let size = value;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size.toLocaleString(language, { maximumFractionDigits: size < 10 && unit > 0 ? 1 : 0 })} ${units[unit]}`;
}

export const formatPercent = (value: number) => `${value.toLocaleString(language, { maximumFractionDigits: value > 0 && value < 1 ? 2 : 1 })}%`;

export const formatRate = (value: number) => `${formatBytes(value)}/s`;

export const formatRequests = (value: number) => t("{count} req/s", { count: value.toLocaleString(language, { maximumFractionDigits: 2 }) });

export const formatSeconds = (value: number) => (value < 1 ? `${Math.round(value * 1000)} ms` : `${value.toLocaleString(language, { maximumFractionDigits: 2 })} s`);
