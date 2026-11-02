import { t } from "@/lib/i18n";
export function AppearancePreview() {
  return (
    <div className="grid grid-cols-[88px_minmax(0,1fr)] overflow-hidden rounded-panel border border-line" aria-hidden="true">
      <div className="grid content-start gap-1.5 bg-sunken p-2">
        <span className="h-2 w-10 rounded-xs bg-seafoam" />
        <span className="mt-2 h-5 rounded-control bg-white/8" />
        <span className="h-5 rounded-control" />
        <span className="h-5 rounded-control" />
      </div>
      <div className="min-w-0 bg-canvas p-3">
        <div className="flex items-center justify-between gap-3">
          <span className="font-display text-base font-semibold">{t("Overview")}</span>
          <span className="rounded-control bg-seafoam px-2 py-1 text-xs font-semibold text-on-accent">{t("New project")}</span>
        </div>
        <div className="mt-3 grid gap-2.5 rounded-panel bg-raised p-3">
          <span className="flex items-center justify-between gap-3">
            <span className="h-2 w-3/5 rounded-xs bg-ink/70" />
            <span className="h-2 w-6 rounded-xs bg-ink-soft/50" />
          </span>
          <span className="flex items-center justify-between gap-3">
            <span className="h-2 w-2/5 rounded-xs bg-ink/70" />
            <span className="h-2 w-6 rounded-xs bg-ink-soft/50" />
          </span>
        </div>
      </div>
    </div>
  );
}
