import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import "@xterm/xterm/css/xterm.css";
import { useEffect, useRef, useState } from "react";

type Status = { state: "connecting" } | { state: "connected" } | { state: "closed"; message: string };

export default function TerminalView({ path, label }: { path: string; label: string }) {
  const host = useRef<HTMLDivElement>(null);
  const [status, setStatus] = useState<Status>({ state: "connecting" });
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    setStatus({ state: "connecting" });
    const styles = getComputedStyle(document.documentElement);
    const terminal = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
      scrollback: 5000,
      theme: { background: styles.getPropertyValue("--color-canvas").trim(), foreground: styles.getPropertyValue("--color-ink").trim(), cursor: styles.getPropertyValue("--color-ink").trim() },
    });
    const fit = new FitAddon();
    terminal.loadAddon(fit);
    terminal.open(host.current!);
    fit.fit();
    terminal.focus();

    const scheme = window.location.protocol === "https:" ? "wss" : "ws";
    const socket = new WebSocket(`${scheme}://${window.location.host}/api${path}`);
    socket.binaryType = "arraybuffer";
    const sendSize = () => {
      if (socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ cols: terminal.cols, rows: terminal.rows }));
    };
    const encoder = new TextEncoder();
    const input = terminal.onData((data) => {
      if (socket.readyState === WebSocket.OPEN) socket.send(encoder.encode(data));
    });
    const resized = terminal.onResize(sendSize);
    let opened = false;
    socket.onopen = () => {
      opened = true;
      setStatus({ state: "connected" });
      sendSize();
    };
    socket.onmessage = (event) => terminal.write(new Uint8Array(event.data as ArrayBuffer));
    socket.onclose = (event) => setStatus({ state: "closed", message: event.reason || (opened ? "The connection was lost." : "Could not connect. Check your access and that you have fewer than 5 terminals open.") });
    const observer = new ResizeObserver(() => fit.fit());
    observer.observe(host.current!);

    return () => {
      observer.disconnect();
      input.dispose();
      resized.dispose();
      socket.onclose = null;
      socket.close();
      terminal.dispose();
    };
  }, [path, attempt]);

  return (
    <div className="grid gap-2">
      <div className="flex min-h-9 flex-wrap items-center justify-between gap-2">
        <p className={`text-meta ${status.state === "closed" ? "text-ember" : "text-ink-soft"}`} role="status">
          {status.state === "connecting" ? "Connecting…" : status.state === "connected" ? "Connected" : status.message}
        </p>
        {status.state === "closed" && (
          <button className="btn" type="button" onClick={() => setAttempt((value) => value + 1)}>
            Reconnect
          </button>
        )}
      </div>
      <div className="h-[32rem] max-h-[70vh] overflow-hidden rounded-control bg-canvas p-2" ref={host} role="application" aria-label={label} />
    </div>
  );
}
