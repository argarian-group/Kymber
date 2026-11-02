import { t } from "@/lib/i18n";
export const engines: Record<string, { label: string; description: string }> = {
  mysql: { label: "MySQL", description: t("Relational database") },
  redis: { label: "Redis", description: t("In-memory data store") },
  docker: { label: t("Docker image"), description: t("Any public image from Docker Hub") },
};

export type AppCategory = "application" | "database" | "service";

export const appCategories: { id: AppCategory; label: string; item: string }[] = [
  { id: "application", label: t("Applications"), item: t("application") },
  { id: "database", label: t("Databases"), item: t("database") },
  { id: "service", label: t("Services"), item: t("service") },
];

export function engineLabel(engine: string) {
  return engines[engine]?.label ?? engine;
}
