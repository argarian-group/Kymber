import { type FormEvent, useState } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { useAuth } from "@/features/auth/AuthContext";
import { AuthScreen } from "@/features/auth/components/AuthScreen";
import { LoginPage } from "@/features/auth/pages/LoginPage";
import { rememberWorkspace } from "@/features/projects/ProjectsContext";
import { type Acceptance } from "@/features/workspaces/types";
import { api, ApiError } from "@/lib/api";
import { useAction } from "@/lib/useAction";

export function InviteSignupPage() {
  const { signIn } = useAuth();
  const token = window.location.hash.slice(1);
  const [form, setForm] = useState({ username: "", password: "" });
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loggingIn, setLoggingIn] = useState(false);
  const { pending, error, setError, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (confirmPassword !== form.password) {
      setError("The passwords do not match.");
      return;
    }
    await run(async () => {
      try {
        const { user, workspace_id } = await api<Acceptance>("/invitations/accept", { method: "POST", body: { token, ...form } });
        rememberWorkspace(user.id, workspace_id);
        window.history.replaceState(null, "", "/projects");
        signIn(user);
      } catch (failure) {
        if (!(failure instanceof ApiError && failure.status === 401)) throw failure;
        setLoggingIn(true);
      }
    });
  }

  if (loggingIn) return <LoginPage />;

  const update = (key: keyof typeof form) => (event: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: event.target.value });

  return (
    <AuthScreen title="Join a workspace" description="Create your account to accept this invitation.">
      <form className="grid gap-4" onSubmit={submit}>
        <Field label="Username" autoComplete="username" required autoFocus minLength={3} maxLength={32} hint="3 to 32 letters, numbers, _ or -." value={form.username} onChange={update("username")} />
        <Field label="Password" type="password" autoComplete="new-password" required minLength={8} maxLength={128} hint="8 to 128 characters." value={form.password} onChange={update("password")} />
        <Field label="Confirm password" type="password" autoComplete="new-password" required value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} />
        <FormError message={error} />
        <button className="btn btn-primary justify-center" type="submit" disabled={pending}>
          {pending ? "Accepting…" : "Create account and join"}
        </button>
      </form>
      <p className="mt-4 text-meta text-ink-soft">
        Already have an account?{" "}
        <button className="text-button" type="button" onClick={() => setLoggingIn(true)}>
          Log in
        </button>
      </p>
    </AuthScreen>
  );
}
