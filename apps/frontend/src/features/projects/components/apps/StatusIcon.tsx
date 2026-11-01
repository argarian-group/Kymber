import { deployStates } from "@/features/dashboard/deployStates";
import { type SampleDeployStatus } from "@/features/projects/sampleData";

export function StatusIcon({ status }: { status: SampleDeployStatus }) {
  const { icon: Icon, iconTone } = deployStates[status];
  return <Icon className={`flex-none ${iconTone}`} aria-hidden="true" size={16} />;
}
