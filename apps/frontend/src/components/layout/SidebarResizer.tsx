import { t } from "@/lib/i18n";

export const sidebarWidth = { min: 200, max: 400, initial: 224 };

const storageKey = "kymber.sidebar-width";

const clamp = (value: number) => Math.round(Math.min(sidebarWidth.max, Math.max(sidebarWidth.min, value)));

export function readSidebarWidth() {
  try {
    const stored = Number(localStorage.getItem(storageKey));
    return stored ? clamp(stored) : sidebarWidth.initial;
  } catch {
    return sidebarWidth.initial;
  }
}

function save(width: number) {
  try {
    localStorage.setItem(storageKey, String(width));
  } catch {
    return;
  }
}

export function SidebarResizer({ width, onChange }: { width: number; onChange: (width: number) => void }) {
  function set(next: number) {
    const value = clamp(next);
    onChange(value);
    save(value);
  }

  return (
    <div
      className="group absolute inset-y-0 -right-1.5 z-20 w-3 cursor-col-resize touch-none focus-visible:outline-none"
      role="separator"
      aria-orientation="vertical"
      aria-label={t("Resize sidebar")}
      aria-valuemin={sidebarWidth.min}
      aria-valuemax={sidebarWidth.max}
      aria-valuenow={width}
      tabIndex={0}
      title={t("Drag to resize, double-click to reset")}
      onPointerDown={(event) => {
        event.preventDefault();
        event.currentTarget.setPointerCapture(event.pointerId);
      }}
      onPointerMove={(event) => {
        if (event.currentTarget.hasPointerCapture(event.pointerId)) onChange(clamp(event.clientX));
      }}
      onPointerUp={(event) => {
        event.currentTarget.releasePointerCapture(event.pointerId);
        set(event.clientX);
      }}
      onDoubleClick={() => set(sidebarWidth.initial)}
      onKeyDown={(event) => {
        const steps: Record<string, number> = { ArrowLeft: width - 16, ArrowRight: width + 16, Home: sidebarWidth.min, End: sidebarWidth.max };
        if (event.key in steps) {
          event.preventDefault();
          set(steps[event.key]);
        }
      }}
    >
      <span className="mx-auto block h-full w-0.5 bg-transparent transition-colors duration-150 group-hover:bg-seafoam/60 group-focus-visible:bg-seafoam group-active:bg-seafoam" />
    </div>
  );
}
