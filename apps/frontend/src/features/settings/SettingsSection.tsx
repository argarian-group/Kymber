import { useId, type ReactNode } from "react";
import { FormError } from "@/features/auth/AuthScreen";

export function SettingsSection({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  const titleId = useId();
  return (
    <section className="grid gap-x-10 gap-y-3 px-4 py-6 md:px-6 lg:grid-cols-[220px_minmax(0,1fr)]" aria-labelledby={titleId}>
      <div>
        <h2 id={titleId} className="font-display text-base font-semibold">
          {title}
        </h2>
        <p className="mt-1 text-meta text-ink-soft">{description}</p>
      </div>
      <div className="max-w-2xl min-w-0 rounded-panel bg-raised">{children}</div>
    </section>
  );
}

export function FormFooter({ error, saved, children }: { error: string | null; saved: string | null; children: ReactNode }) {
  return (
    <div className="flex flex-wrap items-center justify-end gap-x-4 gap-y-2 border-t border-line px-4 py-3">
      <div className="mr-auto min-w-0">
        <FormError message={error} />
        {!error && <Saved message={saved} />}
      </div>
      {children}
    </div>
  );
}

export function Saved({ message }: { message: string | null }) {
  return (
    <p className="min-h-5 text-meta text-ink-soft" role="status">
      {message}
    </p>
  );
}
