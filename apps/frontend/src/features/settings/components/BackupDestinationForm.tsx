import { useState, type FormEvent } from "react";
import { Field } from "@/components/ui/Field";
import { LoadError } from "@/components/ui/LoadError";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

type Destination = { endpoint: string; region: string; bucket: string; prefix: string; access_key: string };
const empty: Destination = { endpoint: "", region: "", bucket: "", prefix: "", access_key: "" };

export function BackupDestinationForm() {
  const [form, setForm] = useState(empty);
  const [secretKey, setSecretKey] = useState("");
  const [hasSecret, setHasSecret] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [saved, setSaved] = useState<string | null>(null);
  const { pending, error, run } = useAction();

  useLoadEffect(
    () => api<{ destination: Destination; has_secret: boolean }>("/settings/backup-destination"),
    ({ destination, has_secret }) => {
      setForm(destination);
      setHasSecret(has_secret);
      setLoadError(null);
    },
    setLoadError,
    [attempt],
  );

  function save(body: Destination & { secret_key: string }, message: string) {
    setSaved(null);
    return run(async () => {
      await api("/settings/backup-destination", { method: "PUT", body });
      setHasSecret(Boolean(body.bucket));
      if (!body.bucket) setForm(empty);
      setSecretKey("");
      setSaved(message);
    });
  }

  function submit(event: FormEvent) {
    event.preventDefault();
    void save({ ...form, secret_key: secretKey }, t("Connected. New backups are also sent to this bucket."));
  }

  if (loadError) return <LoadError title={t("Could not load the backup destination")} message={loadError} onRetry={() => setAttempt((value) => value + 1)} />;

  const set = (key: keyof Destination) => (event: { target: { value: string } }) => setForm({ ...form, [key]: event.target.value });

  return (
    <form onSubmit={submit}>
      <div className="grid gap-4 p-4 sm:grid-cols-2">
        <Field label={t("Endpoint")} required placeholder="s3.amazonaws.com" autoCapitalize="none" spellCheck={false} value={form.endpoint} onChange={set("endpoint")} hint={t("AWS, R2, B2, MinIO…")} />
        <Field label={t("Bucket")} required placeholder="kymber-backups" autoCapitalize="none" spellCheck={false} value={form.bucket} onChange={set("bucket")} />
        <Field label={t("Region")} placeholder="us-east-1" autoCapitalize="none" spellCheck={false} value={form.region} onChange={set("region")} />
        <Field label={t("Folder in the bucket")} placeholder="kymber" autoCapitalize="none" spellCheck={false} value={form.prefix} onChange={set("prefix")} />
        <Field label={t("Access key")} required autoCapitalize="none" spellCheck={false} autoComplete="off" value={form.access_key} onChange={set("access_key")} />
        <Field label={t("Secret key")} type="password" required={!hasSecret} autoComplete="new-password" placeholder={hasSecret ? t("Unchanged") : ""} value={secretKey} onChange={(event) => setSecretKey(event.target.value)} />
      </div>
      <FormFooter error={error} saved={saved}>
        {hasSecret && <button className="btn" type="button" disabled={pending} onClick={() => { if (window.confirm(t("Stop sending backups to S3? Copies already in the bucket stay there."))) void save({ ...empty, secret_key: "" }, t("Backups are kept only on the servers.")); }}>{t("Disconnect")}</button>}
        <button className="btn btn-primary" type="submit" disabled={pending}>{pending ? t("Testing…") : t("Test and save")}</button>
      </FormFooter>
    </form>
  );
}
