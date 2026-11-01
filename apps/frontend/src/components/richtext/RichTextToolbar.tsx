import { type Editor, useEditorState } from "@tiptap/react";
import { AlignCenter, AlignLeft, AlignRight, Bold, Braces, Code, Highlighter, ImagePlus, Italic, Link2, List, ListChecks, ListOrdered, type LucideIcon, Minus, Palette, Quote, Redo2, RemoveFormatting, Strikethrough, Table2, Underline, Undo2 } from "lucide-react";
import { useState } from "react";
import { RichTextColors } from "@/components/richtext/RichTextColors";
import { RichTextTableTools } from "@/components/richtext/RichTextTableTools";
import { RichTextUrlForm } from "@/components/richtext/RichTextUrlForm";
import { ToolbarButton } from "@/components/richtext/ToolbarButton";

type Panel = "link" | "image" | "color" | null;
type Action = { icon: LucideIcon; label: string; run: () => void; active?: boolean; disabled?: boolean };

export function RichTextToolbar({ editor }: { editor: Editor }) {
  const [panel, setPanel] = useState<Panel>(null);
  const state = useEditorState({
    editor,
    selector: ({ editor: current }) => ({
      block: current.isActive("heading", { level: 1 }) ? "1" : current.isActive("heading", { level: 2 }) ? "2" : current.isActive("heading", { level: 3 }) ? "3" : "0",
      bold: current.isActive("bold"),
      italic: current.isActive("italic"),
      underline: current.isActive("underline"),
      strike: current.isActive("strike"),
      code: current.isActive("code"),
      highlight: current.isActive("highlight"),
      color: Boolean(current.getAttributes("textStyle").color),
      link: current.isActive("link"),
      left: current.isActive({ textAlign: "left" }),
      center: current.isActive({ textAlign: "center" }),
      right: current.isActive({ textAlign: "right" }),
      bulletList: current.isActive("bulletList"),
      orderedList: current.isActive("orderedList"),
      taskList: current.isActive("taskList"),
      blockquote: current.isActive("blockquote"),
      codeBlock: current.isActive("codeBlock"),
      table: current.isActive("table"),
      canUndo: current.can().undo(),
      canRedo: current.can().redo(),
    }),
  });
  const chain = () => editor.chain().focus();
  const togglePanel = (next: Panel) => setPanel((current) => (current === next ? null : next));

  const groups: Action[][] = [
    [
      { icon: Bold, label: "Bold", active: state.bold, run: () => chain().toggleBold().run() },
      { icon: Italic, label: "Italic", active: state.italic, run: () => chain().toggleItalic().run() },
      { icon: Underline, label: "Underline", active: state.underline, run: () => chain().toggleUnderline().run() },
      { icon: Strikethrough, label: "Strikethrough", active: state.strike, run: () => chain().toggleStrike().run() },
      { icon: Code, label: "Inline code", active: state.code, run: () => chain().toggleCode().run() },
    ],
    [
      { icon: Palette, label: "Text color", active: state.color || panel === "color", run: () => togglePanel("color") },
      { icon: Highlighter, label: "Highlight", active: state.highlight, run: () => chain().toggleHighlight().run() },
      { icon: RemoveFormatting, label: "Clear formatting", run: () => chain().unsetAllMarks().clearNodes().run() },
    ],
    [
      { icon: AlignLeft, label: "Align left", active: state.left, run: () => chain().setTextAlign("left").run() },
      { icon: AlignCenter, label: "Align center", active: state.center, run: () => chain().setTextAlign("center").run() },
      { icon: AlignRight, label: "Align right", active: state.right, run: () => chain().setTextAlign("right").run() },
    ],
    [
      { icon: List, label: "Bulleted list", active: state.bulletList, run: () => chain().toggleBulletList().run() },
      { icon: ListOrdered, label: "Numbered list", active: state.orderedList, run: () => chain().toggleOrderedList().run() },
      { icon: ListChecks, label: "Task list", active: state.taskList, run: () => chain().toggleTaskList().run() },
    ],
    [
      { icon: Quote, label: "Quote", active: state.blockquote, run: () => chain().toggleBlockquote().run() },
      { icon: Braces, label: "Code block", active: state.codeBlock, run: () => chain().toggleCodeBlock().run() },
      { icon: Minus, label: "Divider", run: () => chain().setHorizontalRule().run() },
    ],
    [
      { icon: Link2, label: "Link", active: state.link || panel === "link", run: () => togglePanel("link") },
      { icon: ImagePlus, label: "Image from URL", active: panel === "image", run: () => togglePanel("image") },
      { icon: Table2, label: "Insert table", active: state.table, disabled: state.table, run: () => chain().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run() },
    ],
    [
      { icon: Undo2, label: "Undo", disabled: !state.canUndo, run: () => chain().undo().run() },
      { icon: Redo2, label: "Redo", disabled: !state.canRedo, run: () => chain().redo().run() },
    ],
  ];

  return (
    <div className="sticky top-0 z-10 rounded-t-control border-b border-line bg-sunken">
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 p-1.5" role="toolbar" aria-label="Formatting">
        <select
          className="focus-ring min-h-9 cursor-pointer rounded-sm bg-transparent px-1 text-meta md:min-h-7"
          aria-label="Text style"
          value={state.block}
          onChange={(event) => {
            const level = Number(event.target.value) as 0 | 1 | 2 | 3;
            if (level === 0) chain().setParagraph().run();
            else chain().setHeading({ level }).run();
          }}
        >
          <option value="0">Paragraph</option>
          <option value="1">Heading 1</option>
          <option value="2">Heading 2</option>
          <option value="3">Heading 3</option>
        </select>
        {groups.map((group) => (
          <span className="flex" key={group[0].label}>
            {group.map(({ icon, label, active, disabled, run }) => (
              <ToolbarButton icon={icon} label={label} active={active} disabled={disabled} onClick={run} key={label} />
            ))}
          </span>
        ))}
      </div>
      {panel === "color" && <RichTextColors editor={editor} onClose={() => setPanel(null)} />}
      {panel === "link" && (
        <RichTextUrlForm
          label="Link URL"
          initial={editor.getAttributes("link").href ?? ""}
          protocols="https://|http://|mailto:"
          onSubmit={(href) => {
            if (editor.state.selection.empty && !state.link) chain().insertContent({ type: "text", text: href, marks: [{ type: "link", attrs: { href } }] }).run();
            else chain().extendMarkRange("link").setLink({ href }).run();
            setPanel(null);
          }}
          onRemove={state.link ? () => { chain().extendMarkRange("link").unsetLink().run(); setPanel(null); } : undefined}
          onClose={() => setPanel(null)}
        />
      )}
      {panel === "image" && (
        <RichTextUrlForm
          label="Image URL"
          initial=""
          protocols="https://"
          onSubmit={(src) => {
            chain().setImage({ src }).run();
            setPanel(null);
          }}
          onClose={() => setPanel(null)}
        />
      )}
      {state.table && <RichTextTableTools editor={editor} />}
    </div>
  );
}
