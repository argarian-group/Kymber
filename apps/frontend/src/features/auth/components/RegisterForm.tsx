import { useAuth, type User } from "@/features/auth/AuthContext";
import { AuthScreen } from "@/features/auth/components/AuthScreen";
import { CreateAccountForm } from "@/features/auth/components/CreateAccountForm";
import { t } from "@/lib/i18n";

export function RegisterForm({ onLogIn }: { onLogIn: () => void }) {
  const { signIn } = useAuth();

  return (
    <AuthScreen title={t("Create your account")}>
      <CreateAccountForm<{ user: User }>
        endpoint="/auth/register"
        submitLabel={t("Create account")}
        onCreated={({ user }) => {
          window.history.replaceState(null, "", "/");
          signIn(user);
        }}
      />
      <p className="mt-4 text-meta text-ink-soft">
        {t("Already have an account?")}{" "}
        <button className="text-button" type="button" onClick={onLogIn}>
          {t("Log in")}
        </button>
      </p>
    </AuthScreen>
  );
}
