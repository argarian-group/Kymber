import { lazy, Suspense } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { AppShell } from "@/components/layout/AppShell";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { useAuth } from "@/features/auth/AuthContext";
import { AuthScreen } from "@/features/auth/components/AuthScreen";
import { LoginPage } from "@/features/auth/pages/LoginPage";
import { OnboardingPage } from "@/features/auth/pages/OnboardingPage";
import { ProjectsProvider } from "@/features/projects/ProjectsContext";
import { ProjectAppPage } from "@/features/projects/pages/ProjectAppPage";
import { ProjectAppsPage } from "@/features/projects/pages/ProjectAppsPage";
import { ProjectLayout } from "@/features/projects/pages/ProjectLayout";
import { ProjectOverviewPage } from "@/features/projects/pages/ProjectOverviewPage";
import { ProjectSettingsPage } from "@/features/projects/pages/ProjectSettingsPage";
import { ProjectTasksPage } from "@/features/projects/pages/ProjectTasksPage";
import { ProjectsPage } from "@/features/projects/pages/ProjectsPage";
import { SettingsPage } from "@/features/settings/SettingsPage";
import { InviteAcceptPage } from "@/features/workspaces/pages/InviteAcceptPage";
import { InviteSignupPage } from "@/features/workspaces/pages/InviteSignupPage";
import { TeamPage } from "@/features/workspaces/pages/TeamPage";

const DashboardPage = lazy(() => import("@/features/dashboard/DashboardPage").then((module) => ({ default: module.DashboardPage })));

export function AuthGate() {
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
                  <Route path="projects/:projectId" element={<ProjectLayout />}>
                    <Route index element={<ProjectOverviewPage />} />
                    <Route path="apps" element={<ProjectAppsPage />} />
                    <Route path="apps/:appId" element={<ProjectAppPage />} />
                    <Route path="deploys" element={<Navigate to="../apps" replace />} />
                    <Route path="tasks" element={<ProjectTasksPage />} />
                    <Route path="settings" element={<ProjectSettingsPage />} />
                  </Route>
                  <Route path="tasks" element={<Navigate to="/projects" replace />} />
                  <Route path="deploys" element={<Navigate to="/projects" replace />} />
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
