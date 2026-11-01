import { Link } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { PageHeader } from "./PageHeader";

export function PlaceholderPage({ title, description }: { title: string; description: string }) {
  return (
    <>
      <PageHeader title={title} description={description} />
      <EmptyState
        title={`${title} is not built yet`}
        description="This page will appear here once it ships. The overview has the current state of the work."
        action={
          <Link className="btn" to="/">
            Go to overview
          </Link>
        }
      />
    </>
  );
}
