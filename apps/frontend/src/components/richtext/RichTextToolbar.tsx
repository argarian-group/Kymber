import { type Editor, useEditorState } from "@tiptap/react";
import { AlignCenter, AlignLeft, AlignRight, Bold, Braces, Code, Highlighter, ImagePlus, Italic, Link2, List, ListChecks, ListOrdered, type LucideIcon, Minus, Palette, Quote, Redo2, RemoveFormatting, Strikethrough, Table2, Underline, Undo2 } from "lucide-react";
import { useState } from "react";
import { RichTextColors } from "@/components/richtext/RichTextColors";
import { RichTextTableTools } from "@/components/richtext/RichTextTableTools";
import { RichTextUrlForm } from "@/components/richtext/RichTextUrlForm";
import { ToolbarButton } from "@/components/richtext/ToolbarButton";
import { t } from "@/lib/i18n";

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
      { icon: Bold, label: t("Bold"), active: state.bold, run: () => chain().toggleBold().run() },
      { icon: Italic, label: t("Italic"), active: state.italic, run: () => chain().toggleItalic().run() },
      { icon: Underline, label: t("Underline"), active: state.underline, run: () => chain().toggleUnderline().run() },
      { icon: Strikethrough, label: t("Strikethrough"), active: state.strike, run: () => chain().toggleStrike().run() },
      { icon: Code, label: t("Inline code"), active: state.code, run: () => chain().toggleCode().run() },
    ],
    [
      { icon: Palette, label: t("Text color"), active: state.color || panel === "color", run: () => togglePanel("color") },
      { icon: Highlighter, label: t("Highlight"), active: state.highlight, run: () => chain().toggleHighlight().run() },
      { icon: RemoveFormatting, label: t("Clear formatting"), run: () => chain().unsetAllMarks().clearNodes().run() },
    ],
    [
      { icon: AlignLeft, label: t("Align left"), active: state.left, run: () => chain().setTextAlign("left").run() },
      { icon: AlignCenter, label: t("Align center"), active: state.center, run: () => chain().setTextAlign("center").run() },
      { icon: AlignRight, label: t("Align right"), active: state.right, run: () => chain().setTextAlign("right").run() },
    ],
    [
      { icon: List, label: t("Bulleted list"), active: state.bulletList, run: () => chain().toggleBulletList().run() },
      { icon: ListOrdered, label: t("Numbered list"), active: state.orderedList, run: () => chain().toggleOrderedList().run() },
      { icon: ListChecks, label: t("Task list"), active: state.taskList, run: () => chain().toggleTaskList().run() },
    ],
    [
      { icon: Quote, label: t("Quote"), active: state.blockquote, run: () => chain().toggleBlockquote().run() },
      { icon: Braces, label: t("Code block"), active: state.codeBlock, run: () => chain().toggleCodeBlock().run() },
      { icon: Minus, label: t("Divider"), run: () => chain().setHorizontalRule().run() },
    ],
    [
      { icon: Link2, label: t("Link"), active: state.link || panel === "link", run: () => togglePanel("link") },
      { icon: ImagePlus, label: t("Image from URL"), active: panel === "image", run: () => togglePanel("image") },
      { icon: Table2, label: t("Insert table"), active: state.table, disabled: state.table, run: () => chain().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run() },
    ],
    [
      { icon: Undo2, label: t("Undo"), disabled: !state.canUndo, run: () => chain().undo().run() },
      { icon: Redo2, label: t("Redo"), disabled: !state.canRedo, run: () => chain().redo().run() },
    ],
  ];

  return (
    <div className="sticky top-0 z-10 rounded-t-control border-b border-line bg-sunken">
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 p-1.5" role="toolbar" aria-label={t("Formatting")}>
        <select
          className="focus-ring min-h-9 cursor-pointer rounded-sm bg-transparent px-1 text-meta md:min-h-7"
          aria-label={t("Text style")}
          value={state.block}
          onChange={(event) => {
            const level = Number(event.target.value) as 0 | 1 | 2 | 3;
            if (level === 0) chain().setParagraph().run();
            else chain().setHeading({ level }).run();
          }}
        >
          <option value="0">{t("Paragraph")}</option>
          <option value="1">{t("Heading {level}", { level: 1 })}</option>
          <option value="2">{t("Heading {level}", { level: 2 })}</option>
          <option value="3">{t("Heading {level}", { level: 3 })}</option>
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
          label={t("Link URL")}
          initial={editor.getAttributes("link").href ?? ""}
          protocols={["https:", "http:", "mailto:"]}
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
          label={t("Image URL")}
          initial=""
          protocols={["https:"]}
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
