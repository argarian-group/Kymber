import { useState, type FormEvent } from "react";
import { api, errorMessage } from "@/lib/api";
import { useAuth, type User } from "./AuthContext";
import { AuthScreen, Field, FormError } from "./AuthScreen";
import { CreateAccountForm } from "./CreateAccountForm";

function RegisterForm({ onLogIn }: { onLogIn: () => void }) {
  const { signIn } = useAuth();

  return (
    <AuthScreen title="Create your account" description="You get a personal workspace to start with.">
      <CreateAccountForm<{ user: User }>
        endpoint="/auth/register"
        submitLabel="Create account"
        onCreated={({ user }) => {
          window.history.replaceState(null, "", "/");
          signIn(user);
        }}
      />
      <p className="mt-4 text-meta text-ink-soft">
        Already have an account?{" "}
        <button className="focus-ring cursor-pointer rounded-sm font-medium text-ink underline underline-offset-2" type="button" onClick={onLogIn}>
          Log in
        </button>
      </p>
    </AuthScreen>
  );
}

export function LoginPage() {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [registering, setRegistering] = useState(false);

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

  if (registering) return <RegisterForm onLogIn={() => setRegistering(false)} />;

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
      <p className="mt-4 text-meta text-ink-soft">
        New to Kymber?{" "}
        <button className="focus-ring cursor-pointer rounded-sm font-medium text-ink underline underline-offset-2" type="button" onClick={() => setRegistering(true)}>
          Create account
        </button>
      </p>
    </AuthScreen>
  );
}
