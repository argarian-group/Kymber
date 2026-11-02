import { Columns2, Heading, Rows2, Trash2 } from "lucide-react";
import { type Editor } from "@tiptap/react";
import { t } from "@/lib/i18n";

export function RichTextTableTools({ editor }: { editor: Editor }) {
  const chain = () => editor.chain().focus();
  const actions = [
    { label: t("Add column"), icon: Columns2, run: () => chain().addColumnAfter().run() },
    { label: t("Add row"), icon: Rows2, run: () => chain().addRowAfter().run() },
    { label: t("Delete column"), icon: Columns2, run: () => chain().deleteColumn().run() },
    { label: t("Delete row"), icon: Rows2, run: () => chain().deleteRow().run() },
    { label: t("Toggle header"), icon: Heading, run: () => chain().toggleHeaderRow().run() },
    { label: t("Delete table"), icon: Trash2, run: () => chain().deleteTable().run() },
  ];
  return (
    <div className="flex flex-wrap gap-1 border-t border-line p-1.5" role="group" aria-label={t("Table")}>
      {actions.map(({ label, icon: Icon, run }) => (
        <button className="btn min-h-7 border-transparent text-xs" type="button" onMouseDown={(event) => event.preventDefault()} onClick={run} key={label}>
          <Icon aria-hidden="true" size={13} />
          {label}
        </button>
      ))}
    </div>
  );
}
