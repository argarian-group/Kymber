import { language, t } from "@/lib/i18n";
const dateFormat = new Intl.DateTimeFormat(language, { dateStyle: "medium" });

export function formatDate(value: string) {
  return dateFormat.format(new Date(value));
}

export function formatRelative(value: string) {
  const seconds = Math.max(0, Math.round((Date.now() - new Date(value).getTime()) / 1000));
  if (seconds < 60) return t("{n}s ago", { n: seconds });
  if (seconds < 3600) return t("{n}m ago", { n: Math.floor(seconds / 60) });
  if (seconds < 86400) return t("{n}h ago", { n: Math.floor(seconds / 3600) });
  return t("{n}d ago", { n: Math.floor(seconds / 86400) });
}
