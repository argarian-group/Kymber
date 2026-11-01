import { type ReactNode } from "react";
import { FormError } from "@/components/ui/FormError";
import { SavedMessage } from "@/features/settings/components/SavedMessage";

export function FormFooter({ error, saved, children }: { error: string | null; saved: string | null; children: ReactNode }) {
  return (
    <div className="flex flex-wrap items-center justify-end gap-x-4 gap-y-2 border-t border-line px-4 py-3">
      <div className="mr-auto min-w-0">
        <FormError message={error} />
        {!error && <SavedMessage message={saved} />}
      </div>
      {children}
    </div>
  );
}
