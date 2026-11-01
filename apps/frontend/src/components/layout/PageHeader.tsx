import type { ReactNode } from "react";

export function PageHeader({ title, description, actions }: { title: string; description?: string; actions?: ReactNode }) {
  return (
    <header className="flex flex-wrap items-center gap-x-4 gap-y-3 px-4 pt-6 pb-1 md:px-6">
      <div>
        <h1 className="font-display text-xl font-semibold tracking-[-0.01em]">{title}</h1>
        {description && <p className="text-meta text-ink-soft">{description}</p>}
      </div>
      {actions && <div className="flex items-center gap-2 sm:ml-auto">{actions}</div>}
    </header>
  );
}
