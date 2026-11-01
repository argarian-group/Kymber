import logo from "@assets/logo.svg";
import type { ReactNode } from "react";

export function AuthScreen({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  return (
    <main className="flex min-h-dvh items-center justify-center bg-sunken px-4 py-8">
      <div className="w-full max-w-sm">
        <img className="-ml-2 size-12" src={logo} alt="Kymber" width="48" height="48" />
        <h1 className="mt-4 font-display text-xl font-semibold tracking-[-0.01em]">{title}</h1>
        {description && <p className="mt-1 text-ink-soft">{description}</p>}
        <div className="mt-6">{children}</div>
      </div>
    </main>
  );
}

export function Field({ label, hint, ...input }: { label: string; hint?: string } & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <label className="block">
      <span className="mb-1 block text-meta font-medium">{label}</span>
      <input className="field" {...input} />
      {hint && <span className="mt-1 block text-xs text-ink-soft">{hint}</span>}
    </label>
  );
}

export function FormError({ message }: { message: string | null }) {
  if (!message) return null;
  return (
    <p className="text-meta text-ember" role="alert">
      {message}
    </p>
  );
}
