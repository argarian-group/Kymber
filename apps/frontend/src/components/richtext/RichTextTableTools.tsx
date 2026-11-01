import { Columns2, Heading, Rows2, Trash2 } from "lucide-react";
import { type Editor } from "@tiptap/react";

export function RichTextTableTools({ editor }: { editor: Editor }) {
  const actions = [
    { label: "Add column", run: () => editor.chain().focus().addColumnAfter().run() },
    { label: "Add row", run: () => editor.chain().focus().addRowAfter().run() },
    { label: "Delete column", run: () => editor.chain().focus().deleteColumn().run() },
    { label: "Delete row", run: () => editor.chain().focus().deleteRow().run() },
    { label: "Toggle header", run: () => editor.chain().focus().toggleHeaderRow().run() },
    { label: "Delete table", run: () => editor.chain().focus().deleteTable().run() },
  ];
  const icons = [Columns2, Rows2, Columns2, Rows2, Heading, Trash2];
  return (
    <div className="flex flex-wrap gap-1 border-t border-line p-1.5" role="group" aria-label="Table">
      {actions.map(({ label, run }, index) => {
        const Icon = icons[index];
        return (
          <button className="btn min-h-7 border-transparent text-xs" type="button" onMouseDown={(event) => event.preventDefault()} onClick={run} key={label}>
            <Icon aria-hidden="true" size={13} />
            {label}
          </button>
        );
      })}
    </div>
  );
}
