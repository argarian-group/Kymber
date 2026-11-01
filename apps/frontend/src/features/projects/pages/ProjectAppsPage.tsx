import { useSearchParams } from "react-router";
import { NodeSection } from "@/features/projects/components/apps/NodeSection";
import { sampleNodes } from "@/features/projects/sampleData";

export function ProjectAppsPage() {
  const [params, setParams] = useSearchParams();
  const selected = sampleNodes.find((node) => node.toLowerCase() === params.get("node"));
  const filters = [undefined, ...sampleNodes];

  return (
    <div className="grid min-w-0 gap-6 px-4 py-6 md:px-6">
      <div className="flex flex-wrap items-center justify-end gap-3">
        <div className="flex gap-0.5 rounded-control bg-sunken p-0.5" role="group" aria-label="Filter by node">
          {filters.map((node) => (
            <button
              className={`focus-ring min-h-9 cursor-pointer rounded-sm px-2.5 text-meta font-medium transition-colors duration-100 md:min-h-7 ${selected === node ? "bg-raised text-ink" : "text-ink-soft hover:text-ink"}`}
              type="button"
              aria-pressed={selected === node}
              onClick={() => setParams(node ? { node: node.toLowerCase() } : {}, { replace: true })}
              key={node ?? "all"}
            >
              {node ?? "All nodes"}
            </button>
          ))}
        </div>
      </div>
      {(selected ? [selected] : sampleNodes).map((node) => (
        <NodeSection node={node} key={node} />
      ))}
    </div>
  );
}
