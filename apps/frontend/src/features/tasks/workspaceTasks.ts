import { type Priority } from "@/features/dashboard/priorities";

export type WorkspaceTask = { id: string; key: string; title: string; priority: Priority; due_date: string | null; project_id: string; project_name: string; list_name: string; done: boolean };

export const taskLink = (task: WorkspaceTask) => `/projects/${task.project_id}/tasks?task=${task.id}`;
