import { FolderKanban, LayoutDashboard, ListChecks, Rocket, Users } from "lucide-react";

export const workspaceItems = [
  { label: "Overview", to: "/", icon: LayoutDashboard },
  { label: "Projects", to: "/projects", icon: FolderKanban },
  { label: "Tasks", to: "/tasks", icon: ListChecks },
  { label: "Deploys", to: "/deploys", icon: Rocket },
];

export const organizationItems = [{ label: "Team", to: "/team", icon: Users }];

export const navItems = [...workspaceItems, ...organizationItems];
