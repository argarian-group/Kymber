import { Navigate, NavLink } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { AccountSettings } from "./components/AccountSettings";
import { AppearanceSettings } from "./components/AppearanceSettings";
import { SystemSettings } from "./components/SystemSettings";
import { WorkspaceSettings } from "./components/WorkspaceSettings";
import { t } from "@/lib/i18n";

const descriptions: Partial<Record<string, string>> = {
  account: t("Your profile and password."),
  workspace: t("Name and lifecycle of your workspaces."),
  system: t("Options for everyone on this server."),
  users: t("Manage accounts on this server."),
};

export function SettingsPage({ section }: { section: "account" | "appearance" | "workspace" | "system" | "users" }) {
  const user = useCurrentUser();
  if ((section === "system" || section === "users") && !user.is_root) return <Navigate to="/settings" replace />;
  const tabs = [
    { label: t("Account"), to: "/settings" },
    { label: t("Appearance"), to: "/settings/appearance" },
    { label: t("Workspace"), to: "/settings/workspace" },
    ...(user.is_root ? [{ label: t("System"), to: "/settings/system" }, { label: t("Users"), to: "/settings/users" }] : []),
  ];

  return (
    <>
      <PageHeader title={t("Settings")} description={descriptions[section]} />
      <nav className="mt-4 flex gap-5 overflow-x-auto border-b border-line px-4 md:px-6" aria-label={t("Settings sections")}>
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
      {section === "users" && <SystemSettings usersOnly />}
      <p className="px-4 pt-2 pb-8 text-xs text-ink-soft md:px-6">
        {t("Kymber is free software under the AGPL-3.0 license.")}{" "}
        <a className="text-button" href="https://github.com/argarian-group/Kymber" target="_blank" rel="noreferrer">
          {t("Source code")}
        </a>
      </p>
    </>
  );
}
