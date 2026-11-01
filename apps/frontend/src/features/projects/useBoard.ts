import { useState } from "react";
import { api, errorMessage } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { boardPath, moveInBoard, type Board } from "./tasks";
import { useProjectOutlet } from "./useProjectOutlet";

export function useBoard() {
  const { project, workspace } = useProjectOutlet();
  const path = boardPath(workspace.id, project.id);
  const [board, setBoard] = useState<Board | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const reload = () => setAttempt((value) => value + 1);

  useLoadEffect(
    () => api<Board>(`${path}/board`),
    (response) => {
      setBoard(response);
      setError(null);
    },
    setError,
    [path, attempt],
  );

  async function persist(action: () => Promise<unknown>) {
    try {
      await action();
    } catch (failure) {
      setError(errorMessage(failure));
      const fresh = await api<Board>(`${path}/board`).catch(() => null);
      if (fresh) setBoard(fresh);
      return;
    }
    reload();
  }

  async function createTask(listId: string, title: string) {
    await api(`${path}/tasks`, { method: "POST", body: { list_id: listId, title } });
    reload();
  }

  function moveTask(taskId: string, listId: string, position: number) {
    setError(null);
    setBoard((current) => current && moveInBoard(current, taskId, listId, position));
    return persist(() => api(`${path}/tasks/${taskId}/move`, { method: "POST", body: { list_id: listId, position } }));
  }

  function reorderLists(ids: string[]) {
    setError(null);
    setBoard((current) => current && { ...current, lists: current.lists.map((list) => ({ ...list, position: ids.indexOf(list.id) + 1 })) });
    return persist(() => api(`${path}/task-lists/order`, { method: "PUT", body: { ids } }));
  }

  async function createList(name: string) {
    await api(`${path}/task-lists`, { method: "POST", body: { name, is_done: false } });
    reload();
  }

  async function updateList(id: string, input: { name: string; is_done: boolean }) {
    await api(`${path}/task-lists/${id}`, { method: "PUT", body: input });
    reload();
  }

  async function deleteList(id: string) {
    await api(`${path}/task-lists/${id}`, { method: "DELETE" });
    reload();
  }

  return { board, error, path, reload, createTask, moveTask, reorderLists, createList, updateList, deleteList };
}
