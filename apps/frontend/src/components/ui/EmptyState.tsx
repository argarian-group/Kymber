import type { ReactNode } from "react";

export function EmptyState({ title, description, action }: { title: string; description: string; action?: ReactNode }) {
  return (
    <div className="px-4 py-8 md:px-6">
      <p className="font-medium">{title}</p>
      <p className="mt-1 max-w-[56ch] text-meta text-ink-soft">{description}</p>
      {action && <div className="mt-3">{action}</div>}
    </div>
  );
}
