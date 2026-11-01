import { type ReactNode } from "react";

export function ProjectDetail({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="text-xs text-ink-soft">{label}</dt>
      <dd className="mt-0.5 wrap-break-word">{children}</dd>
    </div>
  );
}
