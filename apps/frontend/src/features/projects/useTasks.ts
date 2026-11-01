import { useState } from "react";
import { api, errorMessage } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { taskInput, tasksPath, type Task, type TaskInput, type TaskStatus } from "./tasks";
import { useProjectOutlet } from "./useProjectOutlet";

export function useTasks() {
  const { project, workspace } = useProjectOutlet();
  const path = tasksPath(workspace.id, project.id);
  const [tasks, setTasks] = useState<Task[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const reload = () => setAttempt((value) => value + 1);

  useLoadEffect(
    () => api<{ tasks: Task[] }>(path),
    (response) => {
      setTasks(response.tasks);
      setError(null);
    },
    setError,
    [path, attempt],
  );

  async function createTask(input: TaskInput) {
    const { task } = await api<{ task: Task }>(path, { method: "POST", body: input });
    setTasks((current) => [...(current ?? []), task]);
  }

  async function updateTask(id: string, input: TaskInput) {
    const { task } = await api<{ task: Task }>(`${path}/${id}`, { method: "PUT", body: input });
    setTasks((current) => current?.map((item) => (item.id === id ? task : item)) ?? null);
  }

  async function deleteTask(id: string) {
    await api(`${path}/${id}`, { method: "DELETE" });
    setTasks((current) => current?.filter((item) => item.id !== id) ?? null);
  }

  async function moveTask(task: Task, status: TaskStatus) {
    if (task.status === status) return;
    setError(null);
    setTasks((current) => current?.map((item) => (item.id === task.id ? { ...item, status } : item)) ?? null);
    try {
      await updateTask(task.id, { ...taskInput(task), status });
    } catch (failure) {
      setTasks((current) => current?.map((item) => (item.id === task.id ? task : item)) ?? null);
      setError(errorMessage(failure));
    }
  }

  return { tasks, error, reload, createTask, updateTask, deleteTask, moveTask };
}
