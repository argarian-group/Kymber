import { useState } from "react";
import { useAuth, type User } from "./AuthContext";
import { AuthScreen } from "./AuthScreen";
import { CreateAccountForm } from "./CreateAccountForm";

export function OnboardingPage() {
  const { signIn: enterApp } = useAuth();
  const [restartFor, setRestartFor] = useState<User | null>(null);
  const signIn = (user: User) => {
    window.history.replaceState(null, "", "/");
    enterApp(user);
  };

  if (restartFor) {
    return (
      <AuthScreen title="Restart the server" description="Secure onboarding keeps the API on localhost until it restarts. Restart it to accept other connections.">
        <button className="btn btn-primary" type="button" onClick={() => signIn(restartFor)}>
          Continue to Kymber
        </button>
      </AuthScreen>
    );
  }

  return (
    <AuthScreen title="Set up Kymber" description="Create the root account. It can manage every workspace on this server.">
      <CreateAccountForm<{ user: User; restart_required: boolean }>
        endpoint="/onboarding"
        submitLabel="Create root account"
        onCreated={({ user, restart_required }) => (restart_required ? setRestartFor(user) : signIn(user))}
      />
    </AuthScreen>
  );
}
