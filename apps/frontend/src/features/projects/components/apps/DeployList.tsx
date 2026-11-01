import { deployStates } from "@/features/dashboard/deployStates";
import { StatusIcon } from "@/features/projects/components/apps/StatusIcon";
import { sampleApps, type SampleDeploy } from "@/features/projects/sampleData";

export function DeployList({ deploys }: { deploys: SampleDeploy[] }) {
  return (
    <ol className="border-t border-line">
      {deploys.map((deploy) => {
        const app = sampleApps.find(({ id }) => id === deploy.appId);
        return (
          <li className="grid grid-cols-[16px_minmax(0,1fr)_auto] items-center gap-x-3 border-b border-line py-3" key={deploy.id}>
            <StatusIcon status={deploy.status} />
            <div className="min-w-0">
              <p className="truncate font-medium">{deploy.message}</p>
              <p className="truncate text-meta text-ink-soft tabular-nums">
                {app ? `${app.name} on ${app.node}, ` : ""}
                {deploy.version} by {deploy.author}, took {deploy.duration}
              </p>
            </div>
            <div className="text-right text-meta">
              <p className={deployStates[deploy.status].labelTone}>{deployStates[deploy.status].label}</p>
              <p className="text-ink-faint tabular-nums">{deploy.when}</p>
            </div>
          </li>
        );
      })}
    </ol>
  );
}
