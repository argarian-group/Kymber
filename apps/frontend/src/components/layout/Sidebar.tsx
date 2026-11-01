import logo from "@assets/logo.svg";
import type { LucideIcon } from "lucide-react";
import { Link, NavLink } from "react-router";
import { useProjects } from "@/features/projects/ProjectsContext";
import { WorkspaceSwitcher } from "@/features/projects/WorkspaceSwitcher";
import { organizationItems, workspaceItems } from "./navItems";
import { SidebarProfile } from "./SidebarProfile";

const rowClass =
  "flex items-center gap-3 rounded-control px-2.5 transition-colors duration-150 focus-ring";

function NavGroup({ title, items, hints = {} }: { title: string; items: { label: string; to: string; icon: LucideIcon }[]; hints?: Record<string, string> }) {
  return (
    <div>
      <p className="mb-1 px-2.5 text-xs font-medium text-ink-faint">{title}</p>
      <ul className="grid gap-0.5">
        {items.map(({ label, to, icon: Icon }) => (
          <li key={to}>
            <NavLink
              className={({ isActive }) =>
                `${rowClass} min-h-8 ${isActive ? "bg-white/8 font-medium text-ink" : "font-medium text-ink-soft hover:bg-white/5 hover:text-ink"}`
              }
              to={to}
              end={to === "/"}
            >
              {({ isActive }) => (
                <>
                  <Icon className={isActive ? "text-seafoam" : ""} aria-hidden="true" size={16} />
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
  const { state } = useProjects();
  const projects = state.status === "ready" ? state.projects : [];

  return (
    <aside className="sticky top-0 hidden h-dvh flex-col self-start bg-sunken md:flex">
      <div className="flex h-14 flex-none items-center justify-center gap-2 font-display text-base font-semibold tracking-[-0.01em]">
        <img className="-mx-2 -my-2 size-10 -translate-y-px" src={logo} alt="" width="40" height="40" />
        <span>Kymber</span>
      </div>
      <div className="flex-none px-3 pb-2 [&>p]:px-2.5">
        <WorkspaceSwitcher />
      </div>

      <nav className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto p-3 pt-2 [scrollbar-color:var(--color-line)_transparent] [scrollbar-width:thin]" aria-label="Main">
        <NavGroup title="Workspace" items={workspaceItems} hints={projects.length > 0 ? { "/projects": String(projects.length) } : {}} />

        {projects.length > 0 && (
          <div>
            <p className="mb-1 px-2.5 text-xs font-medium text-ink-faint">Projects</p>
            <ul className="grid gap-0.5">
              {projects.map((project) => (
                <li key={project.id}>
                  <Link className={`${rowClass} min-h-8 text-meta text-ink-soft hover:bg-white/5 hover:text-ink`} to="/projects">
                    <span className="min-w-0 truncate">{project.name}</span>
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        )}

        <NavGroup title="Organization" items={organizationItems} />
      </nav>

      <SidebarProfile />
    </aside>
  );
}
