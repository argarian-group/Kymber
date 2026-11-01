import { lazy, Suspense } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { AppShell } from "@/components/layout/AppShell";
import { PlaceholderPage } from "@/components/layout/PlaceholderPage";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { AuthProvider, useAuth } from "@/features/auth/AuthContext";
import { AuthScreen } from "@/features/auth/AuthScreen";
import { LoginPage } from "@/features/auth/LoginPage";
import { OnboardingPage } from "@/features/auth/OnboardingPage";
import { InviteAcceptPage, InviteSignupPage } from "@/features/workspaces/InvitePages";
import { ProjectsProvider } from "@/features/projects/ProjectsContext";
import { ProjectsPage } from "@/features/projects/ProjectsPage";
import { SettingsPage } from "@/features/settings/SettingsPage";
import { TeamPage } from "@/features/workspaces/TeamPage";

const DashboardPage = lazy(() => import("@/features/dashboard/DashboardPage").then((module) => ({ default: module.DashboardPage })));

function AuthGate() {
  const { state, retry } = useAuth();

  switch (state.status) {
    case "loading":
      return <PageSkeleton />;
    case "error":
      return (
        <AuthScreen title="Kymber is unavailable" description={state.message}>
          <button className="btn" type="button" onClick={retry}>
            Try again
          </button>
        </AuthScreen>
      );
    case "onboarding":
      return <OnboardingPage />;
    case "anonymous":
      return window.location.pathname === "/invite" ? <InviteSignupPage /> : <LoginPage />;
    case "authenticated":
      return (
        <ProjectsProvider key={state.user.id}>
          <BrowserRouter>
            <AppShell>
              <Suspense fallback={<PageSkeleton />}>
                <Routes>
                  <Route index element={<DashboardPage />} />
                  <Route path="projects" element={<ProjectsPage />} />
                  <Route path="tasks" element={<PlaceholderPage title="Tasks" />} />
                  <Route path="deploys" element={<PlaceholderPage title="Deploys" />} />
                  <Route path="team" element={<TeamPage />} />
                  <Route path="invite" element={<InviteAcceptPage />} />
                  <Route path="settings" element={<SettingsPage section="account" />} />
                  <Route path="settings/appearance" element={<SettingsPage section="appearance" />} />
                  <Route path="settings/workspace" element={<SettingsPage section="workspace" />} />
                  <Route path="settings/system" element={<SettingsPage section="system" />} />
                  <Route path="*" element={<Navigate to="/" replace />} />
                </Routes>
              </Suspense>
            </AppShell>
          </BrowserRouter>
        </ProjectsProvider>
      );
  }
}

export function App() {
  return (
    <AuthProvider>
      <AuthGate />
    </AuthProvider>
  );
}
