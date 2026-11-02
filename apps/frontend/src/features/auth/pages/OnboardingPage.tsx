import { useState } from "react";
import { useAuth, type User } from "@/features/auth/AuthContext";
import { AuthScreen } from "@/features/auth/components/AuthScreen";
import { CreateAccountForm } from "@/features/auth/components/CreateAccountForm";
import { PanelDomainForm } from "@/features/settings/components/PanelDomainForm";

export function OnboardingPage() {
  const { signIn: enterApp } = useAuth();
  const [restartFor, setRestartFor] = useState<User | null>(null);
  const [domainFor, setDomainFor] = useState<User | null>(null);
  const signIn = (user: User) => {
    window.history.replaceState(null, "", "/");
    enterApp(user);
  };

  if (restartFor) {
    return (
      <AuthScreen title="Restart the server" description="Secure onboarding keeps the API on localhost until it restarts. Restart it to accept other connections.">
        <button className="btn btn-primary" type="button" onClick={() => { setDomainFor(restartFor); setRestartFor(null); }}>
          Continue to Kymber
        </button>
      </AuthScreen>
    );
  }

  if (domainFor) {
    return (
      <AuthScreen title="Add a domain for the panel?" description="Optional. Point DNS to this server and Kymber will serve the dashboard over HTTPS. You can skip this and configure it later in System settings.">
        <PanelDomainForm onComplete={() => signIn(domainFor)} />
      </AuthScreen>
    );
  }

  return (
    <AuthScreen title="Set up Kymber" description="Create the root account. It can manage every workspace on this server.">
      <CreateAccountForm<{ user: User; restart_required: boolean }>
        endpoint="/onboarding"
        submitLabel="Create root account"
        onCreated={({ user, restart_required }) => (restart_required ? setRestartFor(user) : setDomainFor(user))}
      />
    </AuthScreen>
  );
}
