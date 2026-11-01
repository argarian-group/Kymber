import { Link } from "react-router";
import { Panel } from "@/components/ui/Panel";
import { type Deploy } from "@/features/dashboard/data";
import { deployStates } from "@/features/dashboard/deployStates";

export function DeploysPanel({ deploys }: { deploys: Deploy[] }) {
  return (
    <Panel title="Recent deploys">
      <ol>
        {deploys.map((deploy) => {
          const { label, icon: Icon, iconTone, labelTone } = deployStates[deploy.status];
          return (
            <li key={`${deploy.project}-${deploy.version}`}>
              <Link className="row-link grid grid-cols-[16px_minmax(0,1fr)_auto] items-center gap-x-3 px-4 py-2 md:px-6" to="/projects">
                <Icon className={iconTone} aria-hidden="true" size={16} />
                <span className="min-w-0">
                  <span className="block truncate font-medium">{deploy.project}</span>
                  <span className="block truncate text-meta text-ink-soft tabular-nums">
                    {deploy.version} to {deploy.environment}
                  </span>
                </span>
                <span className="text-right text-meta">
                  <span className={`block ${labelTone}`}>{label}</span>
                  <span className="block text-ink-faint tabular-nums">{deploy.when}</span>
                </span>
              </Link>
            </li>
          );
        })}
      </ol>
    </Panel>
  );
}
