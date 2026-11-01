import { type Priority } from "@/features/dashboard/data";

export const priorities: Record<Priority, { label: string; level: number; tone: string }> = {
  urgent: { label: "Urgent", level: 3, tone: "fill-ember" },
  high: { label: "High", level: 2, tone: "fill-ink" },
  normal: { label: "Normal", level: 1, tone: "fill-ink-soft" },
};
