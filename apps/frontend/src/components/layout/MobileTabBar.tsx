import { NavLink } from "react-router";
import { navItems } from "./navItems";

export function MobileTabBar() {
  return (
    <nav
      className="fixed inset-x-2 bottom-[calc(8px+env(safe-area-inset-bottom))] z-10 rounded-panel border border-line-strong bg-raised p-1 md:hidden"
      aria-label="Main"
    >
      <ul className="grid grid-cols-5">
        {navItems.map(({ label, to, icon: Icon }) => (
          <li key={to}>
            <NavLink
              className={({ isActive }) =>
                `flex min-h-14 w-full flex-col items-center justify-center gap-0.5 rounded-control px-0.5 py-1 text-tab transition-[background-color,color,scale] duration-300 ease-spring focus-ring active:scale-96 ${
                  isActive
                    ? "bg-white/10 font-semibold text-ink"
                    : "font-medium text-ink-soft"
                }`
              }
              to={to}
              end
            >
              <Icon aria-hidden="true" size={20} />
              <span>{label}</span>
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  );
}
