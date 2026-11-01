import { useAuth, type User } from "@/features/auth/AuthContext";
import { AuthScreen } from "@/features/auth/components/AuthScreen";
import { CreateAccountForm } from "@/features/auth/components/CreateAccountForm";

export function RegisterForm({ onLogIn }: { onLogIn: () => void }) {
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
        <button className="text-button" type="button" onClick={onLogIn}>
          Log in
        </button>
      </p>
    </AuthScreen>
  );
}
