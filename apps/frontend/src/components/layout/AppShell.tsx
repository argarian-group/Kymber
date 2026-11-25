import { Bell } from "lucide-react";
import { type CSSProperties, type ReactNode, useState } from "react";
import { Link } from "react-router";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { WorkspaceSwitcher } from "@/features/projects/components/WorkspaceSwitcher";
import { MobileTabBar } from "./MobileTabBar";
import { Sidebar } from "@/components/layout/Sidebar";
import { readSidebarWidth } from "@/components/layout/SidebarResizer";
import { LogoutButton } from "./LogoutButton";
import { TaskPalette } from "@/features/tasks/TaskPalette";
import { useUnreadNotifications } from "@/features/notifications/notifications";
import { t } from "@/lib/i18n";

export function AppShell({ children }: { children: ReactNode }) {
  const user = useCurrentUser();
  const [sidebarWidth, setSidebarWidth] = useState(readSidebarWidth);
  const unread = useUnreadNotifications();
  return (
    <div className="grid grid-cols-1 md:grid-cols-[var(--sidebar-width)_minmax(0,1fr)]" style={{ "--sidebar-width": `${sidebarWidth}px` } as CSSProperties}>
      <Sidebar width={sidebarWidth} unread={unread} onResize={setSidebarWidth} />
      <MobileTabBar />
      <main className="flex min-h-dvh min-w-0 flex-col pb-[calc(96px+env(safe-area-inset-bottom))] md:pb-0">
        <div className="bg-sunken px-4 py-2 md:hidden">
          <div className="flex items-center justify-between gap-3">
            <span className="min-w-0 truncate text-meta text-ink-soft">{user.username}</span>
            <div className="flex flex-none items-center gap-2">
              <Link className="btn relative px-2" to="/notifications" aria-label={unread > 0 ? t("Notifications, {count} unread", { count: unread }) : t("Notifications")}>
                <Bell aria-hidden="true" size={16} />
                {unread > 0 && <span className="absolute -top-1.5 -right-1.5 min-w-4 rounded-full bg-ember px-1 text-center text-[10px] leading-4 font-semibold text-white tabular-nums">{unread > 99 ? "99+" : unread}</span>}
              </Link>
              <LogoutButton />
            </div>
          </div>
          <div className="mt-2 empty:hidden">
            <WorkspaceSwitcher />
          </div>
        </div>
        {children}
      </main>
      <TaskPalette />
    </div>
  );
}
