import { useEffect, useState } from "react";
import { api } from "@/lib/api";

export type Notification = { id: string; kind: "assigned" | "mentioned" | "commented"; actor: { id: string; username: string } | null; task_id: string; task_key: string; task_title: string; project_id: string; read_at: string | null; created_at: string };

export const notificationsChanged = "kymber:notifications-changed";

export function useUnreadNotifications() {
  const [unread, setUnread] = useState(0);

  useEffect(() => {
    let active = true;
    const load = () =>
      api<{ unread: number }>("/notifications")
        .then(({ unread }) => { if (active) setUnread(unread); })
        .catch(() => undefined);
    void load();
    const timer = window.setInterval(load, 30_000);
    window.addEventListener(notificationsChanged, load);
    return () => {
      active = false;
      window.clearInterval(timer);
      window.removeEventListener(notificationsChanged, load);
    };
  }, []);

  return unread;
}
