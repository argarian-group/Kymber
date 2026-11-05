import { lazy, Suspense } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { AppShell } from "@/components/layout/AppShell";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { useAuth } from "@/features/auth/AuthContext";
import { AuthScreen } from "@/features/auth/components/AuthScreen";
import { LoginPage } from "@/features/auth/pages/LoginPage";
import { OnboardingPage } from "@/features/auth/pages/OnboardingPage";
import { ProjectsProvider } from "@/features/projects/ProjectsContext";
import { AppConnectionPage } from "@/features/projects/pages/app/AppConnectionPage";
import { AppDangerPage } from "@/features/projects/pages/app/AppDangerPage";
import { AppDeploymentsPage } from "@/features/projects/pages/app/AppDeploymentsPage";
import { AppEnvironmentPage } from "@/features/projects/pages/app/AppEnvironmentPage";
import { AppFilesPage } from "@/features/projects/pages/app/AppFilesPage";
import { AppGeneralPage } from "@/features/projects/pages/app/AppGeneralPage";
import { AppLayout } from "@/features/projects/pages/app/AppLayout";
import { AppLogsPage } from "@/features/projects/pages/app/AppLogsPage";
import { AppMetricsPage } from "@/features/projects/pages/app/AppMetricsPage";
import { AppStoragePage } from "@/features/projects/pages/app/AppStoragePage";
import { AppTerminalPage } from "@/features/projects/pages/app/AppTerminalPage";
import { ProjectNewAppPage } from "@/features/projects/pages/ProjectNewAppPage";
import { ProjectAppsPage } from "@/features/projects/pages/ProjectAppsPage";
import { ProjectLayout } from "@/features/projects/pages/ProjectLayout";
import { ProjectOverviewPage } from "@/features/projects/pages/ProjectOverviewPage";
import { ProjectSettingsPage } from "@/features/projects/pages/ProjectSettingsPage";
import { ProjectTasksPage } from "@/features/projects/pages/ProjectTasksPage";
import { ProjectsPage } from "@/features/projects/pages/ProjectsPage";
import { ServersPage } from "@/features/servers/pages/ServersPage";
import { SettingsPage } from "@/features/settings/SettingsPage";
import { InviteAcceptPage } from "@/features/workspaces/pages/InviteAcceptPage";
import { InviteSignupPage } from "@/features/workspaces/pages/InviteSignupPage";
import { TeamPage } from "@/features/workspaces/pages/TeamPage";
import { t } from "@/lib/i18n";

const TerminalPage = lazy(() => import("@/features/terminal/TerminalPage").then((module) => ({ default: module.TerminalPage })));
const DashboardPage = lazy(() => import("@/features/dashboard/DashboardPage").then((module) => ({ default: module.DashboardPage })));

export function AuthGate() {
  const { state, retry } = useAuth();

  switch (state.status) {
    case "loading":
      return <PageSkeleton />;
    case "error":
      return (
        <AuthScreen title={t("Kymber is unavailable")} description={state.message}>
          <button className="btn" type="button" onClick={retry}>
            {t("Try again")}
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
                    <Route path="apps/new" element={<ProjectNewAppPage />} />
                    <Route path="apps/:appId" element={<AppLayout />}>
                      <Route index element={<AppGeneralPage />} />
                      <Route path="environment" element={<AppEnvironmentPage />} />
                      <Route path="connection" element={<AppConnectionPage />} />
                      <Route path="files" element={<AppFilesPage />} />
                      <Route path="terminal" element={<AppTerminalPage />} />
                      <Route path="logs" element={<AppLogsPage />} />
                      <Route path="deployments" element={<AppDeploymentsPage />} />
                      <Route path="metrics" element={<AppMetricsPage />} />
                      <Route path="storage" element={<AppStoragePage />} />
                      <Route path="danger" element={<AppDangerPage />} />
                    </Route>
                    <Route path="deploys" element={<Navigate to="../apps" replace />} />
                    <Route path="tasks" element={<ProjectTasksPage />} />
                    <Route path="settings" element={<ProjectSettingsPage />} />
                  </Route>
                  <Route path="tasks" element={<Navigate to="/projects" replace />} />
                  <Route path="deploys" element={<Navigate to="/projects" replace />} />
                  <Route path="team" element={<TeamPage section="members" />} />
                  <Route path="team/roles" element={<TeamPage section="roles" />} />
                  <Route path="servers" element={<ServersPage />} />
                  <Route path="terminal" element={<TerminalPage />} />
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
