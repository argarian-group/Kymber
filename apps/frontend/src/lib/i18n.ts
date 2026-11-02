import { ptBR } from "@/locales/pt-BR";

export const languages = [
  { id: "en", label: "English" },
  { id: "pt-BR", label: "Português (Brasil)" },
] as const;

export type Language = (typeof languages)[number]["id"];

const storageKey = "kymber.language";
const dictionaries: Record<Language, Record<string, string>> = { en: {}, "pt-BR": ptBR };

function readLanguage(): Language {
  let stored: string | null = null;
  try {
    stored = localStorage.getItem(storageKey);
  } catch {
    stored = null;
  }
  const preferred = stored ?? (typeof navigator === "undefined" ? "en" : navigator.language);
  if (preferred === "pt" || preferred.startsWith("pt-")) return "pt-BR";
  return "en";
}

export const language: Language = readLanguage();

if (typeof document !== "undefined") document.documentElement.lang = language;

// English text is the key, so a missing translation falls back to English.
export function t(text: string, values?: Record<string, string | number>) {
  const dictionary = dictionaries[language];
  const translated = Object.hasOwn(dictionary, text) ? dictionary[text] : text;
  if (!values) return translated;
  return translated.replace(/\{(\w+)\}/g, (match, key: string) => (key in values ? String(values[key]) : match));
}

// The language is read once at startup so module-level labels translate too; switching reloads the page.
export function setLanguage(next: Language) {
  try {
    localStorage.setItem(storageKey, next);
  } catch {
    return;
  }
  window.location.reload();
}
