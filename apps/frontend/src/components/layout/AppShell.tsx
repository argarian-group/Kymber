import type { ReactNode } from "react";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { WorkspaceSwitcher } from "@/features/projects/WorkspaceSwitcher";
import { MobileTabBar } from "./MobileTabBar";
import { Sidebar } from "./Sidebar";
import { LogoutButton } from "./SidebarProfile";

export function AppShell({ children }: { children: ReactNode }) {
  const user = useCurrentUser();
  return (
    <div className="grid grid-cols-1 md:grid-cols-[224px_minmax(0,1fr)]">
      <Sidebar />
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
