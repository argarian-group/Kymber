import { FolderKanban, LayoutDashboard, Settings, Users } from "lucide-react";

export const workspaceItems = [
  { label: "Overview", to: "/", icon: LayoutDashboard },
  { label: "Projects", to: "/projects", icon: FolderKanban },
];

export const organizationItems = [
  { label: "Team", to: "/team", icon: Users },
  { label: "Settings", to: "/settings", icon: Settings },
];

export const navItems = [...workspaceItems, ...organizationItems];
