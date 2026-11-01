import { type FormEvent, useState } from "react";
import { Field } from "@/components/ui/Field";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

export function PasswordForm() {
  const [form, setForm] = useState({ current: "", next: "", confirm: "" });
  const [saved, setSaved] = useState<string | null>(null);
  const { pending, error, setError, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    if (form.next !== form.confirm) {
      setError("The new passwords do not match.");
      return;
    }
    await run(async () => {
      await api("/account/password", { method: "PUT", body: { current_password: form.current, new_password: form.next } });
      setForm({ current: "", next: "", confirm: "" });
      setSaved("Password changed. Other devices were logged out.");
    });
  }

  const update = (key: keyof typeof form) => (event: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: event.target.value });

  return (
    <SettingsSection title="Password" description="Changing it logs you out on your other devices.">
      <form onSubmit={submit}>
        <div className="grid items-start gap-4 p-4 sm:grid-cols-2">
          <div className="sm:col-span-2 sm:max-w-[calc(50%-0.5rem)]">
            <Field label="Current password" type="password" autoComplete="current-password" required value={form.current} onChange={update("current")} />
          </div>
          <Field label="New password" type="password" autoComplete="new-password" required minLength={8} maxLength={128} hint="8 to 128 characters." value={form.next} onChange={update("next")} />
          <Field label="Confirm new password" type="password" autoComplete="new-password" required value={form.confirm} onChange={update("confirm")} />
        </div>
        <FormFooter error={error} saved={saved}>
          <button className="btn btn-primary" type="submit" disabled={pending}>
            {pending ? "Changing…" : "Change password"}
          </button>
        </FormFooter>
      </form>
    </SettingsSection>
  );
}
