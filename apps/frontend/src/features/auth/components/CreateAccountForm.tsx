import { useState, type FormEvent } from "react";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import type { User } from "@/features/auth/AuthContext";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { t } from "@/lib/i18n";

type CreateAccountFormProps<T> = {
  endpoint: string;
  submitLabel: string;
  onCreated: (response: T) => void;
};

export function CreateAccountForm<T extends { user: User }>({ endpoint, submitLabel, onCreated }: CreateAccountFormProps<T>) {
  const [form, setForm] = useState({ username: "", email: "", password: "" });
  const [confirmPassword, setConfirmPassword] = useState("");
  const { pending, error, setError, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (confirmPassword !== form.password) {
      setError(t("The passwords do not match."));
      return;
    }
    await run(async () => onCreated(await api<T>(endpoint, { method: "POST", body: form })));
  }

  const update = (key: keyof typeof form) => (event: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: event.target.value });

  return (
    <form className="grid gap-4" onSubmit={submit}>
      <Field label={t("Username")} autoComplete="username" required autoFocus minLength={3} maxLength={32} hint={t("3 to 32 letters, numbers, _ or -.")} value={form.username} onChange={update("username")} />
      <Field label={t("Email")} type="email" autoComplete="email" required value={form.email} onChange={update("email")} />
      <Field label={t("Password")} type="password" autoComplete="new-password" required minLength={8} maxLength={128} hint={t("8 to 128 characters.")} value={form.password} onChange={update("password")} />
      <Field label={t("Confirm password")} type="password" autoComplete="new-password" required value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} />
      <FormError message={error} />
      <button className="btn btn-primary justify-center" type="submit" disabled={pending}>
        {pending ? t("Creating account…") : submitLabel}
      </button>
    </form>
  );
}
