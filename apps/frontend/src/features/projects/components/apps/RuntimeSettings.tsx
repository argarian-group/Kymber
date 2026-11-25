import { useState, type FormEvent } from "react";
import { Field } from "@/components/ui/Field";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import type { Service } from "@/features/projects/services";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function RuntimeSettings({ service, path, onChanged }: { service: Service; path: string; onChanged: () => void }) {
  const [cpu, setCpu] = useState(service.cpu_limit ? String(service.cpu_limit) : "");
  const [memory, setMemory] = useState(service.memory_limit_mb ? String(service.memory_limit_mb) : "");
  const [healthCmd, setHealthCmd] = useState(service.health_cmd ?? "");
  const [saved, setSaved] = useState<string | null>(null);
  const { pending, error, run } = useAction();
  const changed = Number(cpu || 0) !== (service.cpu_limit ?? 0) || Number(memory || 0) !== (service.memory_limit_mb ?? 0) || healthCmd.trim() !== (service.health_cmd ?? "");

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    const ok = await run(async () => {
      await api(`${path}/${service.id}/runtime`, { method: "PUT", body: { cpu_limit: Number(cpu || 0), memory_limit_mb: Number(memory || 0), health_cmd: healthCmd.trim() } });
    });
    if (ok) {
      setSaved(t("Saved. Applies on the next deploy."));
      onChanged();
    }
  }

  return (
    <AppSection title={t("Resources and health")} description={t("Leave a limit empty for no limit. Changes apply on the next deploy.")}>
      <form onSubmit={submit}>
        <div className="grid items-start gap-4 p-4 sm:grid-cols-2">
          <Field label={t("CPU limit (cores)")} type="number" min={0} max={256} step={0.01} inputMode="decimal" placeholder={t("No limit")} value={cpu} onChange={(event) => setCpu(event.target.value)} />
          <Field label={t("Memory limit (MB)")} type="number" min={6} max={1048576} step={1} inputMode="numeric" placeholder={t("No limit")} value={memory} onChange={(event) => setMemory(event.target.value)} />
          <div className="sm:col-span-2">
            <Field label={t("Health check command")} maxLength={512} placeholder="curl -f http://localhost:3000/health" hint={t("Runs inside the container. The deploy fails if it does not pass within 3 minutes.")} value={healthCmd} onChange={(event) => setHealthCmd(event.target.value)} />
          </div>
        </div>
        <FormFooter error={error} saved={saved}>
          <button className="btn btn-primary" type="submit" disabled={pending || !changed}>{pending ? t("Saving…") : t("Save")}</button>
        </FormFooter>
      </form>
    </AppSection>
  );
}
