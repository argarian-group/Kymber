import { FolderKanban, LayoutDashboard, Server, Settings, SquareTerminal, Users } from "lucide-react";
import { t } from "@/lib/i18n";

export const workspaceItems = [
  { label: t("Overview"), to: "/", icon: LayoutDashboard },
  { label: t("Projects"), to: "/projects", icon: FolderKanban },
];

export function organizationItems(isRoot: boolean) {
  return [
    { label: t("Team"), to: "/team", icon: Users },
    ...(isRoot
      ? [
          { label: t("Servers"), to: "/servers", icon: Server },
          { label: t("Terminal"), to: "/terminal", icon: SquareTerminal },
        ]
      : []),
    { label: t("Settings"), to: "/settings", icon: Settings },
  ];
}
