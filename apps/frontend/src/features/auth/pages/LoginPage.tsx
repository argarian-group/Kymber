import { type FormEvent, useState } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { useAuth, type User } from "@/features/auth/AuthContext";
import { AuthScreen } from "@/features/auth/components/AuthScreen";
import { RegisterForm } from "@/features/auth/components/RegisterForm";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

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
    <AuthScreen title={t("Log in to Kymber")}>
      <form className="grid gap-4" onSubmit={submit}>
        <Field label={t("Email")} type="email" autoComplete="email" required autoFocus value={email} onChange={(event) => setEmail(event.target.value)} />
        <Field label={t("Password")} type="password" autoComplete="current-password" required value={password} onChange={(event) => setPassword(event.target.value)} />
        <FormError message={error} />
        <button className="btn btn-primary justify-center" type="submit" disabled={pending}>
          {pending ? t("Logging in…") : t("Log in")}
        </button>
      </form>
      <p className="mt-4 text-meta text-ink-soft">
        {t("New to Kymber?")}{" "}
        <button className="text-button" type="button" onClick={() => setRegistering(true)}>
          {t("Create account")}
        </button>
      </p>
    </AuthScreen>
  );
}
