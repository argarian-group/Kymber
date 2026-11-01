import { CodeBlockLowlight } from "@tiptap/extension-code-block-lowlight";
import { Highlight } from "@tiptap/extension-highlight";
import { Image } from "@tiptap/extension-image";
import { TaskItem, type TaskItemOptions, TaskList } from "@tiptap/extension-list";
import { TableKit } from "@tiptap/extension-table";
import { TextAlign } from "@tiptap/extension-text-align";
import { Color, TextStyle } from "@tiptap/extension-text-style";
import { Placeholder } from "@tiptap/extensions";
import { StarterKit } from "@tiptap/starter-kit";
import { common, createLowlight } from "lowlight";

const lowlight = createLowlight(common);

type RichTextOptions = {
  placeholder?: string;
  openLinks?: boolean;
  onReadOnlyTask?: TaskItemOptions["onReadOnlyChecked"];
};

export function richTextExtensions({ placeholder = "", openLinks = false, onReadOnlyTask }: RichTextOptions) {
  return [
    StarterKit.configure({
      codeBlock: false,
      link: { openOnClick: openLinks, autolink: true, HTMLAttributes: { rel: "noopener noreferrer nofollow", target: "_blank" } },
    }),
    CodeBlockLowlight.configure({ lowlight }),
    TextStyle,
    Color,
    Highlight,
    TextAlign.configure({ types: ["heading", "paragraph"] }),
    TaskList,
    TaskItem.configure({ nested: true, onReadOnlyChecked: onReadOnlyTask ?? (() => false) }),
    TableKit.configure({ table: { resizable: false } }),
    Image,
    Placeholder.configure({ placeholder }),
  ];
}

function escapeHTML(text: string) {
  return text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

export function toEditorHTML(stored: string) {
  if (/^<(p|h[1-6]|ul|ol|blockquote|pre|table|hr|img)\b/.test(stored)) return stored;
  return stored
    .split("\n")
    .map((line) => `<p>${escapeHTML(line)}</p>`)
    .join("");
}
