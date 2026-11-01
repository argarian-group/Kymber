import { useState, type FormEvent } from "react";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useAuth, type User } from "./AuthContext";
import { AuthScreen, Field, FormError } from "./AuthScreen";
import { CreateAccountForm } from "./CreateAccountForm";

function RegisterForm({ onLogIn }: { onLogIn: () => void }) {
  const { signIn } = useAuth();

  return (
    <AuthScreen title="Create your account">
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
  const { pending, error, run } = useAction();
  const [registering, setRegistering] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      const { user } = await api<{ user: User }>("/auth/login", { method: "POST", body: { email, password } });
      signIn(user);
    });
  }

  if (registering) return <RegisterForm onLogIn={() => setRegistering(false)} />;

  return (
    <AuthScreen title="Log in to Kymber">
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
