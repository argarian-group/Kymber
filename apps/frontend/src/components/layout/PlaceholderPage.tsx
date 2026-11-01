import { Panel } from "@/components/ui/Panel";
import { PageHeader } from "./PageHeader";

export function PlaceholderPage({ title, description }: { title: string; description: string }) {
  return (
    <>
      <PageHeader title={title} description={description} />
      <Panel title="Coming soon">
        <p className="text-ink-soft">This page is not built yet.</p>
      </Panel>
    </>
  );
}
