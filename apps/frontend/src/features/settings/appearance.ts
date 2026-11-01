export const themes = [
  { id: "pine", label: "Pine", sunken: "#062621", canvas: "#0a302a", raised: "#114239" },
  { id: "midnight", label: "Midnight", sunken: "#0b1424", canvas: "#111c30", raised: "#1a2842" },
  { id: "graphite", label: "Graphite", sunken: "#131517", canvas: "#1a1d20", raised: "#25292d" },
] as const;

export const accents = [
  { id: "seafoam", label: "Seafoam", color: "#47cbb7" },
  { id: "sky", label: "Sky", color: "#5cc4f2" },
  { id: "violet", label: "Violet", color: "#b49cf7" },
  { id: "amber", label: "Amber", color: "#f2c14e" },
  { id: "rose", label: "Rose", color: "#f28fb0" },
] as const;

export const densities = [
  { id: "comfortable", label: "Comfortable", description: "More room between rows." },
  { id: "compact", label: "Compact", description: "More content on screen." },
] as const;

export type Appearance = {
  theme: (typeof themes)[number]["id"];
  accent: (typeof accents)[number]["id"];
  density: (typeof densities)[number]["id"];
};

export const defaultAppearance: Appearance = { theme: "pine", accent: "seafoam", density: "comfortable" };

const storageKey = "kymber.appearance";

export function readAppearance(): Appearance {
  try {
    const stored = JSON.parse(localStorage.getItem(storageKey) ?? "{}");
    return {
      theme: themes.find(({ id }) => id === stored?.theme)?.id ?? defaultAppearance.theme,
      accent: accents.find(({ id }) => id === stored?.accent)?.id ?? defaultAppearance.accent,
      density: densities.find(({ id }) => id === stored?.density)?.id ?? defaultAppearance.density,
    };
  } catch {
    return defaultAppearance;
  }
}

export function applyAppearance(appearance: Appearance) {
  const root = document.documentElement;
  root.dataset.theme = appearance.theme;
  root.dataset.accent = appearance.accent;
  root.dataset.density = appearance.density;
  const canvas = themes.find(({ id }) => id === appearance.theme)?.canvas;
  if (canvas) document.querySelector('meta[name="theme-color"]')?.setAttribute("content", canvas);
}

export function saveAppearance(appearance: Appearance) {
  applyAppearance(appearance);
  try {
    localStorage.setItem(storageKey, JSON.stringify(appearance));
  } catch {
    return;
  }
}
