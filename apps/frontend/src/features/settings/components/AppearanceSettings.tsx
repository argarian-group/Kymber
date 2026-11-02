import { Check } from "lucide-react";
import { useState } from "react";
import { accents, type Appearance, defaultAppearance, densities, readAppearance, saveAppearance, themes } from "@/features/settings/appearance";
import { AppearancePreview } from "@/features/settings/components/AppearancePreview";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { language, languages, setLanguage, t } from "@/lib/i18n";

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
      <SettingsSection title={t("Preview")}>
        <div className="p-4">
          <AppearancePreview />
          <div className="mt-3 flex justify-end">
            <button className="btn" type="button" disabled={isDefault} onClick={() => update(defaultAppearance)}>
              {t("Reset to default")}
            </button>
          </div>
        </div>
      </SettingsSection>

      <SettingsSection title={t("Language")} description={t("Changing it reloads the page.")}>
        <div className="p-4">
          <select className="field w-auto cursor-pointer" aria-label={t("Language")} value={language} onChange={(event) => setLanguage(event.target.value as typeof language)}>
            {languages.map(({ id, label }) => (
              <option value={id} key={id}>
                {label}
              </option>
            ))}
          </select>
        </div>
      </SettingsSection>

      <SettingsSection title={t("Theme")}>
        <fieldset className="grid gap-3 p-4 sm:grid-cols-3">
          <legend className="sr-only">{t("Theme")}</legend>
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

      <SettingsSection title={t("Accent color")}>
        <fieldset className="flex flex-wrap gap-x-5 gap-y-3 p-4">
          <legend className="sr-only">{t("Accent color")}</legend>
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

      <SettingsSection title={t("Density")}>
        <fieldset className="grid gap-2 p-4 sm:grid-cols-2">
          <legend className="sr-only">{t("Density")}</legend>
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
