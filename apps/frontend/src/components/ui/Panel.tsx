import { useId, type ReactNode } from "react";

type PanelProps = { title: string; aside?: ReactNode; className?: string; children: ReactNode };

export function Panel({ title, aside, className = "", children }: PanelProps) {
  const titleId = useId();
  return (
    <section className={`py-5 ${className}`} aria-labelledby={titleId}>
      <header className="mb-2 flex min-h-8 flex-wrap items-center justify-between gap-x-3 gap-y-1 px-4 md:px-6">
        <h2 id={titleId} className="section-title">
          {title}
        </h2>
        {aside}
      </header>
      {children}
    </section>
  );
}
