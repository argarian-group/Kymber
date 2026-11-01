import { deployStates } from "@/features/dashboard/deployStates";
import { StatusIcon } from "@/features/projects/components/apps/StatusIcon";
import { type SampleDeployStatus } from "@/features/projects/sampleData";

export function AppStatus({ status }: { status: SampleDeployStatus }) {
  const { label, labelTone } = deployStates[status];
  return (
    <span className={`flex flex-none items-center gap-1.5 text-meta ${labelTone}`}>
      <StatusIcon status={status} />
      {label}
    </span>
  );
}
