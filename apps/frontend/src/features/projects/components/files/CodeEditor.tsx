import { oneDarkHighlightStyle } from "@codemirror/theme-one-dark";
import { syntaxHighlighting, LanguageDescription } from "@codemirror/language";
import { languages } from "@codemirror/language-data";
import { Compartment, EditorState } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { basicSetup } from "codemirror";
import { useEffect, useRef } from "react";

const theme = EditorView.theme(
  {
    "&": { height: "100%", backgroundColor: "var(--color-canvas)", color: "var(--color-ink)", fontSize: "13px" },
    "&.cm-focused": { outline: "none" },
    ".cm-scroller": { fontFamily: "var(--font-mono, monospace)", overflow: "auto" },
    ".cm-content": { caretColor: "var(--color-ink)" },
    ".cm-cursor": { borderLeftColor: "var(--color-ink)" },
    ".cm-gutters": { backgroundColor: "var(--color-canvas)", color: "var(--color-ink-faint)", border: "none" },
    ".cm-activeLine, .cm-activeLineGutter": { backgroundColor: "rgb(255 255 255 / 0.04)" },
    "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection": { backgroundColor: "rgb(71 203 183 / 0.3)" },
    ".cm-panels": { backgroundColor: "var(--color-raised)", color: "var(--color-ink)" },
  },
  { dark: true },
);

export default function CodeEditor({ filename, value, readOnly, label, onChange, onSave }: { filename: string; value: string; readOnly: boolean; label: string; onChange: (value: string) => void; onSave: () => void }) {
  const host = useRef<HTMLDivElement>(null);
  const callbacks = useRef({ onChange, onSave });
  callbacks.current = { onChange, onSave };

  // The parent remounts this per opened file, so the editor owns the text after the first render.
  useEffect(() => {
    const language = new Compartment();
    const view = new EditorView({
      parent: host.current!,
      state: EditorState.create({
        doc: value,
        extensions: [
          keymap.of([{ key: "Mod-s", preventDefault: true, run: () => (callbacks.current.onSave(), true) }]),
          basicSetup,
          language.of([]),
          syntaxHighlighting(oneDarkHighlightStyle),
          theme,
          EditorState.readOnly.of(readOnly),
          EditorView.contentAttributes.of({ "aria-label": label }),
          EditorView.updateListener.of((update) => {
            if (update.docChanged) callbacks.current.onChange(update.state.doc.toString());
          }),
        ],
      }),
    });
    void LanguageDescription.matchFilename(languages, filename)
      ?.load()
      .then((support) => {
        view.dispatch({ effects: language.reconfigure(support) });
      });
    return () => view.destroy();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return <div className="h-full min-h-0" ref={host} />;
}
