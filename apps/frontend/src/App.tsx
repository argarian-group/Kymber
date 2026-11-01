import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { AppShell } from "@/components/layout/AppShell";
import { PlaceholderPage } from "@/components/layout/PlaceholderPage";
import { DashboardPage } from "@/features/dashboard/DashboardPage";

export function App() {
  return (
    <BrowserRouter>
      <AppShell>
        <Routes>
          <Route index element={<DashboardPage />} />
          <Route path="projects" element={<PlaceholderPage title="Projects" description="Every project in one place." />} />
          <Route path="tasks" element={<PlaceholderPage title="Tasks" description="Plan the work and follow it through." />} />
          <Route path="deploys" element={<PlaceholderPage title="Deploys" description="What shipped, where and when." />} />
          <Route path="team" element={<PlaceholderPage title="Team" description="The people behind the work." />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AppShell>
    </BrowserRouter>
  );
}
