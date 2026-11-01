import { Check } from "lucide-react";
import { useState } from "react";
import { accents, defaultAppearance, densities, readAppearance, saveAppearance, themes, type Appearance } from "./appearance";
import { SettingsSection } from "./SettingsSection";

function Preview() {
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
          <span className="font-display text-base font-semibold">Overview</span>
          <span className="rounded-control bg-seafoam px-2 py-1 text-xs font-semibold text-on-accent">New project</span>
        </div>
        <div className="mt-3 grid gap-1 rounded-panel bg-raised p-3 text-meta">
          <span className="flex justify-between gap-3">
            <span>Customer portal</span>
            <span className="text-ink-soft">82%</span>
          </span>
          <span className="flex justify-between gap-3">
            <span>Company website</span>
            <span className="text-ink-soft">95%</span>
          </span>
        </div>
      </div>
    </div>
  );
}

export function AppearanceSettings() {
  const [appearance, setAppearance] = useState(readAppearance);

  function update(next: Partial<Appearance>) {
    const merged = { ...appearance, ...next };
    setAppearance(merged);
    saveAppearance(merged);
  }

  const isDefault = appearance.theme === defaultAppearance.theme && appearance.accent === defaultAppearance.accent && appearance.density === defaultAppearance.density;

  return (
    <>
      <SettingsSection title="Preview" description="Changes apply right away and stay saved in this browser.">
        <div className="p-4">
          <Preview />
          <div className="mt-3 flex justify-end">
            <button className="btn" type="button" disabled={isDefault} onClick={() => update(defaultAppearance)}>
              Reset to default
            </button>
          </div>
        </div>
      </SettingsSection>

      <SettingsSection title="Theme" description="The colors behind everything.">
        <fieldset className="grid gap-3 p-4 sm:grid-cols-3">
          <legend className="sr-only">Theme</legend>
          {themes.map((theme) => {
            const selected = appearance.theme === theme.id;
            return (
              <label className="group cursor-pointer" key={theme.id}>
                <input className="peer sr-only" type="radio" name="theme" checked={selected} onChange={() => update({ theme: theme.id })} />
                <span
                  className={`block overflow-hidden rounded-panel border-2 transition-colors duration-100 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-seafoam ${selected ? "border-seafoam" : "border-line-strong group-hover:border-ink-faint"}`}
                >
                  <span className="grid h-16 grid-cols-[1fr_3fr]" aria-hidden="true">
                    <span style={{ background: theme.sunken }} />
                    <span className="p-2" style={{ background: theme.canvas }}>
                      <span className="block h-full rounded-control" style={{ background: theme.raised }} />
                    </span>
                  </span>
                </span>
                <span className="mt-1.5 flex items-center gap-1.5 text-meta font-medium">
                  {theme.label}
                  {selected && <Check className="text-seafoam" aria-hidden="true" size={14} />}
                </span>
              </label>
            );
          })}
        </fieldset>
      </SettingsSection>

      <SettingsSection title="Accent color" description="Used for selection, focus, your avatar and main buttons.">
        <fieldset className="flex flex-wrap gap-x-5 gap-y-3 p-4">
          <legend className="sr-only">Accent color</legend>
          {accents.map((accent) => {
            const selected = appearance.accent === accent.id;
            return (
              <label className="flex w-14 cursor-pointer flex-col items-center gap-1.5 text-xs font-medium" key={accent.id}>
                <input className="peer sr-only" type="radio" name="accent" checked={selected} onChange={() => update({ accent: accent.id })} />
                <span
                  className={`flex size-9 items-center justify-center rounded-full text-on-accent ring-2 ring-offset-2 ring-offset-raised transition-shadow duration-100 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-4 peer-focus-visible:outline-ink ${selected ? "ring-ink" : "ring-transparent"}`}
                  style={{ background: accent.color }}
                >
                  {selected && <Check aria-hidden="true" size={16} />}
                </span>
                <span className={selected ? "text-ink" : "text-ink-soft"}>{accent.label}</span>
              </label>
            );
          })}
        </fieldset>
      </SettingsSection>

      <SettingsSection title="Density" description="How much space the interface uses.">
        <fieldset className="grid gap-2 p-4 sm:grid-cols-2">
          <legend className="sr-only">Density</legend>
          {densities.map((density) => {
            const selected = appearance.density === density.id;
            return (
              <label className="cursor-pointer" key={density.id}>
                <input className="peer sr-only" type="radio" name="density" checked={selected} onChange={() => update({ density: density.id })} />
                <span
                  className={`block rounded-control border-2 px-3 py-2 transition-colors duration-100 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-seafoam ${selected ? "border-seafoam" : "border-line-strong hover:border-ink-faint"}`}
                >
                  <span className="block font-medium">{density.label}</span>
                  <span className="block text-meta text-ink-soft">{density.description}</span>
                </span>
              </label>
            );
          })}
        </fieldset>
      </SettingsSection>
    </>
  );
}
