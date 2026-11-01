export type Health = "on-track" | "at-risk" | "late";

export const healthMeta: Record<Health, { label: string; text: string; dot: string; rank: number }> = {
  late: { label: "Late", text: "text-ember", dot: "bg-ember", rank: 0 },
  "at-risk": { label: "At risk", text: "text-amber", dot: "bg-amber", rank: 1 },
  "on-track": { label: "On track", text: "text-ok", dot: "bg-ok", rank: 2 },
};
