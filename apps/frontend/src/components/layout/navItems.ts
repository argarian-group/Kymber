import { FolderKanban, LayoutDashboard, Server, Settings, SquareTerminal, Users } from "lucide-react";

export const workspaceItems = [
  { label: "Overview", to: "/", icon: LayoutDashboard },
  { label: "Projects", to: "/projects", icon: FolderKanban },
];

export function organizationItems(isRoot: boolean) {
  return [
    { label: "Team", to: "/team", icon: Users },
    ...(isRoot
      ? [
          { label: "Servers", to: "/servers", icon: Server },
          { label: "Terminal", to: "/terminal", icon: SquareTerminal },
        ]
      : []),
    { label: "Settings", to: "/settings", icon: Settings },
  ];
}
