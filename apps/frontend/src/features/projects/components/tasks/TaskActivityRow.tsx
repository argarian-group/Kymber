import { priorities, type Priority } from "@/features/dashboard/priorities";
import { type TaskActivity } from "@/features/projects/tasks";
import { formatRelative } from "@/lib/format";
import { language, t, type TranslationKey } from "@/lib/i18n";

const dayFormat = new Intl.DateTimeFormat(language, { dateStyle: "medium" });

function day(value: string) {
  const [year, month, date] = value.split("-").map(Number);
  return dayFormat.format(new Date(year, month - 1, date));
}

function describe({ kind, payload }: TaskActivity) {
  const from = String(payload.from ?? "");
  const to = String(payload.to ?? "");
  const names = (key: "added" | "removed") => ((payload[key] as string[] | undefined) ?? []).join(", ");
  const changes = (added: TranslationKey, removed: TranslationKey) => [names("added") && t(added, { names: names("added") }), names("removed") && t(removed, { names: names("removed") })].filter(Boolean).join("; ");
  switch (kind) {
    case "created": return t("created this task");
    case "title": return t("renamed it to “{to}”", { to });
    case "priority": return t("changed the priority to {to}", { to: priorities[to as Priority]?.label ?? to });
    case "due_date": return to ? t("set the due date to {date}", { date: day(to) }) : t("removed the due date");
    case "start_date": return to ? t("set the start date to {date}", { date: day(to) }) : t("removed the start date");
    case "description": return t("edited the description");
    case "assignees": return changes("assigned {names}", "unassigned {names}");
    case "labels": return changes("added the label {names}", "removed the label {names}");
    case "moved": return t("moved it from {from} to {to}", { from, to });
    case "archived": return t("archived this task");
    case "restored": return t("restored this task");
    default: return kind;
  }
}

export function TaskActivityRow({ activity }: { activity: TaskActivity }) {
  return (
    <li className="text-meta text-ink-soft">
      <span className="font-medium text-ink">{activity.actor?.username ?? t("Deleted user")}</span> {describe(activity)}
      <span className="text-ink-faint"> · {formatRelative(activity.created_at)}</span>
    </li>
  );
}
