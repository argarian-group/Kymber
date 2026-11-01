import logo from "@assets/logo.svg";
import type { LucideIcon } from "lucide-react";
import { Link, NavLink } from "react-router";
import { deploys, projects, stages, type Health } from "@/features/dashboard/data";
import { organizationItems, workspaceItems } from "./navItems";
import { SidebarProfile } from "./SidebarProfile";

const openTasks = stages.filter((stage) => stage.name !== "Done").reduce((sum, stage) => sum + stage.count, 0);
const runningDeploys = deploys.filter((deploy) => deploy.status === "running").length;

const hints: Record<string, string> = {
  "/projects": String(projects.length),
  "/tasks": String(openTasks),
  "/deploys": `${runningDeploys} running`,
};

const healthDots: Record<Health, { dot: string; label: string }> = {
  "on-track": { dot: "bg-seafoam", label: "On track" },
  "at-risk": { dot: "bg-amber", label: "At risk" },
  late: { dot: "bg-ember", label: "Late" },
};

const rowClass =
  "flex items-center gap-3 rounded-[10px] px-3 transition-colors duration-150 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-mist";

function NavGroup({ title, items }: { title: string; items: { label: string; to: string; icon: LucideIcon }[] }) {
  return (
    <div>
      <p className="mb-1 px-3 text-xs font-medium text-ink-faint">{title}</p>
      <ul className="grid gap-0.5">
        {items.map(({ label, to, icon: Icon }) => (
          <li key={to}>
            <NavLink
              className={({ isActive }) =>
                `${rowClass} min-h-10 ${isActive ? "bg-white/10 font-semibold text-ink" : "font-medium text-ink-soft hover:bg-white/5 hover:text-ink"}`
              }
              to={to}
              end
            >
              {({ isActive }) => (
                <>
                  <Icon className={isActive ? "text-seafoam" : ""} aria-hidden="true" size={18} />
                  <span>{label}</span>
                  {hints[to] && <span className="ml-auto text-xs font-medium text-ink-faint tabular-nums">{hints[to]}</span>}
                </>
              )}
            </NavLink>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function Sidebar() {
  return (
    <aside className="sticky top-0 hidden h-dvh flex-col self-start border-r border-hairline bg-deep-pine/80 md:flex">
      <div className="flex h-16 flex-none items-center justify-center gap-2 border-b border-hairline text-lg font-[650] tracking-[-0.02em]">
        <img className="-mx-[10.5px] -my-2 size-12 -translate-y-px" src={logo} alt="" width="48" height="48" />
        <span>Kymber</span>
      </div>

      <nav className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto p-3 pt-5 [scrollbar-color:var(--color-hairline)_transparent] [scrollbar-width:thin]" aria-label="Main">
        <NavGroup title="Workspace" items={workspaceItems} />

        <div>
          <p className="mb-1 px-3 text-xs font-medium text-ink-faint">Active projects</p>
          <ul className="grid gap-0.5">
            {projects.map((project) => (
              <li key={project.name}>
                <Link className={`${rowClass} min-h-9 text-meta text-ink-soft hover:bg-white/5 hover:text-ink`} to="/projects">
                  <span className={`size-2 flex-none rounded-full ${healthDots[project.health].dot}`} aria-hidden="true" />
                  <span className="min-w-0 truncate">{project.name}</span>
                  <span className="sr-only">, {healthDots[project.health].label}</span>
                </Link>
              </li>
            ))}
          </ul>
        </div>

        <NavGroup title="Organization" items={organizationItems} />
      </nav>

      <SidebarProfile name="Arthur Melo" role="Admin" />
    </aside>
  );
}
