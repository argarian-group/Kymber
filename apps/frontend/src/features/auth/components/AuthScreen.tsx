import logo from "@assets/logo.svg";
import type { ReactNode } from "react";

export function AuthScreen({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  return (
    <main className="flex min-h-dvh items-center justify-center bg-sunken px-4 py-8">
      <div className="w-full max-w-sm">
        <img className="-ml-2 size-12" src={logo} alt="Kymber" width="48" height="48" />
        <h1 className="mt-4 page-title">{title}</h1>
        {description && <p className="mt-1 text-ink-soft">{description}</p>}
        <div className="mt-6">{children}</div>
      </div>
    </main>
  );
}
