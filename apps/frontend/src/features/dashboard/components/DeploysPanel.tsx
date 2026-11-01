import { CheckCircle2, CircleAlert, LoaderCircle, type LucideIcon } from "lucide-react";
import { Panel } from "@/components/ui/Panel";
import type { Deploy, DeployStatus } from "../data";

const deployStates: Record<DeployStatus, { label: string; icon: LucideIcon }> = {
  live: { label: "Live", icon: CheckCircle2 },
  running: { label: "Deploying", icon: LoaderCircle },
  failed: { label: "Failed", icon: CircleAlert },
};

export function DeploysPanel({ deploys, className }: { deploys: Deploy[]; className?: string }) {
  return (
    <Panel title="Recent deploys" className={className}>
      <ol>
        {deploys.map((deploy) => {
          const { label, icon: Icon } = deployStates[deploy.status];
          const failed = deploy.status === "failed";
          return (
            <li
              className="relative grid grid-cols-[20px_minmax(0,1fr)_auto] gap-3 pb-6 last:pb-0 not-last:before:absolute not-last:before:top-6.5 not-last:before:bottom-1 not-last:before:left-2.25 not-last:before:w-0.5 not-last:before:rounded-full not-last:before:bg-white/20"
              key={`${deploy.project}-${deploy.version}`}
            >
              <Icon
                className={`mt-px ${failed ? "text-ember" : "text-mist"} ${deploy.status === "running" ? "animate-deploying motion-reduce:animate-none" : ""}`}
                aria-hidden="true"
                size={20}
              />
              <div>
                <strong className="block font-semibold">{deploy.project}</strong>
                <span className="block text-meta text-ink-soft tabular-nums">
                  {deploy.version} · {deploy.environment}
                </span>
              </div>
              <div className="text-right">
                <strong className={`block font-semibold ${failed ? "text-ember" : ""}`}>{label}</strong>
                <span className="block text-meta text-ink-soft tabular-nums">{deploy.when}</span>
              </div>
            </li>
          );
        })}
      </ol>
    </Panel>
  );
}
