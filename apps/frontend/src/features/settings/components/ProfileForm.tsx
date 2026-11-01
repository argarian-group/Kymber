import { type FormEvent, useState } from "react";
import { Field } from "@/components/ui/Field";
import { useAuth, useCurrentUser, type User } from "@/features/auth/AuthContext";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

export function ProfileForm() {
  const user = useCurrentUser();
  const { signIn } = useAuth();
  const [username, setUsername] = useState(user.username);
  const [email, setEmail] = useState(user.email);
  const [currentPassword, setCurrentPassword] = useState("");
  const [saved, setSaved] = useState<string | null>(null);
  const { pending, error, run } = useAction();

  const changed = username !== user.username || email !== user.email;

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!changed) return;
    setSaved(null);
    await run(async () => {
      const response = await api<{ user: User }>("/account/profile", {
        method: "PUT",
        body: { username, email, current_password: currentPassword },
      });
      signIn(response.user);
      setUsername(response.user.username);
      setEmail(response.user.email);
      setCurrentPassword("");
      setSaved("Profile saved.");
    });
  }

  return (
    <SettingsSection title="Profile" description="How you appear in Kymber and the email you log in with.">
      <form onSubmit={submit}>
        <div className="grid items-start gap-4 p-4 sm:grid-cols-2">
          <Field label="Username" autoComplete="username" required minLength={3} maxLength={32} hint="3 to 32 letters, numbers, _ or -." value={username} onChange={(event) => setUsername(event.target.value)} />
          <Field label="Email" type="email" autoComplete="email" required value={email} onChange={(event) => setEmail(event.target.value)} />
          {changed && (
            <Field label="Confirm with your password" type="password" autoComplete="current-password" required hint="Needed to save these changes." value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} />
          )}
        </div>
        <FormFooter error={error} saved={saved}>
          <button className="btn btn-primary" type="submit" disabled={pending || !changed}>
            {pending ? "Saving…" : "Save profile"}
          </button>
        </FormFooter>
      </form>
    </SettingsSection>
  );
}
