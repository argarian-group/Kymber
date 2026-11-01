import { CheckCircle2, CircleAlert, CircleDashed, CircleStop, LoaderCircle, type LucideIcon } from "lucide-react";

export type TemplateField = { name: string; label: string; type: string; required?: boolean; secret?: boolean; default?: string; generate?: boolean; min?: number; max?: number; pattern?: string };

export type DatabaseTemplate = { engine: string; image: string; default_version: string; port: number; data_path: string; fields: TemplateField[] };

export type Service = {
  id: string;
  engine: string;
  image: string;
  image_version: string;
  server_id?: string;
  server_name?: string;
  name: string;
  database_name: string;
  database_user: string;
  status: string;
  last_error?: string;
  internal_host?: string;
  port?: number;
  connection_url?: string;
  environment?: Record<string, string>;
};

export type ServiceAction = "deploy" | "start" | "stop" | "restart" | "remove_container";

const statuses: Record<string, { label: string; icon: LucideIcon; iconTone: string; labelTone: string }> = {
  draft: { label: "Draft", icon: CircleDashed, iconTone: "text-ink-faint", labelTone: "text-ink-soft" },
  queued: { label: "Queued", icon: LoaderCircle, iconTone: "text-seafoam animate-deploying motion-reduce:animate-none", labelTone: "font-medium text-ink" },
  deploying: { label: "Working", icon: LoaderCircle, iconTone: "text-seafoam animate-deploying motion-reduce:animate-none", labelTone: "font-medium text-ink" },
  running: { label: "Running", icon: CheckCircle2, iconTone: "text-ok", labelTone: "text-ink-soft" },
  stopped: { label: "Stopped", icon: CircleStop, iconTone: "text-ink-faint", labelTone: "text-ink-soft" },
  failed: { label: "Failed", icon: CircleAlert, iconTone: "text-ember", labelTone: "font-medium text-ember" },
};

export function serviceStatus(status: string) {
  return statuses[status] ?? { ...statuses.draft, label: status };
}

export function isServiceBusy(status: string) {
  return status === "queued" || status === "deploying";
}

export function servicesPath(workspaceId: string, projectId: string) {
  return `/workspaces/${workspaceId}/projects/${projectId}/services`;
}
