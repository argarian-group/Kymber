import { FolderKanban, LayoutDashboard, Server, Settings, Users } from "lucide-react";

export const workspaceItems = [
  { label: "Overview", to: "/", icon: LayoutDashboard },
  { label: "Projects", to: "/projects", icon: FolderKanban },
];

export function organizationItems(isRoot: boolean) {
  return [
    { label: "Team", to: "/team", icon: Users },
    ...(isRoot ? [{ label: "Servers", to: "/servers", icon: Server }] : []),
    { label: "Settings", to: "/settings", icon: Settings },
  ];
}
