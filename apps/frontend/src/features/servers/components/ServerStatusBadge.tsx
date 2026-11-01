import type { ServerStatus } from "@/features/servers/types";

const styles: Record<ServerStatus, { label: string; dot: string }> = {
  online: { label: "Online", dot: "bg-ok" },
  offline: { label: "Offline", dot: "bg-ember" },
  pending: { label: "Waiting to connect", dot: "bg-amber motion-safe:animate-pulse" },
};

export function ServerStatusBadge({ status }: { status: ServerStatus }) {
  const { label, dot } = styles[status];
  return (
    <span className="flex flex-none items-center gap-2 text-meta font-medium">
      <span className={`size-2 rounded-full ${dot}`} aria-hidden="true" />
      {label}
    </span>
  );
}
