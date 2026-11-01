import { serviceStatus } from "@/features/projects/services";

export function ServiceStatus({ status }: { status: string }) {
  const { label, icon: Icon, iconTone, labelTone } = serviceStatus(status);
  return (
    <span className={`flex flex-none items-center gap-1.5 text-meta ${labelTone}`}>
      <Icon className={`flex-none ${iconTone}`} aria-hidden="true" size={16} />
      {label}
    </span>
  );
}
