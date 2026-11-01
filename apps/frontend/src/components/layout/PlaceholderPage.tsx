import { Link } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { PageHeader } from "./PageHeader";

export function PlaceholderPage({ title }: { title: string }) {
  return (
    <>
      <PageHeader title={title} />
      <EmptyState
        title={`${title} is not built yet`}
        action={
          <Link className="btn" to="/">
            Go to overview
          </Link>
        }
      />
    </>
  );
}
