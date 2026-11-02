import { t } from "@/lib/i18n";
export type Health = "on-track" | "at-risk" | "late";

export const healthMeta: Record<Health, { label: string; text: string; dot: string; rank: number }> = {
  late: { label: t("Late"), text: "text-ember", dot: "bg-ember", rank: 0 },
  "at-risk": { label: t("At risk"), text: "text-amber", dot: "bg-amber", rank: 1 },
  "on-track": { label: t("On track"), text: "text-ok", dot: "bg-ok", rank: 2 },
};
