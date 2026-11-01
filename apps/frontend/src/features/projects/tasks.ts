import type { Priority } from "@/features/dashboard/data";

export type TaskStatus = "todo" | "in_progress" | "in_review" | "done";

export type Task = {
  id: string;
  project_id: string;
  title: string;
  status: TaskStatus;
  priority: Priority;
  assignee: { id: string; username: string } | null;
  due_date: string | null;
  created_at: string;
  updated_at: string;
};

export type TaskInput = {
  title: string;
  status: TaskStatus;
  priority: Priority;
  assignee_id: string | null;
  due_date: string | null;
};

export const taskStatuses: { id: TaskStatus; name: string }[] = [
  { id: "todo", name: "To do" },
  { id: "in_progress", name: "In progress" },
  { id: "in_review", name: "In review" },
  { id: "done", name: "Done" },
];

export function tasksPath(workspaceId: string, projectId: string) {
  return `/workspaces/${workspaceId}/projects/${projectId}/tasks`;
}

export function taskInput(task: Task): TaskInput {
  return { title: task.title, status: task.status, priority: task.priority, assignee_id: task.assignee?.id ?? null, due_date: task.due_date };
}

export function dueDay(task: Task) {
  if (!task.due_date) return null;
  const [year, month, day] = task.due_date.split("-").map(Number);
  return new Date(year, month - 1, day);
}

function daysUntil(day: Date) {
  const now = new Date();
  return Math.round((Date.UTC(day.getFullYear(), day.getMonth(), day.getDate()) - Date.UTC(now.getFullYear(), now.getMonth(), now.getDate())) / 86400000);
}

export function isLate(task: Task) {
  const day = dueDay(task);
  return day !== null && task.status !== "done" && daysUntil(day) < 0;
}

export function dueLabel(task: Task) {
  const day = dueDay(task);
  if (!day) return "";
  const days = daysUntil(day);
  if (task.status !== "done") {
    if (days === 0) return "Today";
    if (days === 1) return "Tomorrow";
    if (days === -1) return "1 day late";
    if (days < 0) return `${-days} days late`;
    if (days < 7) return `In ${days} days`;
  }
  return new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric" }).format(day);
}
