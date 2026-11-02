import logo from "@assets/logo.svg";
import { Link, NavLink } from "react-router";
import { NavGroup, rowClass } from "@/components/layout/NavGroup";
import { SidebarProfile } from "@/components/layout/SidebarProfile";
import { SidebarResizer, sidebarWidth } from "@/components/layout/SidebarResizer";
import { organizationItems, workspaceItems } from "@/components/layout/navItems";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { useProjects } from "@/features/projects/ProjectsContext";
import { ProjectMark } from "@/features/projects/components/ProjectMark";
import { WorkspaceSwitcher } from "@/features/projects/components/WorkspaceSwitcher";
import { t } from "@/lib/i18n";

export function Sidebar({ width, onResize }: { width: number; onResize: (width: number) => void }) {
  const user = useCurrentUser();
  const { state } = useProjects();
  const projects = state.status === "ready" ? state.projects : [];
  const compact = width === sidebarWidth.compact;

  return (
    <aside className="sticky top-0 hidden h-dvh flex-col self-start bg-sunken md:flex">
      <SidebarResizer width={width} onChange={onResize} />
      <div className="flex h-14 flex-none items-center justify-center">
        <Link className="focus-ring flex items-center gap-2 rounded-sm font-display text-base font-semibold tracking-[-0.01em]" to="/" aria-label={t("Kymber overview")}>
          <img className={compact ? "size-9 flex-none" : "-mx-2 -my-2 size-10 -translate-y-px"} src={logo} alt="" width="40" height="40" />
          {!compact && <span>Kymber</span>}
        </Link>
      </div>
      {!compact && (
        <div className="flex-none px-3 pb-2 [&>p]:px-2.5">
          <WorkspaceSwitcher />
        </div>
      )}

      <nav className={`flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto ${compact ? "px-2" : "px-3"} pt-2 pb-3 [scrollbar-color:var(--color-line)_transparent] [scrollbar-width:thin]`} aria-label={t("Main")}>
        <NavGroup title={t("Workspace")} items={workspaceItems} hints={projects.length > 0 ? { "/projects": String(projects.length) } : {}} compact={compact} />

        {projects.length > 0 && (
          <div>
            <p className={compact ? "sr-only" : "mb-1 px-2.5 text-xs font-medium text-ink-faint"}>{t("Projects")}</p>
            <ul className="grid grid-cols-1 gap-0.5">
              {projects.map((project) => (
                <li key={project.id}>
                  <NavLink
                    className={({ isActive }) => `${rowClass} min-h-8 ${compact ? "justify-center px-0" : ""} text-meta hover:bg-white/5 hover:text-ink ${isActive ? "bg-white/8 text-ink" : "text-ink-soft"}`}
                    to={`/projects/${project.id}`}
                    title={compact ? project.name : undefined}
                  >
                    <ProjectMark project={project} size="xs" />
                    <span className={compact ? "sr-only" : "min-w-0 truncate"}>{project.name}</span>
                  </NavLink>
                </li>
              ))}
            </ul>
          </div>
        )}

        <NavGroup title={t("Organization")} items={organizationItems(user.is_root)} compact={compact} />
      </nav>

      <SidebarProfile compact={compact} />
    </aside>
  );
}
