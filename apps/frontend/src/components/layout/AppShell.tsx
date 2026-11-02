import { type CSSProperties, type ReactNode, useState } from "react";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { WorkspaceSwitcher } from "@/features/projects/components/WorkspaceSwitcher";
import { MobileTabBar } from "./MobileTabBar";
import { Sidebar } from "@/components/layout/Sidebar";
import { readSidebarWidth } from "@/components/layout/SidebarResizer";
import { LogoutButton } from "./LogoutButton";

export function AppShell({ children }: { children: ReactNode }) {
  const user = useCurrentUser();
  const [sidebarWidth, setSidebarWidth] = useState(readSidebarWidth);
  return (
    <div className="grid grid-cols-1 md:grid-cols-[var(--sidebar-width)_minmax(0,1fr)]" style={{ "--sidebar-width": `${sidebarWidth}px` } as CSSProperties}>
      <Sidebar width={sidebarWidth} onResize={setSidebarWidth} />
      <MobileTabBar />
      <main className="flex min-h-dvh min-w-0 flex-col pb-[calc(96px+env(safe-area-inset-bottom))] md:pb-0">
        <div className="bg-sunken px-4 py-2 md:hidden">
          <div className="flex items-center justify-between gap-3">
            <span className="min-w-0 truncate text-meta text-ink-soft">{user.username}</span>
            <div className="flex-none">
              <LogoutButton />
            </div>
          </div>
          <div className="mt-2 empty:hidden">
            <WorkspaceSwitcher />
          </div>
        </div>
        {children}
      </main>
    </div>
  );
}
