import { useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { LoadError } from "@/components/ui/LoadError";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";

export function SystemSettings() {
  const [enabled, setEnabled] = useState<boolean | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const { pending, error, run } = useAction();

  useLoadEffect(
    () => api<{ registration_enabled: boolean }>("/settings/registration"),
    (response) => setEnabled(response.registration_enabled),
    setLoadError,
    [attempt],
  );

  function toggle(next: boolean) {
    return run(async () => {
      const response = await api<{ registration_enabled: boolean }>("/settings/registration", { method: "PUT", body: { enabled: next } });
      setEnabled(response.registration_enabled);
    });
  }

  if (loadError) {
    return (
      <LoadError
        title="Could not load system settings"
        message={loadError}
        onRetry={() => {
          setLoadError(null);
          setAttempt((value) => value + 1);
        }}
      />
    );
  }

  return (
    <>
      <SettingsSection title="Access">
        <label className="flex cursor-pointer items-center justify-between gap-6 p-4">
          <span>
            <span className="block font-medium">Allow new registrations</span>
            <span className="mt-0.5 block text-meta text-ink-soft">When this is off, new accounts cannot be created. Existing accounts keep working.</span>
          </span>
          <span className="relative flex-none">
            <input
              className="peer sr-only"
              type="checkbox"
              role="switch"
              checked={enabled ?? false}
              disabled={enabled === null || pending}
              onChange={(event) => toggle(event.target.checked)}
            />
            <span
              className="block h-6 w-10 rounded-full bg-white/20 transition-colors duration-150 peer-checked:bg-seafoam peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-seafoam peer-disabled:opacity-60 after:absolute after:top-0.5 after:left-0.5 after:size-5 after:rounded-full after:bg-ink after:transition-transform after:duration-150 peer-checked:after:translate-x-4 motion-reduce:after:transition-none"
              aria-hidden="true"
            />
          </span>
        </label>
        {error && (
          <div className="border-t border-line px-4 py-3">
            <FormError message={error} />
          </div>
        )}
      </SettingsSection>
    </>
  );
}
