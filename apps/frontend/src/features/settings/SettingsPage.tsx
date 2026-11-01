import { Navigate, NavLink } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { AccountSettings } from "./components/AccountSettings";
import { AppearanceSettings } from "./components/AppearanceSettings";
import { SystemSettings } from "./components/SystemSettings";
import { WorkspaceSettings } from "./components/WorkspaceSettings";

const descriptions: Partial<Record<string, string>> = {
  account: "Your profile and password.",
  workspace: "Name and lifecycle of your workspaces.",
  system: "Options for everyone on this server.",
};

export function SettingsPage({ section }: { section: "account" | "appearance" | "workspace" | "system" }) {
  const user = useCurrentUser();
  if (section === "system" && !user.is_root) return <Navigate to="/settings" replace />;
  const tabs = [
    { label: "Account", to: "/settings" },
    { label: "Appearance", to: "/settings/appearance" },
    { label: "Workspace", to: "/settings/workspace" },
    ...(user.is_root ? [{ label: "System", to: "/settings/system" }] : []),
  ];

  return (
    <>
      <PageHeader title="Settings" description={descriptions[section]} />
      <nav className="mt-4 flex gap-5 overflow-x-auto border-b border-line px-4 md:px-6" aria-label="Settings sections">
        {tabs.map(({ label, to }) => (
          <NavLink
            className={({ isActive }) =>
              `focus-ring flex min-h-10 flex-none items-center border-b-2 text-meta font-medium transition-colors duration-100 ${isActive ? "border-seafoam text-ink" : "border-transparent text-ink-soft hover:text-ink"}`
            }
            to={to}
            end
            key={to}
          >
            {label}
          </NavLink>
        ))}
      </nav>
      {section === "account" && <AccountSettings />}
      {section === "appearance" && <AppearanceSettings />}
      {section === "workspace" && <WorkspaceSettings />}
      {section === "system" && <SystemSettings />}
    </>
  );
}
