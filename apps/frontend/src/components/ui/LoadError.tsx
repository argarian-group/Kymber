import { EmptyState } from "./EmptyState";

export function LoadError({ title, message, onRetry }: { title: string; message: string; onRetry: () => void }) {
  return (
    <EmptyState
      title={title}
      description={message}
      action={
        <button className="btn" type="button" onClick={onRetry}>
          Try again
        </button>
      }
    />
  );
}
