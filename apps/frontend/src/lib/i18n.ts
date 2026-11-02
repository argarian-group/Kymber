import { ptBR } from "@/locales/pt-BR";

export const languages = [
  { id: "en", label: "English" },
  { id: "pt-BR", label: "Português (Brasil)" },
] as const;

export type Language = (typeof languages)[number]["id"];

export type TranslationKey = keyof typeof ptBR;

type Values = Record<string, string | number>;

const storageKey = "kymber.language";
const dictionaries: Record<Language, Record<string, string>> = { en: {}, "pt-BR": ptBR };

function readLanguage(): Language {
  let stored: string | null = null;
  try {
    stored = localStorage.getItem(storageKey);
  } catch {
    stored = null;
  }
  const preferred = stored ?? navigator.language;
  if (preferred === "pt" || preferred.startsWith("pt-")) return "pt-BR";
  return "en";
}

export const language: Language = readLanguage();

document.documentElement.lang = language;

// Text known only at runtime (API errors, roles, template labels); English text is the key and a missing one falls back to English.
export function translate(text: string, values?: Values) {
  const dictionary = dictionaries[language];
  const translated = Object.hasOwn(dictionary, text) ? dictionary[text] : text;
  if (!values) return translated;
  return translated.replace(/\{(\w+)\}/g, (match, key: string) => (key in values ? String(values[key]) : match));
}

// Fixed UI text: the key type makes the build fail when a translation is missing.
export function t(text: TranslationKey, values?: Values) {
  return translate(text, values);
}

// The language is read once at startup so module-level labels translate too; switching reloads the page.
export function setLanguage(next: Language) {
  try {
    localStorage.setItem(storageKey, next);
  } catch {
    // Storage is blocked, so the page falls back to the browser language after the reload.
  }
  window.location.reload();
}
