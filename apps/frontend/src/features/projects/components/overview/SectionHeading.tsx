import { Link } from "react-router";
import { t } from "@/lib/i18n";

export function SectionHeading({ id, title, to }: { id: string; title: string; to?: string }) {
  return (
    <div className="mb-3 flex items-center justify-between gap-3">
      <h2 id={id} className="section-title">
        {title}
      </h2>
      {to && (
        <Link className="focus-ring rounded-sm text-meta text-ink-soft hover:text-ink" to={to}>
          {t("View all")}
        </Link>
      )}
    </div>
  );
}
