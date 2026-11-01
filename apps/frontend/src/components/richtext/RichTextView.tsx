import { EditorContent, useEditor } from "@tiptap/react";
import { useEffect, useRef } from "react";
import { type Editor } from "@tiptap/react";
import { richTextExtensions, toEditorHTML } from "@/components/richtext/richText";

export function RichTextView({ html, onToggleTask }: { html: string; onToggleTask?: (html: string) => void }) {
  const content = toEditorHTML(html);
  const instance = useRef<Editor | null>(null);
  const editor = useEditor({
    extensions: richTextExtensions({
      openLinks: true,
      onReadOnlyTask: (node, checked) => {
        const current = instance.current;
        if (!current || !onToggleTask) return false;
        current.state.doc.descendants((candidate, position) => {
          if (candidate !== node) return true;
          current.view.dispatch(current.state.tr.setNodeMarkup(position, undefined, { ...candidate.attrs, checked }));
          return false;
        });
        return true;
      },
    }),
    content,
    editable: false,
    editorProps: { attributes: { class: "rich-text focus:outline-none" } },
    onUpdate: ({ editor: current }) => onToggleTask?.(current.getHTML()),
  });

  instance.current = editor;

  useEffect(() => {
    if (editor && editor.getHTML() !== content) editor.commands.setContent(content, { emitUpdate: false });
  }, [editor, content]);

  return <EditorContent editor={editor} />;
}
