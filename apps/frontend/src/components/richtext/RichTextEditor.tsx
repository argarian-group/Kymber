import { EditorContent, useEditor } from "@tiptap/react";
import { richTextExtensions, toEditorHTML } from "@/components/richtext/richText";
import { RichTextToolbar } from "@/components/richtext/RichTextToolbar";

type RichTextEditorProps = {
  label: string;
  placeholder: string;
  initial: string;
  limit: number;
  size: string;
  autoFocus?: boolean;
  value: string;
  onChange: (html: string) => void;
};

export function RichTextEditor({ label, placeholder, initial, limit, size, autoFocus = false, value, onChange }: RichTextEditorProps) {
  const editor = useEditor({
    extensions: richTextExtensions({ placeholder }),
    content: toEditorHTML(initial),
    autofocus: autoFocus ? "end" : false,
    editorProps: { attributes: { class: `rich-text focus:outline-none px-3 py-2 ${size}`, "aria-label": label, role: "textbox", "aria-multiline": "true" } },
    onUpdate: ({ editor: current }) => onChange(current.isEmpty ? "" : current.getHTML()),
  });

  return (
    <div
      className="rounded-control border border-line-strong bg-sunken/60 focus-within:border-seafoam"
      onKeyDown={(event) => {
        if (event.key === "Escape") {
          event.preventDefault();
          event.stopPropagation();
        }
      }}
    >
      {editor && <RichTextToolbar editor={editor} />}
      <EditorContent editor={editor} />
      {value.length > limit && (
        <p className="border-t border-line px-3 py-1.5 text-xs text-ember" role="alert">
          This text is too long to save. Remove some content or formatting.
        </p>
      )}
    </div>
  );
}
