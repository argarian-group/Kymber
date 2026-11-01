import { type LucideIcon } from "lucide-react";
import { NavLink } from "react-router";

export const rowClass =
  "flex items-center gap-3 rounded-control px-2.5 transition-colors duration-150 focus-ring";

export function NavGroup({ title, items, hints = {} }: { title: string; items: { label: string; to: string; icon: LucideIcon }[]; hints?: Record<string, string> }) {
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
