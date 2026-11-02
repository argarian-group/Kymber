import { t } from "@/lib/i18n";
export type Priority = "urgent" | "high" | "normal";

export const priorities: Record<Priority, { label: string; level: number; tone: string }> = {
  urgent: { label: t("Urgent"), level: 3, tone: "fill-ember" },
  high: { label: t("High"), level: 2, tone: "fill-ink" },
  normal: { label: t("Normal"), level: 1, tone: "fill-ink-soft" },
};
