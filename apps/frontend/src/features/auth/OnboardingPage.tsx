import { useState, type FormEvent } from "react";
import { api, errorMessage } from "@/lib/api";
import { useAuth, type User } from "./AuthContext";
import { AuthScreen, Field, FormError } from "./AuthScreen";

export function OnboardingPage() {
  const { signIn: enterApp } = useAuth();
  const signIn = (user: User) => {
    window.history.replaceState(null, "", "/");
    enterApp(user);
  };
  const [form, setForm] = useState({ username: "", email: "", password: "" });
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [restartFor, setRestartFor] = useState<User | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError(null);
    try {
      const { user, restart_required } = await api<{ user: User; restart_required: boolean }>("/onboarding", { method: "POST", body: form });
      if (restart_required) setRestartFor(user);
      else signIn(user);
    } catch (failure) {
      setError(errorMessage(failure));
      setPending(false);
    }
  }

  if (restartFor) {
    return (
      <AuthScreen title="Restart the server" description="Secure onboarding keeps the API on localhost until it restarts. Restart it to accept other connections.">
        <button className="btn btn-primary" type="button" onClick={() => signIn(restartFor)}>
          Continue to Kymber
        </button>
      </AuthScreen>
    );
  }

  const update = (key: keyof typeof form) => (event: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: event.target.value });

  return (
    <AuthScreen title="Set up Kymber" description="Create the root account. It can manage every workspace on this server.">
      <form className="grid gap-4" onSubmit={submit}>
        <Field label="Username" autoComplete="username" required autoFocus minLength={3} maxLength={32} hint="3 to 32 letters, numbers, _ or -." value={form.username} onChange={update("username")} />
        <Field label="Email" type="email" autoComplete="email" required value={form.email} onChange={update("email")} />
        <Field label="Password" type="password" autoComplete="new-password" required value={form.password} onChange={update("password")} />
        <FormError message={error} />
        <button className="btn btn-primary justify-center" type="submit" disabled={pending}>
          {pending ? "Creating account…" : "Create root account"}
        </button>
      </form>
    </AuthScreen>
  );
}
