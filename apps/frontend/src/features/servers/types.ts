export type ServerStatus = "pending" | "online" | "offline";

export type Server = { id: string; name: string; connection_type: "local" | "ssh"; host?: string; ssh_port?: number; ssh_user?: string; ssh_configured: boolean; status: ServerStatus; last_seen: string | null; created_at: string; apps: number };
