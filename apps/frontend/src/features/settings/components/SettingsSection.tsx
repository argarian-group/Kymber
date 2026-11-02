import { type ReactNode, useId } from "react";

export function SettingsSection({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  const titleId = useId();
  return (
    <section className="grid gap-x-10 gap-y-3 px-4 py-6 md:px-6 lg:grid-cols-[220px_minmax(0,1fr)]" aria-labelledby={titleId}>
      <div>
        <h2 id={titleId} className="section-title">
          {title}
        </h2>
        {description && <p className="mt-1 text-meta text-ink-soft">{description}</p>}
      </div>
      <div className="min-w-0 rounded-panel bg-raised">{children}</div>
    </section>
  );
}
