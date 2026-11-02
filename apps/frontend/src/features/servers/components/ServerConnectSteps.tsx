import { useState } from "react";
import { CopyField } from "@/components/ui/CopyField";
import { Field } from "@/components/ui/Field";
import { ServerStatusBadge } from "@/features/servers/components/ServerStatusBadge";
import type { Server } from "@/features/servers/types";

const places = [
  { id: "this", label: "This machine" },
  { id: "remote", label: "Another machine" },
] as const;

export function ServerConnectSteps({ server, token, onDone }: { server: Server; token: string; onDone: () => void }) {
  const [place, setPlace] = useState<"this" | "remote">("this");
  const [apiUrl, setApiUrl] = useState(window.location.protocol === "https:" ? window.location.origin : "");
  const secureUrl = /^https:\/\/\S+$/.test(apiUrl);

  return (
    <section className="grid gap-4 rounded-panel bg-raised p-4" aria-label={`Connect ${server.name}`}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="section-title wrap-break-word">Connect {server.name}</h2>
        <ServerStatusBadge status={server.status} />
      </div>
      {server.connection_type === "ssh" ? <><p className="text-meta text-ink-soft">Install this generated public key in the remote user's ~/.ssh/authorized_keys.</p><CopyField label="Kymber SSH public key" value={token} /><p className="text-meta text-ink-soft">The host key was pinned during registration. Deployments will connect directly over SSH.</p></> : <p className="text-meta text-ink-soft">The token is shown only once, so copy the command now.</p>}
      {server.connection_type === "local" && <fieldset className="flex flex-wrap gap-2">
        <legend className="mb-1 text-meta font-medium">Where will it run?</legend>
        {places.map(({ id, label }) => (
          <label className="cursor-pointer" key={id}>
            <input className="peer sr-only" type="radio" name={`place-${server.id}`} checked={place === id} onChange={() => setPlace(id)} />
            <span className={`block rounded-control border-2 px-3 py-1.5 text-meta font-medium transition-colors duration-100 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-seafoam ${place === id ? "border-seafoam" : "border-line-strong hover:border-ink-faint"}`}>{label}</span>
          </label>
        ))}
      </fieldset>}
      {server.connection_type === "ssh" ? <button className="btn btn-primary" type="button" onClick={onDone}>Done</button> : place === "this" ? (
        <CopyField label="Run in the project folder and keep it open" value={`KYMBER_SERVER_TOKEN=${token} make server`} secret />
      ) : (
        <>
          <p className="text-meta text-ink-soft">The machine needs Docker and must be able to reach Kymber over HTTPS.</p>
          <Field label="Kymber address" type="url" placeholder="https://kymber.example.com" hint="Must start with https://." value={apiUrl} onChange={(event) => setApiUrl(event.target.value.trim())} />
          <CopyField label="1. Compile here, in the project folder" value="make build-server" />
          <CopyField label="2. Copy to the machine" value="apps/backend/build/kymber-server" />
          {secureUrl && <CopyField label="3. Run there and keep it open" value={`KYMBER_API_URL=${apiUrl} KYMBER_SERVER_TOKEN=${token} ./kymber-server`} secret />}
        </>
      )}
      <div>
        <button className={`btn ${server.status === "online" ? "btn-primary" : ""}`} type="button" onClick={onDone}>
          {server.status === "online" ? "Done" : "Finish later"}
        </button>
      </div>
    </section>
  );
}
