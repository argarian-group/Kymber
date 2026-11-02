import { useState } from "react";
import { useAuth, type User } from "@/features/auth/AuthContext";
import { AuthScreen } from "@/features/auth/components/AuthScreen";
import { CreateAccountForm } from "@/features/auth/components/CreateAccountForm";
import { PanelDomainForm } from "@/features/settings/components/PanelDomainForm";
import { t } from "@/lib/i18n";

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
      <AuthScreen title={t("Restart the server")} description={t("Secure onboarding keeps the API on localhost until it restarts. Restart it to accept other connections.")}>
        <button className="btn btn-primary" type="button" onClick={() => { setDomainFor(restartFor); setRestartFor(null); }}>
          {t("Continue to Kymber")}
        </button>
      </AuthScreen>
    );
  }

  if (domainFor) {
    return (
      <AuthScreen title={t("Add a domain for the panel?")} description={t("Optional. Point DNS to this server and Kymber will serve the dashboard over HTTPS. You can skip this and configure it later in System settings.")}>
        <PanelDomainForm onComplete={() => signIn(domainFor)} />
      </AuthScreen>
    );
  }

  return (
    <AuthScreen title={t("Set up Kymber")} description={t("Create the root account. It can manage every workspace on this server.")}>
      <CreateAccountForm<{ user: User; restart_required: boolean }>
        endpoint="/onboarding"
        submitLabel={t("Create root account")}
        onCreated={({ user, restart_required }) => (restart_required ? setRestartFor(user) : setDomainFor(user))}
      />
    </AuthScreen>
  );
}
