import { EmptyState } from "./EmptyState";
import { t } from "@/lib/i18n";

export function LoadError({ title, message, onRetry }: { title: string; message: string; onRetry: () => void }) {
  return (
    <EmptyState
      title={title}
      description={message}
      action={
        <button className="btn" type="button" onClick={onRetry}>
          {t("Try again")}
        </button>
      }
    />
  );
}
