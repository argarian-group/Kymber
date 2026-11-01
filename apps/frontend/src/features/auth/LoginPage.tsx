import { useState, type FormEvent } from "react";
import { api, errorMessage } from "@/lib/api";
import { useAuth, type User } from "./AuthContext";
import { AuthScreen, Field, FormError } from "./AuthScreen";

export function LoginPage() {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError(null);
    try {
      const { user } = await api<{ user: User }>("/auth/login", { method: "POST", body: { email, password } });
      signIn(user);
    } catch (failure) {
      setError(errorMessage(failure));
      setPending(false);
    }
  }

  return (
    <AuthScreen title="Log in to Kymber" description="Use the email and password of your account.">
      <form className="grid gap-4" onSubmit={submit}>
        <Field label="Email" type="email" autoComplete="email" required autoFocus value={email} onChange={(event) => setEmail(event.target.value)} />
        <Field label="Password" type="password" autoComplete="current-password" required value={password} onChange={(event) => setPassword(event.target.value)} />
        <FormError message={error} />
        <button className="btn btn-primary justify-center" type="submit" disabled={pending}>
          {pending ? "Logging in…" : "Log in"}
        </button>
      </form>
    </AuthScreen>
  );
}
