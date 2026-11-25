import { useState, type FormEvent } from "react";
import { CopyField } from "@/components/ui/CopyField";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { api } from "@/lib/api";
import { formatDate, formatRelative } from "@/lib/format";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

type ApiToken = { id: string; name: string; created_at: string; last_used_at: string | null };

export function ApiTokens() {
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [secret, setSecret] = useState<string | null>(null);
  const { pending, error, run } = useAction();

  useLoadEffect(() => api<{ tokens: ApiToken[] }>("/account/tokens"), (response) => setTokens(response.tokens), setLoadError, []);

  async function create(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      const created = await api<{ token: ApiToken; secret: string }>("/account/tokens", { method: "POST", body: { name } });
      setTokens([created.token, ...tokens]);
      setSecret(created.secret);
      setName("");
    });
  }

  async function revoke(token: ApiToken) {
    if (!window.confirm(t("Revoke {name}? Scripts using it stop working.", { name: token.name }))) return;
    await run(async () => {
      await api(`/account/tokens/${token.id}`, { method: "DELETE" });
      setTokens(tokens.filter(({ id }) => id !== token.id));
    });
  }

  return (
    <SettingsSection title={t("API tokens")} description={t("For the kymber CLI and scripts. A token can do everything your account can.")}>
      {loadError && <div className="p-4"><FormError message={loadError} /></div>}
      {secret && (
        <div className="grid gap-2 border-b border-line p-4">
          <CopyField label={t("New token")} value={secret} />
          <p className="text-xs text-ink-soft">{t("Copy it now. It will not be shown again.")}</p>
        </div>
      )}
      {tokens.length > 0 && (
        <ul className="divide-y divide-line border-b border-line">
          {tokens.map((token) => (
            <li className="flex flex-wrap items-center gap-x-4 gap-y-1 px-4 py-3" key={token.id}>
              <div className="min-w-0 flex-1">
                <p className="text-meta font-medium wrap-break-word">{token.name}</p>
                <p className="text-xs text-ink-soft">
                  {t("Created {date}", { date: formatDate(token.created_at) })} · {token.last_used_at ? t("Last used {when}", { when: formatRelative(token.last_used_at) }) : t("Never used")}
                </p>
              </div>
              <button className="btn" type="button" disabled={pending} onClick={() => void revoke(token)}>{t("Revoke")}</button>
            </li>
          ))}
        </ul>
      )}
      <form onSubmit={create}>
        <div className="p-4">
          <Field label={t("Token name")} required maxLength={80} placeholder="GitHub Actions" value={name} onChange={(event) => setName(event.target.value)} />
        </div>
        <FormFooter error={error} saved={null}>
          <button className="btn btn-primary" type="submit" disabled={pending || !name.trim()}>{pending ? t("Creating…") : t("Create token")}</button>
        </FormFooter>
      </form>
    </SettingsSection>
  );
}
