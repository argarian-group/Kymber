import { type Editor } from "@tiptap/react";
import { palette } from "@/lib/palette";

export function RichTextColors({ editor, onClose }: { editor: Editor; onClose: () => void }) {
  return (
    <div className="flex flex-wrap items-center gap-1 border-t border-line p-2" role="group" aria-label="Text color">
      {Object.values(palette).map((color) => (
        <button
          className="focus-ring size-7 cursor-pointer rounded-sm"
          style={{ backgroundColor: color }}
          type="button"
          aria-label={`Text color ${color}`}
          onMouseDown={(event) => event.preventDefault()}
          onClick={() => {
            editor.chain().focus().setColor(color).run();
            onClose();
          }}
          key={color}
        />
      ))}
      <button
        className="btn ml-1"
        type="button"
        onMouseDown={(event) => event.preventDefault()}
        onClick={() => {
          editor.chain().focus().unsetColor().run();
          onClose();
        }}
      >
        Default color
      </button>
    </div>
  );
}
