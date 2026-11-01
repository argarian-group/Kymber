import { useId, type ReactNode } from "react";

export function Panel({ title, className = "", children }: { title: string; className?: string; children: ReactNode }) {
  const titleId = useId();
  return (
    <section className={`rounded-panel border border-hairline bg-surface p-5 md:p-6 ${className}`} aria-labelledby={titleId}>
      <h2 id={titleId} className="mb-4 text-base font-semibold tracking-[-0.01em]">
        {title}
      </h2>
      {children}
    </section>
  );
}
