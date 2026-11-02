import type { User } from "@/features/auth/AuthContext";

export type Member = { user_id: string; username: string; email: string; role: string; created_at: string };

export type Acceptance = { user: User; workspace_id: string; role: string };

export type Role = { name: string; permissions: string[]; builtin: boolean };
