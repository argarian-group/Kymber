export type ServerStatus = "pending" | "online" | "offline";

export type Server = { id: string; name: string; status: ServerStatus; last_seen: string | null; created_at: string; apps: number };
