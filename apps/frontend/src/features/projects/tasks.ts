import type { Priority } from "@/features/dashboard/data";

export type TaskPerson = { id: string; username: string };

export type Task = {
  id: string;
  project_id: string;
  list_id: string;
  position: number;
  title: string;
  has_description: boolean;
  priority: Priority;
  due_date: string | null;
  assignees: TaskPerson[];
  label_ids: string[];
  checklist_done: number;
  checklist_total: number;
  comment_count: number;
  created_at: string;
  updated_at: string;
};

export type TaskListColumn = { id: string; name: string; is_done: boolean; position: number };

export type TaskLabel = { id: string; name: string; color: LabelColor };

export type Board = { lists: TaskListColumn[]; labels: TaskLabel[]; tasks: Task[] };

export type ChecklistItem = { id: string; title: string; done: boolean };

export type Checklist = { id: string; title: string; items: ChecklistItem[] };

export type TaskComment = { id: string; author: TaskPerson | null; body: string; created_at: string; updated_at: string };

export type TaskDetail = Task & { description: string; checklists: Checklist[]; comments: TaskComment[] };

export type TaskFilters = { search: string; assignee: string; label: string; priority: string; due: string };

export const labelColors = {
  red: "#ff9d8c",
  orange: "#ffb870",
  yellow: "#ffd27a",
  green: "#8fdc8a",
  teal: "#47cbb7",
  blue: "#8ab8ff",
  purple: "#c7a6ff",
  gray: "#b4d0c8",
};

export type LabelColor = keyof typeof labelColors;

export function boardPath(workspaceId: string, projectId: string) {
  return `/workspaces/${workspaceId}/projects/${projectId}`;
}

export function tasksInList(tasks: Task[], listId: string) {
  return tasks.filter(({ list_id }) => list_id === listId).sort((a, b) => a.position - b.position || a.id.localeCompare(b.id));
}

export function moveInBoard(board: Board, taskId: string, listId: string, position: number): Board {
  const moved = board.tasks.find(({ id }) => id === taskId);
  if (!moved) return board;
  const others = tasksInList(board.tasks, listId).filter(({ id }) => id !== taskId);
  const ordered = [...others.slice(0, position), { ...moved, list_id: listId }, ...others.slice(position)];
  const positions = new Map(ordered.map((task, index) => [task.id, index + 1]));
  return {
    ...board,
    tasks: board.tasks.map((task) => (task.id === taskId ? { ...task, list_id: listId, position: positions.get(taskId) ?? 1 } : positions.has(task.id) ? { ...task, position: positions.get(task.id) ?? task.position } : task)),
  };
}

export function positionBefore(board: Board, taskId: string, listId: string, beforeId: string | null) {
  const others = tasksInList(board.tasks, listId).filter(({ id }) => id !== taskId);
  const index = others.findIndex(({ id }) => id === beforeId);
  return index === -1 ? others.length : index;
}

export function dueDay(task: Pick<Task, "due_date">) {
  if (!task.due_date) return null;
  const [year, month, day] = task.due_date.split("-").map(Number);
  return new Date(year, month - 1, day);
}

function daysUntil(day: Date) {
  const now = new Date();
  return Math.round((Date.UTC(day.getFullYear(), day.getMonth(), day.getDate()) - Date.UTC(now.getFullYear(), now.getMonth(), now.getDate())) / 86400000);
}

export function isLate(task: Task, done: boolean) {
  const day = dueDay(task);
  return day !== null && !done && daysUntil(day) < 0;
}

export function dueLabel(task: Task, done: boolean) {
  const day = dueDay(task);
  if (!day) return "";
  const days = daysUntil(day);
  if (!done) {
    if (days === 0) return "Today";
    if (days === 1) return "Tomorrow";
    if (days === -1) return "1 day late";
    if (days < 0) return `${-days} days late`;
    if (days < 7) return `In ${days} days`;
  }
  return new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric" }).format(day);
}

export function matchesFilters(task: Task, filters: TaskFilters, done: boolean) {
  if (filters.search && !task.title.toLocaleLowerCase().includes(filters.search.trim().toLocaleLowerCase())) return false;
  if (filters.assignee === "none" && task.assignees.length > 0) return false;
  if (filters.assignee && filters.assignee !== "none" && !task.assignees.some(({ id }) => id === filters.assignee)) return false;
  if (filters.label && !task.label_ids.includes(filters.label)) return false;
  if (filters.priority && task.priority !== filters.priority) return false;
  const day = dueDay(task);
  if (filters.due === "none") return day === null;
  if (filters.due === "late") return isLate(task, done);
  if (filters.due === "week") return day !== null && !done && daysUntil(day) >= 0 && daysUntil(day) < 7;
  return true;
}
