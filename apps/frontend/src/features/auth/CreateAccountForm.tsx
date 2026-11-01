import { useState, type FormEvent } from "react";
import { api, errorMessage } from "@/lib/api";
import type { User } from "./AuthContext";
import { Field, FormError } from "./AuthScreen";

type CreateAccountFormProps<T> = {
  endpoint: string;
  submitLabel: string;
  onCreated: (response: T) => void;
};

export function CreateAccountForm<T extends { user: User }>({ endpoint, submitLabel, onCreated }: CreateAccountFormProps<T>) {
  const [form, setForm] = useState({ username: "", email: "", password: "" });
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError(null);
    try {
      onCreated(await api<T>(endpoint, { method: "POST", body: form }));
    } catch (failure) {
      setError(errorMessage(failure));
      setPending(false);
    }
  }

  const update = (key: keyof typeof form) => (event: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: event.target.value });

  return (
    <form className="grid gap-4" onSubmit={submit}>
      <Field label="Username" autoComplete="username" required autoFocus minLength={3} maxLength={32} hint="3 to 32 letters, numbers, _ or -." value={form.username} onChange={update("username")} />
      <Field label="Email" type="email" autoComplete="email" required value={form.email} onChange={update("email")} />
      <Field label="Password" type="password" autoComplete="new-password" required minLength={8} maxLength={128} hint="8 to 128 characters." value={form.password} onChange={update("password")} />
      <FormError message={error} />
      <button className="btn btn-primary justify-center" type="submit" disabled={pending}>
        {pending ? "Creating account…" : submitLabel}
      </button>
    </form>
  );
}
