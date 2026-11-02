import { type LucideIcon } from "lucide-react";
import { NavLink } from "react-router";

export const rowClass =
  "flex items-center gap-2.5 rounded-control px-2.5 transition-colors duration-150 focus-ring";

export function NavGroup({ title, items, hints = {}, compact = false }: { title: string; items: { label: string; to: string; icon: LucideIcon }[]; hints?: Record<string, string>; compact?: boolean }) {
  return (
    <div>
      <p className={compact ? "sr-only" : "mb-1 px-2.5 text-xs font-medium text-ink-faint"}>{title}</p>
      <ul className="grid gap-0.5">
        {items.map(({ label, to, icon: Icon }) => (
          <li key={to}>
            <NavLink
              className={({ isActive }) =>
                `${rowClass} min-h-8 ${compact ? "justify-center px-0" : ""} ${isActive ? "bg-white/8 font-medium text-ink" : "font-medium text-ink-soft hover:bg-white/5 hover:text-ink"}`
              }
              to={to}
              end={to === "/"}
              title={compact ? label : undefined}
            >
              {({ isActive }) => (
                <>
                  <Icon className={isActive ? "text-seafoam" : ""} aria-hidden="true" size={16} />
                  <span className={compact ? "sr-only" : "min-w-0 truncate"}>{label}</span>
                  {hints[to] && !compact && <span className="ml-auto text-xs font-medium text-ink-faint tabular-nums">{hints[to]}</span>}
                </>
              )}
            </NavLink>
          </li>
        ))}
      </ul>
    </div>
  );
}
