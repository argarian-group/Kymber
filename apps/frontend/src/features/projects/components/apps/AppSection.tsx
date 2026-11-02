import type { ReactNode } from "react";

export function AppSection({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  return (
    <section className="grid gap-3">
      <div>
        <h3 className="section-title">{title}</h3>
        {description && <p className="mt-0.5 text-meta text-ink-soft">{description}</p>}
      </div>
      <div className="rounded-panel bg-raised">{children}</div>
    </section>
  );
}
