import { useState } from "react";
import { isSafeURL } from "@/components/richtext/richText";

type RichTextUrlFormProps = {
  label: string;
  initial: string;
  protocols: string;
  onSubmit: (url: string) => void;
  onRemove?: () => void;
  onClose: () => void;
};

export function RichTextUrlForm({ label, initial, protocols, onSubmit, onRemove, onClose }: RichTextUrlFormProps) {
  const [url, setUrl] = useState(initial);
  const [invalid, setInvalid] = useState(false);

  function submit() {
    const value = url.trim();
    if (!isSafeURL(value) || !protocols.split("|").some((protocol) => value.startsWith(protocol))) {
      setInvalid(true);
      return;
    }
    onSubmit(value);
  }

  return (
    <div className="flex flex-wrap items-center gap-2 border-t border-line p-2">
      <input
        className="field min-w-48 flex-1"
        type="url"
        aria-label={label}
        placeholder="https://"
        autoFocus
        value={url}
        onChange={(event) => {
          setUrl(event.target.value);
          setInvalid(false);
        }}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            submit();
          }
          if (event.key === "Escape") {
            event.preventDefault();
            event.stopPropagation();
            onClose();
          }
        }}
      />
      <button className="btn btn-primary" type="button" onClick={submit}>
        Apply
      </button>
      {onRemove && (
        <button className="btn" type="button" onClick={onRemove}>
          Remove
        </button>
      )}
      <button className="btn" type="button" onClick={onClose}>
        Cancel
      </button>
      {invalid && (
        <p className="w-full text-xs text-ember" role="alert">
          Enter a full URL starting with {protocols.split("|").join(" or ")}
        </p>
      )}
    </div>
  );
}
