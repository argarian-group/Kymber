import type { ReactNode } from "react";
import { MobileTabBar } from "./MobileTabBar";
import { Sidebar } from "./Sidebar";

export function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-[248px_minmax(0,1fr)]">
      <Sidebar />
      <MobileTabBar />
      <main className="mx-auto flex w-full max-w-320 flex-col gap-4 p-4 pb-[calc(96px+env(safe-area-inset-bottom))] md:pb-4">
        {children}
      </main>
    </div>
  );
}
